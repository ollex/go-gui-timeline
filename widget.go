package timeline

import (
	"fmt"
	"math"
	"time"

	gg "github.com/go-gui-org/go-gui/gui"
)

const (
	defaultWidth         = float32(960)
	defaultHeight        = float32(360)
	defaultResourceWidth = float32(160)
	defaultHeaderHeight  = float32(44)
	defaultLaneHeight    = float32(48)
	defaultEventInset    = float32(5)
	viewportStateNS      = "timeline.viewport-version"
	viewportStateCap     = 128
	sceneCacheNS         = "timeline.scene-cache"
	sceneCacheCap        = 128
	tooltipStateNS       = "timeline.tooltip"
	tooltipStateCap      = 128
)

// Config describes a timeline widget.
type Config struct {
	OnEventClick func(gg.EventCtx, Event)
	ID           string
	// ContentVersion is an optional application-managed revision for Resources
	// and Events. Zero enables automatic change detection. When non-zero,
	// increment it whenever either slice or any of their elements changes.
	ContentVersion  uint64
	ResourceHeader  string
	Resources       []Resource
	Events          []Event
	View            ViewSpec
	SelectedEventID EventID
	Width           float32
	Height          float32
	ContentWidth    float32
	ResourceWidth   float32
	HeaderHeight    float32
	LaneHeight      float32
	EventInset      float32
}

type timelineView struct {
	cfg Config
}

type viewportState struct {
	X                   float32
	Y                   float32
	ContentWidth        float32
	ResourceFingerprint uint64
	ViewKind            viewKind
	Initialized         bool
}

type viewportLimits struct {
	X float32
	Y float32
}

type eventTooltipState struct {
	EventID    EventID
	EventIndex int
	Visible    bool
}

type sceneCacheEntry struct {
	Scene       Scene
	Fingerprint uint64
	Explicit    bool
}

type canvasVersions struct {
	Body           uint64
	TimeHeader     uint64
	ResourceHeader uint64
	Corner         uint64
}

// GenerateLayout builds the timeline under its enclosing effective-ID scope.
func (v *timelineView) GenerateLayout(w *gg.Window) gg.Layout {
	return gg.GenerateViewLayout(buildWidget(w, v.cfg), w)
}

// New constructs a controlled timeline widget.
func New(_ *gg.Window, cfg Config) gg.View {
	gg.RequireID("Timeline", cfg.ID)
	applyConfigDefaults(&cfg)
	validateWidgetConfig(cfg)
	return &timelineView{cfg: cfg}
}

