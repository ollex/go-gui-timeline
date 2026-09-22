package timeline

import (
	"strings"
	"unicode"

	gg "github.com/go-gui-org/go-gui/gui"
)

const eventTextPadding = float32(8)

const timeLabelGap = float32(8)

var defaultEventColor = gg.RGB(61, 125, 216)

type widgetStyle struct {
	resourceText gg.TextStyle
	timeText     gg.TextStyle
	eventText    gg.TextStyle
	background   gg.Color
	header       gg.Color
	row          gg.Color
	rowAlt       gg.Color
	grid         gg.Color
	selection    gg.Color
	radius       float32
}

type widgetRenderer struct {
	cfg           Config
	scene         Scene
	viewport      viewportState
	style         widgetStyle
	activeEventID EventID
	activeFocused bool
}

func newWidgetRenderer(cfg Config, scene Scene, viewport viewportState) *widgetRenderer {
	theme := gg.CurrentTheme()
	resourceText := theme.TextStyleBody
	timeText := theme.TextStyleSecondary
	eventText := theme.TextStyleBodySmall
	eventText.Color = gg.RGB(250, 250, 252)
	return &widgetRenderer{
		cfg:      cfg,
		scene:    scene,
		viewport: viewport,
		style: widgetStyle{
			resourceText: resourceText,
			timeText:     timeText,
			eventText:    eventText,
			background:   theme.ColorBackground,
			header:       theme.ColorInterior,
			row:          theme.ColorPanel,
			rowAlt:       theme.ColorBackground,
			grid:         theme.ColorBorder,
			selection:    theme.ColorFocus,
			radius:       theme.RadiusMedium,
		},
	}
}

func eventColor(event Event) gg.Color {
	if event.Color.IsSet() {
		return event.Color
	}
	return defaultEventColor
}

func eventTextColor(event Event, fallback gg.Color) gg.Color {
	if event.TextColor.IsSet() {
		return event.TextColor
	}
	return fallback
}

func (r *widgetRenderer) drawBody(dc *gg.DrawContext) {
	r.drawBodyRows(dc)
	r.drawGrid(dc)
	r.drawEvents(dc)
}

func (r *widgetRenderer) drawBodyRows(dc *gg.DrawContext) {
	for rowIndex, row := range r.scene.Rows {
		if !r.rowVisible(row) {
			continue
		}
		y := r.cfg.HeaderHeight + row.Rect.Y - r.viewport.Y
		dc.FilledRect(r.cfg.ResourceWidth, y,
			r.cfg.Width-r.cfg.ResourceWidth, row.Rect.Height, r.rowColor(rowIndex))
		dc.Line(r.cfg.ResourceWidth, y+row.Rect.Height, r.cfg.Width,
			y+row.Rect.Height, r.style.grid, 1)
	}
}

func (r *widgetRenderer) drawResourceRows(dc *gg.DrawContext) {
	dc.FilledRect(0, r.cfg.HeaderHeight, r.cfg.ResourceWidth,
		r.cfg.Height-r.cfg.HeaderHeight, r.style.background)
	for rowIndex, row := range r.scene.Rows {
		if !r.rowVisible(row) {
			continue
		}
		y := r.cfg.HeaderHeight + row.Rect.Y - r.viewport.Y
		dc.FilledRect(0, y, r.cfg.ResourceWidth, row.Rect.Height,
			r.rowColor(rowIndex))
		dc.Line(0, y+row.Rect.Height, r.cfg.ResourceWidth,
			y+row.Rect.Height, r.style.grid, 1)
		textHeight := dc.FontHeight(r.style.resourceText)
		labelY := y + (row.Rect.Height-textHeight)/2
		if textHeight <= row.Rect.Height &&
			boxIntersects(labelY, textHeight, r.cfg.HeaderHeight, r.cfg.Height) {
			label := fitTextWithEllipsis(
				r.cfg.Resources[row.SourceIndex].Label,
				r.cfg.ResourceWidth-28,
				func(text string) float32 {
					return dc.TextWidth(text, r.style.resourceText)
				},
			)
			if label != "" {
				dc.Text(14, labelY, label, r.style.resourceText)
			}
		}
	}
}

func (r *widgetRenderer) drawTimeHeader(dc *gg.DrawContext) {
	dc.FilledRect(r.cfg.ResourceWidth, 0, r.cfg.Width-r.cfg.ResourceWidth,
		r.cfg.HeaderHeight, r.style.header)
	labelY := (r.cfg.HeaderHeight - r.style.timeText.Size) / 2
	lastLabelRight := r.cfg.ResourceWidth - timeLabelGap
	for _, tick := range r.cfg.View.Ticks {
		x := r.cfg.ResourceWidth + r.scene.scale.X(tick.At) - r.viewport.X
		if x < r.cfg.ResourceWidth || x >= r.cfg.Width {
			continue
		}
		labelLeft := x + 5
		labelRight := labelLeft + dc.TextWidth(tick.Label, r.style.timeText)
		if tick.Label != "" && labelLeft >= lastLabelRight+timeLabelGap &&
			labelRight <= r.cfg.Width {
			dc.Text(labelLeft, labelY, tick.Label, r.style.timeText)
			lastLabelRight = labelRight
		}
	}
}

