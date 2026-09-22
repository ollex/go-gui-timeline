package timeline

import (
	"math"
	"testing"
	"time"

	gg "github.com/go-gui-org/go-gui/gui"
)

func TestApplyConfigDefaults(t *testing.T) {
	t.Parallel()

	var cfg Config
	applyConfigDefaults(&cfg)
	if cfg.Width != defaultWidth || cfg.ResourceWidth != defaultResourceWidth {
		t.Errorf("width defaults = (%v, %v), want (%v, %v)",
			cfg.Width, cfg.ResourceWidth, defaultWidth, defaultResourceWidth)
	}
	if cfg.Height != defaultHeight || cfg.ContentWidth != defaultWidth-defaultResourceWidth {
		t.Errorf("viewport defaults = (%v, %v), want (%v, %v)",
			cfg.Height, cfg.ContentWidth, defaultHeight, defaultWidth-defaultResourceWidth)
	}
	if cfg.HeaderHeight != defaultHeaderHeight || cfg.LaneHeight != defaultLaneHeight {
		t.Errorf("height defaults = (%v, %v), want (%v, %v)",
			cfg.HeaderHeight, cfg.LaneHeight, defaultHeaderHeight, defaultLaneHeight)
	}
	if cfg.EventInset != defaultEventInset {
		t.Errorf("EventInset = %v, want %v", cfg.EventInset, defaultEventInset)
	}
	if cfg.ResourceHeader != "Resource" {
		t.Errorf("ResourceHeader = %q, want %q", cfg.ResourceHeader, "Resource")
	}
	if cfg.A11YLabel != "Timeline" {
		t.Errorf("A11YLabel = %q, want %q", cfg.A11YLabel, "Timeline")
	}
}

func TestApplyConfigDefaultsPreservesValues(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Width:          700,
		Height:         300,
		ContentWidth:   900,
		ResourceWidth:  120,
		HeaderHeight:   30,
		LaneHeight:     32,
		EventInset:     3,
		ResourceHeader: "Team",
	}
	want := cfg
	applyConfigDefaults(&cfg)
	if cfg.Width != want.Width || cfg.Height != want.Height ||
		cfg.ContentWidth != want.ContentWidth || cfg.ResourceWidth != want.ResourceWidth ||
		cfg.HeaderHeight != want.HeaderHeight || cfg.LaneHeight != want.LaneHeight ||
		cfg.EventInset != want.EventInset || cfg.ResourceHeader != want.ResourceHeader {
		t.Errorf("applyConfigDefaults changed explicit values: got %+v, want %+v", cfg, want)
	}
}

func TestWidgetVersionIncludesViewport(t *testing.T) {
	t.Parallel()

	base := widgetVersion(7, viewportState{})
	if base != widgetVersion(7, viewportState{}) {
		t.Error("widgetVersion is not deterministic")
	}
	scrolled := widgetVersion(7, viewportState{X: 10, Y: 20})
	if base == scrolled {
		t.Error("widgetVersion does not include the viewport")
	}
}

func TestWidgetFingerprintTracksDrawingInputs(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:              "timeline",
		ResourceHeader:  "Resource",
		Resources:       []Resource{{ID: "room", Label: "Room A"}},
		Events:          []Event{{ID: "event", ResourceID: "room", Title: "Review", Start: start, End: start.Add(time.Hour)}},
		View:            DayView(start),
		SelectedEventID: "event",
		Width:           300,
		Height:          100,
		ContentWidth:    600,
		ResourceWidth:   100,
		HeaderHeight:    20,
		LaneHeight:      40,
		EventInset:      2,
	}
	fingerprint := func() uint64 {
		scene, err := BuildScene(cfg.Resources, cfg.Events, LayoutConfig{
			Range: cfg.View.Range, Width: cfg.ContentWidth,
			LaneHeight: cfg.LaneHeight, EventInset: cfg.EventInset,
		})
		if err != nil {
			t.Fatalf("BuildScene: %v", err)
		}
		return widgetFingerprint(cfg, newWidgetRenderer(cfg, scene, viewportState{}))
	}

	base := fingerprint()
	if got := fingerprint(); got != base {
		t.Fatalf("fingerprint is not deterministic: got %d, want %d", got, base)
	}
	cfg.Events[0].Title = "Updated review"
	if got := fingerprint(); got == base {
		t.Error("event title change did not invalidate drawing")
	}
	cfg.Events[0].Title = "Review"
	beforeColor := fingerprint()
	cfg.Events[0].Color = gg.Red
	if got := fingerprint(); got == beforeColor {
		t.Error("event color change did not invalidate drawing")
	}
	beforeTextColor := fingerprint()
	cfg.Events[0].TextColor = gg.Black
	if got := fingerprint(); got == beforeTextColor {
		t.Error("event text color change did not invalidate drawing")
	}
}