func buildWidget(w *gg.Window, cfg Config) gg.View {
	stateID := w.EffID(cfg.ID)
	scene := cachedWidgetScene(w, stateID, cfg)
	states := gg.StateMap[string, viewportState](w, viewportStateNS, viewportStateCap)
	scrollbarGutter := gg.CurrentTheme().ScrollbarStyle.Size + 4
	scrollAreaWidth := cfg.Width - cfg.ResourceWidth
	scrollAreaHeight := cfg.Height - cfg.HeaderHeight
	bodyWidth := scrollAreaWidth - scrollbarGutter
	bodyHeight := scrollAreaHeight - scrollbarGutter
	limits := widgetViewportLimits(scene, bodyWidth, bodyHeight)
	state := states.GetOr(stateID, viewportState{})
	resourceFingerprint := resourceIDsFingerprint(cfg.Resources)
	if !state.Initialized || state.ViewKind != cfg.View.kind {
		w.ScrollX().Set(stateID, 0)
		w.ScrollY().Set(stateID, 0)
		state = viewportState{
			ContentWidth:        cfg.ContentWidth,
			ResourceFingerprint: resourceFingerprint,
			ViewKind:            cfg.View.kind,
			Initialized:         true,
		}
		states.Set(stateID, state)
	} else if state.ResourceFingerprint != resourceFingerprint {
		w.ScrollY().Set(stateID, 0)
		state.ResourceFingerprint = resourceFingerprint
		states.Set(stateID, state)
	}
	viewportX := -w.ScrollX().GetOr(stateID, 0)
	if state.ContentWidth > 0 && state.ContentWidth != cfg.ContentWidth {
		centerRatio := (viewportX + bodyWidth/2) / state.ContentWidth
		centerRatio = max(0, min(1, centerRatio))
		viewportX = centerRatio*cfg.ContentWidth - bodyWidth/2
		state.ContentWidth = cfg.ContentWidth
		states.Set(stateID, state)
	}
	viewport := clampViewport(viewportState{
		X:                   viewportX,
		Y:                   -w.ScrollY().GetOr(stateID, 0),
		ContentWidth:        state.ContentWidth,
		ResourceFingerprint: state.ResourceFingerprint,
		ViewKind:            state.ViewKind,
		Initialized:         true,
	}, limits)
	w.ScrollX().Set(stateID, -viewport.X)
	w.ScrollY().Set(stateID, -viewport.Y)

	baseRenderer := newWidgetRenderer(cfg, scene, viewport)
	version := resolvedWidgetFingerprint(cfg, baseRenderer)
	versions := widgetCanvasVersions(version, viewport)
	tooltipStates := gg.StateMap[string, eventTooltipState](
		w, tooltipStateNS, tooltipStateCap,
	)
	tooltip := tooltipStates.GetOr(stateID, eventTooltipState{})
	if tooltip.Visible && !tooltipStillValid(
		tooltip, cfg, scene, w, baseRenderer.style.eventText,
	) {
		tooltip = eventTooltipState{}
		tooltipStates.Set(stateID, tooltip)
	}

	bodyCfg := cfg
	bodyCfg.Width = max(scene.Width, bodyWidth)
	bodyCfg.Height = max(scene.Height, bodyHeight)
	bodyCfg.ResourceWidth = 0
	bodyCfg.HeaderHeight = 0
	bodyRenderer := *baseRenderer
	bodyRenderer.cfg = bodyCfg
	bodyRenderer.viewport = viewportState{}
	bodyCanvas := gg.DrawCanvas(gg.DrawCanvasCfg{
		ID:      cfg.ID + "-body",
		Version: versions.Body,
		Width:   bodyCfg.Width,
		Height:  bodyCfg.Height,
		Padding: gg.NoPadding,
		Color:   baseRenderer.style.background,
		A11YCfg: gg.A11YCfg{
			A11YLabel:       "Timeline",
			A11YDescription: "A scrollable resource timeline with scheduled events.",
		},
		OnDraw: bodyRenderer.drawBody,
		OnClick: func(ctx gg.EventCtx) {
			handleWidgetClick(ctx, cfg, scene)
		},
		OnMouseMove: func(ctx gg.EventCtx) {
			handleWidgetMouseMove(
				ctx, cfg, scene, baseRenderer.style.eventText,
				tooltipStates, stateID,
			)
		},
		OnMouseLeave: func(ctx gg.EventCtx) {
			clearWidgetTooltip(ctx.Window, tooltipStates, stateID)
		},
	})

	timeCfg := cfg
	timeCfg.Width = bodyWidth
	timeCfg.ResourceWidth = 0
	timeRenderer := *baseRenderer
	timeRenderer.cfg = timeCfg
	timeHeader := gg.DrawCanvas(gg.DrawCanvasCfg{
		ID:      cfg.ID + "-time-header",
		Version: versions.TimeHeader,
		Width:   bodyWidth,
		Height:  cfg.HeaderHeight,
		Padding: gg.NoPadding,
		Color:   baseRenderer.style.header,
		Clip:    true,
		OnDraw:  timeRenderer.drawTimeHeader,
	})

	resourceCfg := cfg
	resourceCfg.Height = bodyHeight
	resourceCfg.HeaderHeight = 0
	resourceRenderer := *baseRenderer
	resourceRenderer.cfg = resourceCfg
	resourceHeader := gg.DrawCanvas(gg.DrawCanvasCfg{
		ID:      cfg.ID + "-resource-header",
		Version: versions.ResourceHeader,
		Width:   cfg.ResourceWidth,
		Height:  bodyHeight,
		Padding: gg.NoPadding,
		Color:   baseRenderer.style.background,
		Clip:    true,
		OnDraw:  resourceRenderer.drawResourceRows,
	})

	cornerRenderer := *baseRenderer
	corner := gg.DrawCanvas(gg.DrawCanvasCfg{
		ID:      cfg.ID + "-corner",
		Version: versions.Corner,
		Width:   cfg.ResourceWidth,
		Height:  cfg.HeaderHeight,
		Padding: gg.NoPadding,
		Color:   baseRenderer.style.header,
		OnDraw:  cornerRenderer.drawCorner,
	})

	scrollBody := gg.Canvas(gg.ContainerCfg{
		ID:         cfg.ID,
		Sizing:     gg.FixedFixed,
		Width:      scrollAreaWidth,
		Height:     scrollAreaHeight,
		Padding:    gg.NewPadding(0, scrollbarGutter, scrollbarGutter, 0),
		Scrollable: true,
		Overflow:   true,
		OnScroll: func(ctx gg.EventCtx) {
			clearWidgetTooltip(ctx.Window, tooltipStates, stateID)
		},
		ScrollbarCfgX: &gg.ScrollbarCfg{
			ID: cfg.ID + "-horizontal-scrollbar", Overflow: gg.ScrollbarAuto,
			GapEdge: gg.SomeF(2),
		},
		ScrollbarCfgY: &gg.ScrollbarCfg{
			ID: cfg.ID + "-vertical-scrollbar", Overflow: gg.ScrollbarAuto,
			GapEdge: gg.SomeF(2),
		},
		Content: []gg.View{bodyCanvas},
	})

	content := []gg.View{
		gg.Row(gg.ContainerCfg{
			Sizing:  gg.FixedFixed,
			Width:   cfg.Width,
			Height:  cfg.HeaderHeight,
			Padding: gg.NoPadding,
			Spacing: gg.SomeF(0),
			Content: []gg.View{
				corner,
				timeHeader,
				gg.Canvas(gg.ContainerCfg{
					Sizing: gg.FixedFixed,
					Width:  scrollbarGutter,
					Height: cfg.HeaderHeight,
					Color:  baseRenderer.style.header,
					Radius: gg.SomeF(0),
				}),
			},
		}),
		gg.Row(gg.ContainerCfg{
			Sizing:  gg.FixedFixed,
			Width:   cfg.Width,
			Height:  scrollAreaHeight,
			Padding: gg.NoPadding,
			Spacing: gg.SomeF(0),
			Content: []gg.View{
				gg.Column(gg.ContainerCfg{
					Sizing:  gg.FixedFixed,
					Width:   cfg.ResourceWidth,
					Height:  scrollAreaHeight,
					Padding: gg.NoPadding,
					Spacing: gg.SomeF(0),
					Content: []gg.View{
						resourceHeader,
						gg.Canvas(gg.ContainerCfg{
							Sizing: gg.FixedFixed,
							Width:  cfg.ResourceWidth,
							Height: scrollbarGutter,
							Color:  baseRenderer.style.background,
							Radius: gg.SomeF(0),
						}),
					},
				}),
				scrollBody,
			},
		}),
	}
	if tooltip.Visible {
		content = append(content, eventTooltipView(
			cfg, scene, viewport, tooltip, baseRenderer.style, bodyWidth, bodyHeight,
		))
	}

	return gg.Column(gg.ContainerCfg{
		Sizing:  gg.FixedFixed,
		Width:   cfg.Width,
		Height:  cfg.Height,
		Padding: gg.NoPadding,
		Spacing: gg.SomeF(0),
		Content: content,
	})
}