func (r *widgetRenderer) drawCorner(dc *gg.DrawContext) {
	dc.FilledRect(0, 0, r.cfg.ResourceWidth, r.cfg.HeaderHeight, r.style.header)
	textHeight := dc.FontHeight(r.style.resourceText)
	labelY := (r.cfg.HeaderHeight - textHeight) / 2
	label := fitTextWithEllipsis(
		r.cfg.ResourceHeader,
		r.cfg.ResourceWidth-28,
		func(text string) float32 {
			return dc.TextWidth(text, r.style.resourceText)
		},
	)
	if label != "" {
		dc.Text(14, labelY, label, r.style.resourceText)
	}
}

func (r *widgetRenderer) drawGrid(dc *gg.DrawContext) {
	for _, tick := range r.cfg.View.Ticks {
		x := r.cfg.ResourceWidth + r.scene.scale.X(tick.At) - r.viewport.X
		if x < r.cfg.ResourceWidth || x >= r.cfg.Width {
			continue
		}
		dc.Line(x, r.cfg.HeaderHeight, x, r.cfg.Height, r.style.grid, 1)
	}
}

func (r *widgetRenderer) drawEvents(dc *gg.DrawContext) {
	visibleRight := r.viewport.X + r.cfg.Width - r.cfg.ResourceWidth
	for _, row := range r.scene.Rows {
		if !r.rowVisible(row) {
			continue
		}
		for _, eventLayout := range row.Events {
			rect := eventLayout.Rect
			if rect.X+rect.Width <= r.viewport.X || rect.X >= visibleRight {
				continue
			}
			event := r.cfg.Events[eventLayout.SourceIndex]
			x := r.cfg.ResourceWidth + rect.X - r.viewport.X
			y := r.cfg.HeaderHeight + rect.Y - r.viewport.Y
			color := eventColor(event)
			if r.activeFocused && eventLayout.ID == r.activeEventID {
				dc.FilledRoundedRect(x-3, y-3, rect.Width+6, rect.Height+6,
					r.style.radius, r.style.selection)
			}
			if eventLayout.ID == r.cfg.SelectedEventID {
				dc.FilledRoundedRect(x-2, y-2, rect.Width+4, rect.Height+4,
					r.style.radius, r.style.selection)
			}
			dc.FilledRoundedRect(x, y, rect.Width, rect.Height,
				r.style.radius, color)

			textStyle := r.style.eventText
			textStyle.Color = eventTextColor(event, textStyle.Color)
			textLeft := max(x, r.cfg.ResourceWidth) + eventTextPadding
			textRight := min(x+rect.Width, r.cfg.Width) - eventTextPadding
			title := fitTextWithEllipsis(event.Title, textRight-textLeft,
				func(text string) float32 {
					return dc.TextWidth(text, textStyle)
				})
			textHeight := dc.FontHeight(textStyle)
			textY := y + (rect.Height-textHeight)/2
			if title != "" && textHeight <= rect.Height && boxFullyVisible(
				y, rect.Height, r.cfg.HeaderHeight, r.cfg.Height,
			) {
				dc.Text(textLeft, textY, title, textStyle)
			}
		}
	}
}

func (r *widgetRenderer) rowVisible(row ResourceLayout) bool {
	visibleBottom := r.viewport.Y + r.cfg.Height - r.cfg.HeaderHeight
	return row.Rect.Y+row.Rect.Height > r.viewport.Y && row.Rect.Y < visibleBottom
}

func (r *widgetRenderer) rowColor(rowIndex int) gg.Color {
	if rowIndex%2 == 1 {
		return r.style.rowAlt
	}
	return r.style.row
}

func boxFullyVisible(y, height, viewportTop, viewportBottom float32) bool {
	return height > 0 && y >= viewportTop && y+height <= viewportBottom
}

func boxIntersects(y, height, viewportTop, viewportBottom float32) bool {
	return height > 0 && y+height > viewportTop && y < viewportBottom
}

func fitTextWithEllipsis(
	text string,
	maxWidth float32,
	measure func(string) float32,
) string {
	if text == "" || maxWidth <= 0 {
		return ""
	}
	if measure(text) <= maxWidth {
		return text
	}

	const ellipsis = "…"
	if measure(ellipsis) > maxWidth {
		return ""
	}
	runes := []rune(text)
	low, high := 0, len(runes)
	for low < high {
		mid := low + (high-low+1)/2
		candidate := string(runes[:mid]) + ellipsis
		if measure(candidate) <= maxWidth {
			low = mid
		} else {
			high = mid - 1
		}
	}
	prefix := strings.TrimRightFunc(string(runes[:low]), unicode.IsSpace)
	return prefix + ellipsis
}