func TestExplicitContentVersionControlsDataFingerprint(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	cfg := Config{
		ID: "timeline", ContentVersion: 1,
		Resources: []Resource{{ID: "room", Label: "Room A"}},
		Events: []Event{{
			ID: "event", ResourceID: "room", Title: "Review",
			Start: start, End: start.Add(time.Hour),
		}},
		View: DayView(start), Width: 300, Height: 100, ContentWidth: 600,
		ResourceWidth: 100, HeaderHeight: 20, LaneHeight: 40, EventInset: 2,
	}
	scene, err := BuildScene(cfg.Resources, cfg.Events, LayoutConfig{
		Range: cfg.View.Range, Width: cfg.ContentWidth,
		LaneHeight: cfg.LaneHeight, EventInset: cfg.EventInset,
	})
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}
	fingerprint := func() uint64 {
		return resolvedWidgetFingerprint(
			cfg, newWidgetRenderer(cfg, scene, viewportState{}),
		)
	}

	base := fingerprint()
	cfg.Events[0].Title = "Changed without a revision"
	if got := fingerprint(); got != base {
		t.Errorf("event mutation changed explicit fingerprint: got %d, want %d", got, base)
	}
	cfg.ContentVersion++
	if got := fingerprint(); got == base {
		t.Error("content-version change did not invalidate drawing")
	}
}

func TestCachedWidgetSceneAutomaticAndExplicitModes(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	cfg := Config{
		Resources: []Resource{{ID: "room"}},
		Events: []Event{{
			ID: "event", ResourceID: "room",
			Start: start, End: start.Add(time.Hour),
		}},
		View:         ViewSpec{Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)}},
		ContentWidth: 800, LaneHeight: 40, EventInset: 2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})

	first := cachedWidgetScene(w, "timeline", cfg)
	second := cachedWidgetScene(w, "timeline", cfg)
	if &first.Rows[0] != &second.Rows[0] {
		t.Error("unchanged automatic data did not reuse the cached scene")
	}

	cfg.Events[0].Start = start.Add(2 * time.Hour)
	cfg.Events[0].End = start.Add(3 * time.Hour)
	automaticChanged := cachedWidgetScene(w, "timeline", cfg)
	if &first.Rows[0] == &automaticChanged.Rows[0] {
		t.Error("automatic mode reused scene after an event geometry change")
	}
	if &first.eventLocations[0] != &automaticChanged.eventLocations[0] {
		t.Error("scene rebuild did not reuse the event-location buffer")
	}

	cfg.ContentVersion = 1
	explicit := cachedWidgetScene(w, "timeline", cfg)
	cfg.Events[0].Start = start.Add(3 * time.Hour)
	cfg.Events[0].End = start.Add(4 * time.Hour)
	withoutBump := cachedWidgetScene(w, "timeline", cfg)
	if &explicit.Rows[0] != &withoutBump.Rows[0] {
		t.Error("explicit mode rebuilt scene without a content-version change")
	}

	cfg.ContentVersion++
	withBump := cachedWidgetScene(w, "timeline", cfg)
	if &explicit.Rows[0] == &withBump.Rows[0] {
		t.Error("explicit mode reused scene after a content-version change")
	}
}