func handleWidgetClick(ctx gg.EventCtx, cfg Config, scene Scene) {
	if ctx.Event == nil || cfg.OnEventClick == nil {
		return
	}
	hit, ok := scene.HitTest(ctx.Event.MouseX, ctx.Event.MouseY)
	if !ok {
		return
	}
	ctx.Consume()
	cfg.OnEventClick(ctx, cfg.Events[hit.EventIndex])
}

func handleWidgetMouseMove(
	ctx gg.EventCtx,
	cfg Config,
	scene Scene,
	textStyle gg.TextStyle,
	states *gg.BoundedMap[string, eventTooltipState],
	stateID string,
) {
	if ctx.Event == nil {
		return
	}
	next := eventTooltipState{}
	hit, ok := scene.HitTest(ctx.Event.MouseX, ctx.Event.MouseY)
	if ok {
		layout, found := sceneEventLayout(scene, hit.EventIndex)
		event := cfg.Events[hit.EventIndex]
		available := layout.Rect.Width - 2*eventTextPadding
		if found && event.Title != "" &&
			ctx.Window.TextWidth(event.Title, textStyle) > available {
			next = eventTooltipState{
				EventID:    event.ID,
				EventIndex: hit.EventIndex,
				Visible:    true,
			}
		}
	}
	if states.GetOr(stateID, eventTooltipState{}) == next {
		return
	}
	states.Set(stateID, next)
	ctx.Window.InvalidateLayout()
}

