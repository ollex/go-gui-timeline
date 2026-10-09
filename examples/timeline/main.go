package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/go-gui-org/go-gui/gui/backend"
	"github.com/go-gui-org/go-gui/gui/backend/soft"
	timeline "github.com/ollex/go-gui-timeline"
)

const (
	timelineWidth       = float32(970)
	resourceColumnWidth = float32(170)
	resourcePageSize    = 20
)

var zoomLevels = [...]float32{0.55, 0.75, 1, 1.5, 2, 3, 4, 5}

type calendarView uint8

const (
	viewDay calendarView = iota
	viewWeek
	viewMonth
	viewYear
)

type appState struct {
	database          *fakeTimelineDatabase
	resourceHeader    string
	resources         []timeline.Resource
	events            []timeline.Event
	anchor            time.Time
	calendarView      calendarView
	zoomLevel         int
	resourcePage      int
	resourcePageCount int
	requestID         uint64
	loading           bool
	loadError         string
	selected          timeline.EventID
	contentVersion    uint64
	status            string
}

func main() {
	screenshot := flag.String("screenshot", "", "write screenshot and exit")
	large := flag.Bool("large", false, "start with 200 resources and 24,000 events")
	flag.Parse()

	gui.SetTheme(gui.ThemeDark)
	state := newAppState()
	if *large {
		state = newLargeAppState()
	}
	w := gui.NewWindow(gui.WindowCfg{
		State:  state,
		Title:  "Timeline",
		Width:  1060,
		Height: 640,
		OnInit: func(w *gui.Window) {
			w.SetView(mainView)
		},
	})

	if *screenshot != "" {
		if err := soft.RenderToPNG(w, 2, *screenshot); err != nil {
			log.Fatalf("screenshot: %v", err)
		}
		os.Exit(0)
	}
	backend.Run(w)
}

func newAppState() *appState {
	anchor := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.Local)
	return newAppStateWithDatabase(anchor, newFakeTimelineDatabase(anchor), viewDay)
}

func newLargeAppState() *appState {
	anchor := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.Local)
	state := newAppStateWithDatabase(
		anchor, newLargeFakeTimelineDatabase(anchor), viewYear,
	)
	state.status = "Large-data mode: 200 resources and 24,000 events in storage."
	return state
}

func newAppStateWithDatabase(
	anchor time.Time,
	database *fakeTimelineDatabase,
	view calendarView,
) *appState {
	state := &appState{
		database:       database,
		resourceHeader: "Room",
		anchor:         anchor,
		calendarView:   view,
		zoomLevel:      2,
		status:         "",
	}
	state.database.delay = 0
	result, err := state.database.load(state.query())
	state.database.delay = fakeDatabaseDelay
	if err != nil {
		panic(fmt.Sprintf("timeline example: initial load: %v", err))
	}
	state.applyResult(result)
	return state
}

func (state *appState) view() timeline.ViewSpec {
	switch state.calendarView {
	case viewWeek:
		return timeline.WeekView(state.anchor, time.Monday)
	case viewMonth:
		return timeline.MonthView(state.anchor)
	case viewYear:
		return timeline.YearView(state.anchor)
	default:
		return timeline.DayView(state.anchor)
	}
}

func (state *appState) query() timelineQuery {
	return timelineQuery{
		Range:        state.view().Range,
		ResourcePage: state.resourcePage,
		PageSize:     resourcePageSize,
	}
}

func (state *appState) applyResult(result timelineResult) {
	state.resources = result.Resources
	state.events = result.Events
	state.resourcePageCount = result.PageCount
	state.contentVersion++
}

func (state *appState) requestData(w *gui.Window) {
	state.requestID++
	requestID := state.requestID
	query := state.query()
	database := state.database
	state.loading = true
	state.loadError = ""

	go func() {
		result, err := database.load(query)
		w.QueueCommand(func(w *gui.Window) {
			if requestID != state.requestID {
				return
			}
			state.loading = false
			if err != nil {
				state.loadError = err.Error()
			} else {
				state.applyResult(result)
			}
			w.InvalidateLayout()
		})
	}()
}