func TestSceneEventLayoutUsesSourceIndex(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	scene, err := BuildScene(
		[]Resource{{ID: "first"}, {ID: "second"}},
		[]Event{
			{ID: "outside", ResourceID: "first", Start: start.Add(-2 * time.Hour), End: start.Add(-time.Hour)},
			{ID: "visible", ResourceID: "second", Start: start.Add(time.Hour), End: start.Add(2 * time.Hour)},
		},
		LayoutConfig{
			Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)},
			Width: 800, LaneHeight: 40, EventInset: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	if _, ok := sceneEventLayout(scene, 0); ok {
		t.Error("out-of-range event unexpectedly has visible layout")
	}
	layout, ok := sceneEventLayout(scene, 1)
	if !ok || layout.ID != "visible" || layout.SourceIndex != 1 {
		t.Errorf("visible layout = (%+v, %v), want source event 1", layout, ok)
	}
	for _, index := range []int{-1, 2} {
		if _, ok := sceneEventLayout(scene, index); ok {
			t.Errorf("invalid source index %d unexpectedly has layout", index)
		}
	}
}

func TestResetEventLocationsReusesUntilCapacityExceeded(t *testing.T) {
	t.Parallel()

	cache := make([]eventLocation, 2, 4)
	cache[0] = eventLocation{rowIndexPlusOne: 1, eventIndex: 2}
	address := &cache[0]

	reused := resetEventLocations(cache, 3)
	if &reused[0] != address {
		t.Error("event-location buffer moved while requested size fit its capacity")
	}
	for i, location := range reused {
		if location != (eventLocation{}) {
			t.Errorf("reused location %d was not cleared: %+v", i, location)
		}
	}

	grown := resetEventLocations(reused, 5)
	if &grown[0] == address {
		t.Error("overflowing event-location buffer unexpectedly kept its address")
	}
	if len(grown) != 5 || cap(grown) < 5 {
		t.Errorf("grown buffer len/cap = %d/%d, want len 5", len(grown), cap(grown))
	}
}

func TestCanvasVersionsUseOnlyRelevantViewportAxis(t *testing.T) {
	t.Parallel()

	const base = uint64(42)
	atOrigin := widgetCanvasVersions(base, viewportState{})
	horizontal := widgetCanvasVersions(base, viewportState{X: 20})
	vertical := widgetCanvasVersions(base, viewportState{Y: 20})
	if atOrigin.TimeHeader == horizontal.TimeHeader {
		t.Error("time-header version does not track horizontal viewport")
	}
	if atOrigin.ResourceHeader != horizontal.ResourceHeader {
		t.Error("resource-header version unexpectedly tracks horizontal viewport")
	}
	if atOrigin.ResourceHeader == vertical.ResourceHeader {
		t.Error("resource-header version does not track vertical viewport")
	}
	if atOrigin.TimeHeader != vertical.TimeHeader {
		t.Error("time-header version unexpectedly tracks vertical viewport")
	}
	if atOrigin.Body != horizontal.Body || atOrigin.Body != vertical.Body {
		t.Error("body version unexpectedly depends on viewport")
	}
	if atOrigin.Corner != horizontal.Corner || atOrigin.Corner != vertical.Corner {
		t.Error("corner version unexpectedly depends on viewport")
	}
}

func TestClampViewport(t *testing.T) {
	t.Parallel()

	limits := viewportLimits{X: 100, Y: 200}
	tests := []struct {
		name string
		in   viewportState
		want viewportState
	}{
		{name: "inside", in: viewportState{X: 25, Y: 50, ViewKind: viewWeek}, want: viewportState{X: 25, Y: 50, ViewKind: viewWeek}},
		{name: "negative", in: viewportState{X: -1, Y: -2}, want: viewportState{}},
		{name: "past end", in: viewportState{X: 150, Y: 250}, want: viewportState{X: 100, Y: 200}},
		{name: "non-finite", in: viewportState{X: float32(math.NaN()), Y: float32(math.Inf(1))}, want: viewportState{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := clampViewport(test.in, limits); got != test.want {
				t.Errorf("clampViewport(%+v) = %+v, want %+v", test.in, got, test.want)
			}
		})
	}
}