func clearWidgetTooltip(
	w *gg.Window,
	states *gg.BoundedMap[string, eventTooltipState],
	stateID string,
) {
	if !states.GetOr(stateID, eventTooltipState{}).Visible {
		return
	}
	states.Set(stateID, eventTooltipState{})
	w.InvalidateLayout()
}

func tooltipStillValid(
	tooltip eventTooltipState,
	cfg Config,
	scene Scene,
	w *gg.Window,
	textStyle gg.TextStyle,
) bool {
	if !tooltip.Visible || tooltip.EventIndex < 0 || tooltip.EventIndex >= len(cfg.Events) {
		return false
	}
	if cfg.Events[tooltip.EventIndex].ID != tooltip.EventID {
		return false
	}
	layout, ok := sceneEventLayout(scene, tooltip.EventIndex)
	if !ok {
		return false
	}
	title := cfg.Events[tooltip.EventIndex].Title
	return title != "" &&
		w.TextWidth(title, textStyle) > layout.Rect.Width-2*eventTextPadding
}

func sceneEventLayout(scene Scene, eventIndex int) (EventLayout, bool) {
	if eventIndex < 0 || eventIndex >= len(scene.eventLocations) {
		return EventLayout{}, false
	}
	location := scene.eventLocations[eventIndex]
	if location.rowIndexPlusOne == 0 {
		return EventLayout{}, false
	}
	row := scene.Rows[location.rowIndexPlusOne-1]
	return row.Events[location.eventIndex], true
}

func eventTooltipView(
	cfg Config,
	scene Scene,
	viewport viewportState,
	tooltip eventTooltipState,
	style widgetStyle,
	bodyWidth float32,
	bodyHeight float32,
) gg.View {
	event := cfg.Events[tooltip.EventIndex]
	layout, _ := sceneEventLayout(scene, tooltip.EventIndex)
	visibleLeft := max(layout.Rect.X, viewport.X)
	visibleRight := min(layout.Rect.X+layout.Rect.Width, viewport.X+bodyWidth)
	visibleBottom := min(layout.Rect.Y+layout.Rect.Height, viewport.Y+bodyHeight)
	return gg.Tooltip(gg.TooltipCfg{
		ID:      cfg.ID + "-event-tooltip",
		Anchor:  gg.Some(gg.FloatTopLeft),
		OffsetX: gg.SomeF(cfg.ResourceWidth + (visibleLeft+visibleRight)/2 - viewport.X),
		OffsetY: gg.SomeF(cfg.HeaderHeight + visibleBottom - viewport.Y + 6),
		Content: []gg.View{
			gg.Text(gg.TextCfg{
				Text:      event.Title,
				TextStyle: style.resourceText,
				Mode:      gg.TextModeWrap,
			}),
		},
	})
}