func (state *appState) navigate(periods int) {
	switch state.calendarView {
	case viewDay:
		state.anchor = state.anchor.AddDate(0, 0, periods)
	case viewWeek:
		state.anchor = state.anchor.AddDate(0, 0, periods*7)
	case viewMonth:
		year, month, _ := state.anchor.Date()
		state.anchor = time.Date(
			year, month+time.Month(periods), 1, 0, 0, 0, 0, state.anchor.Location(),
		)
	case viewYear:
		year, _, _ := state.anchor.Date()
		state.anchor = time.Date(
			year+periods, time.January, 1, 0, 0, 0, 0, state.anchor.Location(),
		)
	}
}

func (state *appState) setCalendarView(view calendarView) bool {
	if state.calendarView == view {
		return false
	}
	state.calendarView = view
	return true
}

func (state *appState) zoomBy(steps int) {
	next := max(0, min(len(zoomLevels)-1, state.zoomLevel+steps))
	if next == state.zoomLevel {
		return
	}
	state.zoomLevel = next
}

func (state *appState) resetZoom() {
	const defaultZoomLevel = 2
	if state.zoomLevel == defaultZoomLevel {
		return
	}
	state.zoomLevel = defaultZoomLevel
}

func (state *appState) changeResourcePage(pages int) bool {
	if state.resourcePageCount == 0 {
		return false
	}
	next := max(0, min(state.resourcePageCount-1, state.resourcePage+pages))
	if next == state.resourcePage {
		return false
	}
	state.resourcePage = next
	return true
}

func bgColor(event timeline.Event) gui.Color {
	switch event.ResourceID {
	case "studio", "resource-001":
		return gui.RGB(70, 130, 22)
	default:
		return gui.RGB(61, 125, 216)
	}
}