func TestTimelineScrollUpdatesItsViewport(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	resources := []Resource{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	cfg := Config{
		ID:            "timeline",
		Resources:     resources,
		View:          ViewSpec{Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)}},
		Width:         300,
		Height:        100,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View {
		return New(w, cfg)
	})

	if err := w.TestScroll("timeline", -2, -2); err != nil {
		t.Fatalf("TestScroll: %v", err)
	}
	x := w.ScrollX().GetOr("timeline", 0)
	y := w.ScrollY().GetOr("timeline", 0)
	if x >= 0 || y >= 0 {
		t.Errorf("scroll offsets after diagonal scroll = (%v, %v), want both negative", x, y)
	}
}

func TestTimelineUsesStandardScrollbars(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:            "timeline",
		Resources:     []Resource{{ID: "room"}},
		View:          DayView(start),
		Width:         300,
		Height:        100,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	if got := w.ResolveID("timeline-horizontal-scrollbar"); len(got) != 1 {
		t.Errorf("horizontal scrollbar IDs = %v, want one", got)
	}
	if got := w.ResolveID("timeline-vertical-scrollbar"); len(got) != 1 {
		t.Errorf("vertical scrollbar IDs = %v, want one", got)
	}
}

func TestResourceHeaderChangeRedrawsCorner(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID: "timeline", ResourceHeader: "Resource",
		Resources: []Resource{{ID: "room"}}, View: DayView(start),
		Width: 300, Height: 100, ContentWidth: 600,
		ResourceWidth: 100, HeaderHeight: 20, LaneHeight: 40, EventInset: 2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })

	cfg.ResourceHeader = "Room"
	w.TestRender(nil)
	var found bool
	for _, command := range w.Renderers() {
		if command.Kind == gg.RenderText && command.Text == "Room" {
			found = true
			break
		}
	}
	if !found {
		t.Error("changed resource header was not rendered")
	}
}

func TestChangingViewKindResetsScroll(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:            "timeline",
		Resources:     []Resource{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		View:          DayView(start),
		Width:         300,
		Height:        100,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	if err := w.TestScroll("timeline", -2, -2); err != nil {
		t.Fatalf("TestScroll: %v", err)
	}

	cfg.View = WeekView(start, time.Monday)
	w.TestRender(nil)
	if x := w.ScrollX().GetOr("timeline", 0); x != 0 {
		t.Errorf("horizontal offset after view-kind change = %v, want 0", x)
	}
	if y := w.ScrollY().GetOr("timeline", 0); y != 0 {
		t.Errorf("vertical offset after view-kind change = %v, want 0", y)
	}
}

func TestResourceChangePreservesHorizontalScroll(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:            "timeline",
		Resources:     []Resource{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		View:          ViewSpec{Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)}},
		Width:         300,
		Height:        100,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	if err := w.TestScroll("timeline", -2, -2); err != nil {
		t.Fatalf("TestScroll: %v", err)
	}
	wantX := w.ScrollX().GetOr("timeline", 0)
	if wantX >= 0 || w.ScrollY().GetOr("timeline", 0) >= 0 {
		t.Fatal("test setup did not scroll both axes")
	}

	cfg.Resources = []Resource{{ID: "d"}, {ID: "e"}, {ID: "f"}}
	w.TestRender(nil)
	if got := w.ScrollX().GetOr("timeline", 0); got != wantX {
		t.Errorf("horizontal offset after vertical reset = %v, want %v", got, wantX)
	}
	if got := w.ScrollY().GetOr("timeline", 0); got != 0 {
		t.Errorf("vertical offset after vertical reset = %v, want 0", got)
	}
}