func widgetViewportLimits(scene Scene, bodyWidth, bodyHeight float32) viewportLimits {
	return viewportLimits{
		X: max(0, scene.Width-bodyWidth),
		Y: max(0, scene.Height-bodyHeight),
	}
}

func clampViewport(viewport viewportState, limits viewportLimits) viewportState {
	if !finite(viewport.X) {
		viewport.X = 0
	}
	if !finite(viewport.Y) {
		viewport.Y = 0
	}
	viewport.X = max(0, min(limits.X, viewport.X))
	viewport.Y = max(0, min(limits.Y, viewport.Y))
	return viewport
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Width == 0 {
		cfg.Width = defaultWidth
	}
	if cfg.Height == 0 {
		cfg.Height = defaultHeight
	}
	if cfg.ResourceWidth == 0 {
		cfg.ResourceWidth = defaultResourceWidth
	}
	if cfg.HeaderHeight == 0 {
		cfg.HeaderHeight = defaultHeaderHeight
	}
	if cfg.ContentWidth == 0 {
		cfg.ContentWidth = cfg.Width - cfg.ResourceWidth
	}
	if cfg.LaneHeight == 0 {
		cfg.LaneHeight = defaultLaneHeight
	}
	if cfg.EventInset == 0 {
		cfg.EventInset = defaultEventInset
	}
	if cfg.ResourceHeader == "" {
		cfg.ResourceHeader = "Resource"
	}
}

func validateWidgetConfig(cfg Config) {
	if cfg.Width <= cfg.ResourceWidth {
		panic("timeline: width must be greater than resource width")
	}
	if cfg.Height <= cfg.HeaderHeight {
		panic("timeline: height must be greater than header height")
	}
	if cfg.ContentWidth <= 0 {
		panic("timeline: content width must be positive")
	}
	if cfg.HeaderHeight <= 0 {
		panic("timeline: header height must be positive")
	}
	if !cfg.View.Range.End.After(cfg.View.Range.Start) {
		panic("timeline: view range end must be after start")
	}
	for i, tick := range cfg.View.Ticks {
		if tick.At.Before(cfg.View.Range.Start) || tick.At.After(cfg.View.Range.End) {
			panic("timeline: tick must be inside view range")
		}
		if i > 0 && !tick.At.After(cfg.View.Ticks[i-1].At) {
			panic("timeline: ticks must be strictly increasing")
		}
	}
}

const (
	fnvOffset = uint64(14695981039346656037)
	fnvPrime  = uint64(1099511628211)
)

func resourceIDsFingerprint(resources []Resource) uint64 {
	hash := fnvOffset
	hashUint64(&hash, uint64(len(resources)))
	for _, resource := range resources {
		hashString(&hash, string(resource.ID))
	}
	return hash
}

func cachedWidgetScene(w *gg.Window, stateID string, cfg Config) Scene {
	fingerprint, explicit := widgetSceneFingerprint(cfg)
	caches := gg.StateMap[string, sceneCacheEntry](w, sceneCacheNS, sceneCacheCap)
	cached, cachedOK := caches.Get(stateID)
	if cachedOK &&
		cached.Fingerprint == fingerprint && cached.Explicit == explicit {
		return cached.Scene
	}

	var eventLocationCache []eventLocation
	if cachedOK {
		eventLocationCache = cached.Scene.eventLocations
	}
	scene, err := buildScene(cfg.Resources, cfg.Events, LayoutConfig{
		Range:      cfg.View.Range,
		Width:      cfg.ContentWidth,
		LaneHeight: cfg.LaneHeight,
		EventInset: cfg.EventInset,
	}, eventLocationCache)
	if err != nil {
		panic(fmt.Sprintf("timeline widget: %v", err))
	}
	caches.Set(stateID, sceneCacheEntry{
		Scene: scene, Fingerprint: fingerprint, Explicit: explicit,
	})
	return scene
}