func mainView(w *gui.Window) gui.View {
	state := gui.State[appState](w)
	theme := gui.CurrentTheme()
	view := state.view()
	contentWidth := float32(1440)
	periodLabel := view.Range.Start.Format("02.01.2006")
	switch state.calendarView {
	case viewWeek:
		contentWidth = 1680
		periodLabel = fmt.Sprintf("%s – %s",
			view.Range.Start.Format("02.01."),
			view.Range.End.AddDate(0, 0, -1).Format("02.01.2006"),
		)
	case viewMonth:
		contentWidth = 2480
		periodLabel = view.Range.Start.Format("01.2006")
	case viewYear:
		contentWidth = 2400
		periodLabel = view.Range.Start.Format("2006")
	}
	contentWidth *= zoomLevels[state.zoomLevel]
	zoomLabel := fmt.Sprintf("%d%%", int(zoomLevels[state.zoomLevel]*100))
	pageLabel := fmt.Sprintf("Resources %d/%d", state.resourcePage+1, state.resourcePageCount)
	loadStatus := fmt.Sprintf("Loaded %d resources and %d events", len(state.resources), len(state.events))
	if state.loading {
		loadStatus = "Loading resources and events from the fake database..."
	} else if state.loadError != "" {
		loadStatus = "Load failed: " + state.loadError
	}
	return gui.Column(gui.ContainerCfg{
		ID:         "timeline-page",
		Scrollable: true,
		ScrollMode: gui.ScrollVerticalOnly,
		Sizing:     gui.FillFill,
		Padding:    theme.PaddingLarge,
		Spacing:    gui.SpacingPx(8),
		Content: []gui.View{
			gui.Row(gui.ContainerCfg{
				VAlign:  gui.VAlignMiddle,
				Spacing: gui.SpacingPx(8),
				Content: []gui.View{
					iconButton("previous", gui.IconArrowLeft, "Previous period", func(ctx gui.EventCtx) {
						state.navigate(-1)
						state.requestData(ctx.Window)
					}),
					gui.Button(gui.ButtonCfg{ID: "day", Label: "Day", OnClick: func(ctx gui.EventCtx) {
						if state.setCalendarView(viewDay) {
							state.requestData(ctx.Window)
						}
					}}),
					gui.Button(gui.ButtonCfg{ID: "week", Label: "Week", OnClick: func(ctx gui.EventCtx) {
						if state.setCalendarView(viewWeek) {
							state.requestData(ctx.Window)
						}
					}}),
					gui.Button(gui.ButtonCfg{ID: "month", Label: "Month", OnClick: func(ctx gui.EventCtx) {
						if state.setCalendarView(viewMonth) {
							state.requestData(ctx.Window)
						}
					}}),
					gui.Button(gui.ButtonCfg{ID: "year", Label: "Year", OnClick: func(ctx gui.EventCtx) {
						if state.setCalendarView(viewYear) {
							state.requestData(ctx.Window)
						}
					}}),
					iconButton("next", gui.IconArrowRight, "Next period", func(ctx gui.EventCtx) {
						state.navigate(1)
						state.requestData(ctx.Window)
					}),
					gui.Text(gui.TextCfg{Text: periodLabel, TextStyle: theme.TextStyleTitle}),
				},
			}),
			gui.Row(gui.ContainerCfg{
				VAlign:  gui.VAlignMiddle,
				Spacing: gui.SpacingPx(24),
				Content: []gui.View{
					gui.Row(gui.ContainerCfg{
						VAlign:  gui.VAlignMiddle,
						Spacing: gui.SpacingPx(8),
						Content: []gui.View{
							iconButton("zoom-out", gui.IconSearchMinus, "Zoom out", func(_ gui.EventCtx) {
								state.zoomBy(-1)
							}),
							gui.Button(gui.ButtonCfg{ID: "zoom-reset", Label: zoomLabel, OnClick: func(_ gui.EventCtx) {
								state.resetZoom()
							}}),
							iconButton("zoom-in", gui.IconSearchPlus, "Zoom in", func(_ gui.EventCtx) {
								state.zoomBy(1)
							}),
						},
					}),
					gui.Row(gui.ContainerCfg{
						VAlign:  gui.VAlignMiddle,
						Spacing: gui.SpacingPx(8),
						Content: []gui.View{
							iconButton("resources-previous", gui.IconBackward, "Previous resources", func(ctx gui.EventCtx) {
								if state.changeResourcePage(-1) {
									state.requestData(ctx.Window)
								}
							}),
							gui.Text(gui.TextCfg{Text: pageLabel, TextStyle: theme.TextStyleTitle}),
							iconButton("resources-next", gui.IconForward, "Next resources", func(ctx gui.EventCtx) {
								if state.changeResourcePage(1) {
									state.requestData(ctx.Window)
								}
							}),
						},
					}),
				},
			}),
			timeline.New(w, timeline.Config{
				ID:              "production-schedule",
				ContentVersion:  state.contentVersion,
				ResourceHeader:  state.resourceHeader,
				Resources:       state.resources,
				Events:          state.events,
				View:            view,
				SelectedEventID: state.selected,
				Width:           timelineWidth,
				Height:          350,
				ContentWidth:    contentWidth,
				ResourceWidth:   resourceColumnWidth,
				HeaderHeight:    48,
				LaneHeight:      48,
				EventInset:      7,
				A11YLabel:       "Production schedule",
				A11YDescription: "Use arrow keys to move between events. Press Enter or Space to select an event.",
				EventA11YLabel: func(event timeline.Event, resource timeline.Resource) string {
					return fmt.Sprintf("%s, %s, %s to %s",
						event.Title, resource.Label,
						event.Start.Format("15:04"), event.End.Format("15:04"))
				},
				OnEventClick: func(_ gui.EventCtx, event timeline.Event) {
					state.selected = event.ID
					state.status = fmt.Sprintf("Selected %q (%s–%s).",
						event.Title,
						event.Start.Format("15:04"),
						event.End.Format("15:04"),
					)
				},
			}),
			gui.Text(gui.TextCfg{
				Text:      loadStatus,
				TextStyle: theme.TextStyleSecondary,
			}),
			gui.Text(gui.TextCfg{
				Text:      state.status,
				TextStyle: theme.TextStyleTitle,
			}),
		},
	})
}

func iconButton(id, icon, accessibleLabel string, onClick func(gui.EventCtx)) gui.View {
	return gui.Button(gui.ButtonCfg{
		ID:      id,
		OnClick: onClick,
		A11YCfg: gui.A11YCfg{A11YLabel: accessibleLabel},
		Padding: gui.NewPadding(8, 12, 8, 12),
		Content: []gui.View{
			gui.Text(gui.TextCfg{
				Text:      icon,
				TextStyle: gui.CurrentTheme().TextStyleIconSmall,
			}),
		},
	})
}