func TestSameViewKindNavigationPreservesScroll(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:            "timeline",
		Resources:     []Resource{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		View:          DayView(start),
		Width:         300,
		Height:        100,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	if err := w.TestScroll("timeline", -2, -2); err != nil {
		t.Fatalf("TestScroll: %v", err)
	}
	wantX := w.ScrollX().GetOr("timeline", 0)
	wantY := w.ScrollY().GetOr("timeline", 0)

	cfg.View = DayView(start.AddDate(0, 0, 1))
	w.TestRender(nil)
	if got := w.ScrollX().GetOr("timeline", 0); got != wantX {
		t.Errorf("horizontal offset after date navigation = %v, want %v", got, wantX)
	}
	if got := w.ScrollY().GetOr("timeline", 0); got != wantY {
		t.Errorf("vertical offset after date navigation = %v, want %v", got, wantY)
	}
}

func TestContentWidthChangePreservesViewportCenter(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:            "timeline",
		Resources:     []Resource{{ID: "room"}},
		View:          DayView(start),
		Width:         400,
		Height:        120,
		ContentWidth:  600,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 500, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	w.ScrollX().Set("timeline", -120)
	bodyWidth := cfg.Width - cfg.ResourceWidth - gg.CurrentTheme().ScrollbarStyle.Size - 4
	wantCenter := (float32(120) + bodyWidth/2) / cfg.ContentWidth

	cfg.ContentWidth = 1200
	w.TestRender(nil)
	gotX := -w.ScrollX().GetOr("timeline", 0)
	gotCenter := (gotX + bodyWidth/2) / cfg.ContentWidth
	if math.Abs(float64(gotCenter-wantCenter)) > 0.001 {
		t.Errorf("center ratio after zoom = %v, want %v", gotCenter, wantCenter)
	}
}

func TestHandleWidgetClickUsesScrolledContentCoordinates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	event := Event{
		ID:         "meeting",
		ResourceID: "room",
		Start:      start.Add(4 * time.Hour),
		End:        start.Add(5 * time.Hour),
	}
	scene, err := BuildScene(
		[]Resource{{ID: "room"}},
		[]Event{event},
		LayoutConfig{
			Range:      TimeRange{Start: start, End: start.Add(8 * time.Hour)},
			Width:      800,
			LaneHeight: 40,
			EventInset: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	var clicked EventID
	cfg := Config{
		Events:        []Event{event},
		ResourceWidth: 100,
		HeaderHeight:  20,
		OnEventClick: func(_ gg.EventCtx, event Event) {
			clicked = event.ID
		},
	}
	pointer := &gg.Event{MouseX: 450, MouseY: 10}
	handleWidgetClick(gg.EventCtx{Event: pointer}, cfg, scene)
	if clicked != event.ID {
		t.Errorf("clicked ID = %q, want %q", clicked, event.ID)
	}
	if !pointer.IsHandled {
		t.Error("event hit was not consumed")
	}
}

func TestTruncatedEventHoverShowsSingleTooltip(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:        "timeline",
		Resources: []Resource{{ID: "room"}},
		Events: []Event{{
			ID: "meeting", ResourceID: "room", Title: "A title too long to fit",
			Start: start, End: start.Add(time.Hour),
		}},
		View:          DayView(start),
		Width:         300,
		Height:        100,
		ContentWidth:  240,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	applyConfigDefaults(&cfg)
	scene, err := BuildScene(cfg.Resources, cfg.Events, LayoutConfig{
		Range: cfg.View.Range, Width: cfg.ContentWidth,
		LaneHeight: cfg.LaneHeight, EventInset: cfg.EventInset,
	})
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	states := gg.StateMap[string, eventTooltipState](w, tooltipStateNS, tooltipStateCap)
	pointer := &gg.Event{MouseX: 5, MouseY: 10}
	handleWidgetMouseMove(
		gg.EventCtx{Event: pointer, Window: w}, cfg, scene,
		gg.CurrentTheme().TextStyleBodySmall, states, "timeline",
	)
	w.TestRender(nil)

	if got := w.ResolveID("timeline-event-tooltip"); len(got) != 1 {
		t.Errorf("tooltip IDs = %v, want one tooltip for the hovered event", got)
	}
}

func TestScrolledResourceTextRemainsUntilLabelLeavesViewport(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	cfg := Config{
		ID:        "timeline",
		Resources: []Resource{{ID: "room", Label: "Room A"}},
		Events: []Event{
			{ID: "base", ResourceID: "room", Title: "Base event",
				Start: start, End: start.Add(2 * time.Hour)},
			{ID: "upper", ResourceID: "room", Title: "Hidden event",
				Start: start.Add(time.Hour), End: start.Add(3 * time.Hour)},
		},
		View:          ViewSpec{Range: TimeRange{Start: start, End: start.Add(4 * time.Hour)}},
		Width:         300,
		Height:        50,
		ContentWidth:  400,
		ResourceWidth: 100,
		HeaderHeight:  20,
		LaneHeight:    40,
		EventInset:    2,
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 100})
	w.TestRender(func(w *gg.Window) gg.View {
		return New(w, cfg)
	})
	w.ScrollY().Set("timeline", -45)
	w.TestRender(nil)

	var found bool
	for _, cmd := range w.Renderers() {
		if cmd.Kind != gg.RenderText {
			continue
		}
		if cmd.Text == "Room A" {
			found = true
		}
	}
	if !found {
		t.Error("partially visible resource label vanished before reaching the viewport edge")
	}
}

func TestTimelineExposesOneActiveEventProxy(t *testing.T) {
	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	var labeledEvent EventID
	var labeledResource ResourceID
	cfg := Config{
		ID:        "timeline",
		Resources: []Resource{{ID: "room", Label: "Room A"}},
		Events: []Event{{
			ID: "meeting", ResourceID: "room", Title: "Planning",
			Start: start.Add(time.Hour), End: start.Add(2 * time.Hour),
		}},
		View:  ViewSpec{Range: TimeRange{Start: start, End: start.Add(4 * time.Hour)}},
		Width: 300, Height: 100, ContentWidth: 400,
		ResourceWidth: 100, HeaderHeight: 20, LaneHeight: 40, EventInset: 2,
		OnEventClick: func(gg.EventCtx, Event) {},
		EventA11YLabel: func(event Event, resource Resource) string {
			labeledEvent = event.ID
			labeledResource = resource.ID
			return "localized event label"
		},
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	root := w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })

	proxyIDs := w.ResolveID(activeEventIDPart)
	if len(proxyIDs) != 1 {
		t.Fatalf("active event proxy IDs = %v, want exactly one", proxyIDs)
	}
	proxy, ok := root.FindByID(proxyIDs[0])
	if !ok {
		t.Fatalf("active event proxy %q not found", proxyIDs[0])
	}
	if proxy.Shape.A11YRole != gg.AccessRoleButton || !proxy.Shape.Focusable {
		t.Errorf("proxy role/focusable = (%v, %v), want (button, true)",
			proxy.Shape.A11YRole, proxy.Shape.Focusable)
	}
	if labeledEvent != "meeting" || labeledResource != "room" {
		t.Errorf("label callback received (%q, %q), want (meeting, room)",
			labeledEvent, labeledResource)
	}

	for _, leaf := range []string{
		"timeline-body", "timeline-time-header",
		"timeline-resource-header", "timeline-corner",
	} {
		ids := w.ResolveID(leaf)
		if len(ids) != 1 {
			t.Fatalf("%s IDs = %v, want one", leaf, ids)
		}
		painted, found := root.FindByID(ids[0])
		if !found {
			t.Fatalf("painted canvas %q not found", ids[0])
		}
		if painted.Shape.A11YRole != gg.AccessRoleNone {
			t.Errorf("%s accessibility role = %v, want none",
				leaf, painted.Shape.A11YRole)
		}
	}
}