func widgetSceneFingerprint(cfg Config) (uint64, bool) {
	hash := fnvOffset
	for _, value := range [...]float32{cfg.ContentWidth, cfg.LaneHeight, cfg.EventInset} {
		hashFloat32(&hash, value)
	}
	hashTime(&hash, cfg.View.Range.Start)
	hashTime(&hash, cfg.View.Range.End)
	if cfg.ContentVersion != 0 {
		hashUint64(&hash, cfg.ContentVersion)
		return hash, true
	}

	hashUint64(&hash, uint64(len(cfg.Resources)))
	for _, resource := range cfg.Resources {
		hashString(&hash, string(resource.ID))
	}
	hashUint64(&hash, uint64(len(cfg.Events)))
	for _, event := range cfg.Events {
		hashString(&hash, string(event.ID))
		hashString(&hash, string(event.ResourceID))
		hashTime(&hash, event.Start)
		hashTime(&hash, event.End)
	}
	return hash, false
}

func resolvedWidgetFingerprint(cfg Config, renderer *widgetRenderer) uint64 {
	if cfg.ContentVersion == 0 {
		return widgetFingerprint(cfg, renderer)
	}
	hash := widgetBaseFingerprint(cfg, renderer)
	hashUint64(&hash, cfg.ContentVersion)
	return hash
}

func widgetFingerprint(cfg Config, renderer *widgetRenderer) uint64 {
	hash := widgetBaseFingerprint(cfg, renderer)
	hashUint64(&hash, uint64(len(cfg.Resources)))
	for _, resource := range cfg.Resources {
		hashString(&hash, string(resource.ID))
		hashString(&hash, resource.Label)
	}
	hashUint64(&hash, uint64(len(cfg.Events)))
	for _, event := range cfg.Events {
		hashString(&hash, string(event.ID))
		hashString(&hash, string(event.ResourceID))
		hashTime(&hash, event.Start)
		hashTime(&hash, event.End)
		hashString(&hash, event.Title)
		hashColor(&hash, eventColor(event))
		hashColor(&hash, eventTextColor(event, renderer.style.eventText.Color))
	}
	return hash
}

func widgetBaseFingerprint(cfg Config, renderer *widgetRenderer) uint64 {
	hash := fnvOffset
	hashString(&hash, cfg.ResourceHeader)
	hashString(&hash, string(cfg.SelectedEventID))
	for _, value := range [...]float32{
		cfg.Width, cfg.Height, cfg.ContentWidth, cfg.ResourceWidth,
		cfg.HeaderHeight, cfg.LaneHeight, cfg.EventInset,
	} {
		hashFloat32(&hash, value)
	}
	hashTime(&hash, cfg.View.Range.Start)
	hashTime(&hash, cfg.View.Range.End)
	hashUint64(&hash, uint64(len(cfg.View.Ticks)))
	for _, tick := range cfg.View.Ticks {
		hashTime(&hash, tick.At)
		hashString(&hash, tick.Label)
	}
	hashWidgetStyle(&hash, renderer.style)
	return hash
}

func widgetVersion(hash uint64, viewport viewportState) uint64 {
	hash = widgetVersionX(hash, viewport.X)
	return widgetVersionY(hash, viewport.Y)
}

func widgetCanvasVersions(hash uint64, viewport viewportState) canvasVersions {
	return canvasVersions{
		Body:           hash,
		TimeHeader:     widgetVersionX(hash, viewport.X),
		ResourceHeader: widgetVersionY(hash, viewport.Y),
		Corner:         hash,
	}
}

func widgetVersionX(hash uint64, x float32) uint64 {
	hashFloat32(&hash, x)
	return hash
}