func TestActiveEventKeyboardNavigationScrollsAndActivates(t *testing.T) {
	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	var activated EventID
	cfg := Config{
		ID:        "timeline",
		Resources: []Resource{{ID: "room", Label: "Room A"}},
		Events: []Event{
			{ID: "first", ResourceID: "room", Title: "First",
				Start: start, End: start.Add(time.Hour)},
			{ID: "second", ResourceID: "room", Title: "Second",
				Start: start.Add(7 * time.Hour), End: start.Add(8 * time.Hour)},
		},
		View:  ViewSpec{Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)}},
		Width: 300, Height: 100, ContentWidth: 800,
		ResourceWidth: 100, HeaderHeight: 20, LaneHeight: 40, EventInset: 2,
		OnEventClick: func(_ gg.EventCtx, event Event) { activated = event.ID },
	}
	w := gg.NewTestWindow(gg.WindowCfg{Width: 400, Height: 200})
	w.TestRender(func(w *gg.Window) gg.View { return New(w, cfg) })
	proxyIDs := w.ResolveID(activeEventIDPart)
	if len(proxyIDs) != 1 {
		t.Fatalf("active event proxy IDs = %v, want exactly one", proxyIDs)
	}

	if err := w.TestKey(proxyIDs[0], gg.KeyRight, gg.ModNone); err != nil {
		t.Fatalf("Right: %v", err)
	}
	state := gg.StateMap[string, activeEventState](
		w, activeEventStateNS, activeEventStateCap,
	).GetOr("timeline", activeEventState{})
	if state.EventID != "second" {
		t.Errorf("active event = %q, want second", state.EventID)
	}
	if offset := w.ScrollX().GetOr("timeline", 0); offset >= 0 {
		t.Errorf("horizontal scroll offset = %v, want a negative offset", offset)
	}

	if err := w.TestKey(proxyIDs[0], gg.KeyEnter, gg.ModNone); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if activated != "second" {
		t.Errorf("activated event = %q, want second", activated)
	}
	activated = ""
	if err := w.TestKey(proxyIDs[0], gg.KeySpace, gg.ModNone); err != nil {
		t.Fatalf("Space: %v", err)
	}
	if activated != "second" {
		t.Errorf("space-activated event = %q, want second", activated)
	}
}

func TestVerticalEventNavigationChoosesNearestTime(t *testing.T) {
	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "active", ResourceID: "a", Start: start.Add(4 * time.Hour), End: start.Add(5 * time.Hour)},
		{ID: "early", ResourceID: "b", Start: start, End: start.Add(time.Hour)},
		{ID: "near", ResourceID: "b", Start: start.Add(5 * time.Hour), End: start.Add(6 * time.Hour)},
	}
	scene, err := BuildScene(
		[]Resource{{ID: "a"}, {ID: "b"}}, events,
		LayoutConfig{
			Range: TimeRange{Start: start, End: start.Add(8 * time.Hour)},
			Width: 800, LaneHeight: 40, EventInset: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}
	active, ok := sceneEventLayout(scene, 0)
	if !ok {
		t.Fatal("active event missing from scene")
	}
	next, handled := navigateEvent(scene, active, gg.KeyDown, false)
	if !handled || next.ID != "near" {
		t.Errorf("Down = (%q, %v), want (near, true)", next.ID, handled)
	}
}

func TestNewRejectsInvalidWidgetDimensions(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{
			name: "resource column consumes width",
			edit: func(cfg *Config) {
				cfg.Width = 100
				cfg.ResourceWidth = 100
			},
		},
		{
			name: "negative header height",
			edit: func(cfg *Config) {
				cfg.HeaderHeight = -1
			},
		},
		{
			name: "header consumes height",
			edit: func(cfg *Config) {
				cfg.Height = 40
				cfg.HeaderHeight = 40
			},
		},
		{
			name: "negative content width",
			edit: func(cfg *Config) {
				cfg.ContentWidth = -1
			},
		},
		{
			name: "tick outside view range",
			edit: func(cfg *Config) {
				cfg.View.Ticks = []Tick{{At: start.Add(2 * time.Hour)}}
			},
		},
		{
			name: "ticks not strictly increasing",
			edit: func(cfg *Config) {
				cfg.View.Ticks = []Tick{{At: start.Add(30 * time.Minute)}, {At: start}}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{
				ID:   "timeline",
				View: ViewSpec{Range: TimeRange{Start: start, End: start.Add(time.Hour)}},
			}
			test.edit(&cfg)
			defer func() {
				if recover() == nil {
					t.Error("New did not panic")
				}
			}()
			New(nil, cfg)
		})
	}
}