func widgetVersionY(hash uint64, y float32) uint64 {
	hashFloat32(&hash, y)
	return hash
}

func hashWidgetStyle(hash *uint64, style widgetStyle) {
	for _, color := range [...]gg.Color{
		style.resourceText.Color, style.timeText.Color, style.eventText.Color,
		style.background, style.header, style.row, style.rowAlt, style.grid,
		style.selection,
	} {
		hashColor(hash, color)
	}
	for _, text := range [...]gg.TextStyle{
		style.resourceText, style.timeText, style.eventText,
	} {
		hashString(hash, text.Family)
		hashUint64(hash, uint64(text.Typeface))
		hashFloat32(hash, text.Size)
		hashFloat32(hash, text.LineSpacing)
		hashFloat32(hash, text.LetterSpacing)
		hashFloat32(hash, text.RotationRadians)
		hashFloat32(hash, text.StrokeWidth)
		hashFloat32(hash, text.EmojiBoxWidth)
		hashFloat32(hash, text.CellWidth)
		hashFloat32(hash, text.CellHeight)
		hashColor(hash, text.BgColor)
		hashColor(hash, text.StrokeColor)
		hashUint64(hash, uint64(text.Align))
		for _, flag := range [...]bool{
			text.Underline, text.NoBuiltinBoxGlyphs, text.Strikethrough,
		} {
			hashBool(hash, flag)
		}
		if text.AffineTransform == nil {
			hashBool(hash, false)
		} else {
			hashBool(hash, true)
			for _, value := range [...]float32{
				text.AffineTransform.XX, text.AffineTransform.XY,
				text.AffineTransform.YX, text.AffineTransform.YY,
				text.AffineTransform.X0, text.AffineTransform.Y0,
			} {
				hashFloat32(hash, value)
			}
		}
		if text.Gradient == nil {
			hashBool(hash, false)
		} else {
			hashBool(hash, true)
			hashUint64(hash, uint64(text.Gradient.Direction))
			hashUint64(hash, uint64(len(text.Gradient.Stops)))
			for _, stop := range text.Gradient.Stops {
				hashUint64(hash, uint64(stop.Color.R)<<24|
					uint64(stop.Color.G)<<16|uint64(stop.Color.B)<<8|
					uint64(stop.Color.A))
				hashFloat32(hash, stop.Position)
			}
		}
		if text.Features == nil {
			hashBool(hash, false)
		} else {
			hashBool(hash, true)
			hashUint64(hash, uint64(len(text.Features.OpenTypeFeatures)))
			for _, feature := range text.Features.OpenTypeFeatures {
				hashString(hash, feature.Tag)
				hashUint64(hash, uint64(feature.Value))
			}
			hashUint64(hash, uint64(len(text.Features.VariationAxes)))
			for _, axis := range text.Features.VariationAxes {
				hashString(hash, axis.Tag)
				hashFloat32(hash, axis.Value)
			}
		}
	}
	hashFloat32(hash, style.radius)
}

func hashString(hash *uint64, value string) {
	hashUint64(hash, uint64(len(value)))
	for i := range len(value) {
		*hash ^= uint64(value[i])
		*hash *= fnvPrime
	}
}

func hashTime(hash *uint64, value time.Time) {
	hashUint64(hash, uint64(value.UnixNano()))
}

func hashFloat32(hash *uint64, value float32) {
	hashUint64(hash, uint64(math.Float32bits(value)))
}

func hashColor(hash *uint64, color gg.Color) {
	hashUint64(hash, uint64(uint32(color.RGBA8())))
	hashBool(hash, color.IsSet())
}

func hashBool(hash *uint64, value bool) {
	if value {
		hashUint64(hash, 1)
		return
	}
	hashUint64(hash, 0)
}

func hashUint64(hash *uint64, value uint64) {
	for range 8 {
		*hash ^= value & 0xff
		*hash *= fnvPrime
		value >>= 8
	}
}
