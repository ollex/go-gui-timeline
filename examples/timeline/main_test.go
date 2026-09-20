package main

import (
	"testing"
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/ollex/go-gui-timeline"
)

func TestCalendarNavigationChangesAnchor(t *testing.T) {
	t.Parallel()

	state := appState{
		anchor: time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC),
	}
	state.navigate(1)
	if state.anchor.Day() != 22 {
		t.Errorf("navigation moved to day %d, want 22", state.anchor.Day())
	}
}

func TestChangingCalendarScaleChangesView(t *testing.T) {
	t.Parallel()

	state := appState{}
	state.setCalendarView(viewWeek)
	if state.calendarView != viewWeek {
		t.Errorf("calendar view = %v, want week", state.calendarView)
	}
}

func TestPreviousWeekRedrawsDateHeaders(t *testing.T) {
	t.Parallel()

	state := newAppState()
	state.database.delay = 0
	state.calendarView = viewWeek
	w := gui.NewTestWindow(gui.WindowCfg{
		State:  state,
		Width:  1060,
		Height: 560,
	})
	w.TestRender(mainView)
	ids := w.ResolveID("previous")
	if len(ids) != 1 {
		t.Fatalf("previous button IDs = %v, want one", ids)
	}
	if err := w.TestClick(ids[0]); err != nil {
		t.Fatalf("TestClick: %v", err)
	}

	for _, cmd := range w.Renderers() {
		if cmd.Kind == gui.RenderText && cmd.Text == "14.09." {
			return
		}
	}
	t.Error("previous week did not redraw the 14.09. date header")
}

func TestMonthNavigationStartsAtFirstOfAdjacentMonth(t *testing.T) {
	t.Parallel()

	state := appState{
		anchor:       time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC),
		calendarView: viewMonth,
	}
	state.navigate(1)
	want := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	if !state.anchor.Equal(want) {
		t.Errorf("month navigation = %v, want %v", state.anchor, want)
	}
}

func TestYearNavigationStartsAtFirstOfAdjacentYear(t *testing.T) {
	t.Parallel()

	state := appState{
		anchor:       time.Date(2028, time.February, 29, 12, 0, 0, 0, time.UTC),
		calendarView: viewYear,
	}
	state.navigate(1)
	want := time.Date(2029, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !state.anchor.Equal(want) {
		t.Errorf("year navigation = %v, want %v", state.anchor, want)
	}
}

func TestZoomChangesScaleWithoutResettingViewport(t *testing.T) {
	t.Parallel()

	state := appState{zoomLevel: 2}
	state.zoomBy(1)
	if state.zoomLevel != 3 {
		t.Errorf("zoom level = %d, want 3", state.zoomLevel)
	}
	state.zoomBy(100)
	if state.zoomLevel != len(zoomLevels)-1 {
		t.Errorf("maximum zoom level = %d, want %d", state.zoomLevel, len(zoomLevels)-1)
	}
}

func TestMonthAndYearButtonsRedrawCalendarHeaders(t *testing.T) {
	t.Parallel()

	state := newAppState()
	state.database.delay = 0
	w := gui.NewTestWindow(gui.WindowCfg{
		State: state, Width: 1060, Height: 560,
	})
	w.TestRender(mainView)

	clickResolved(t, w, "month")
	if !rendererHasText(w, "01.09.") || !rendererHasText(w, "09.2026") {
		t.Error("month view did not render its daily and period headers")
	}

	clickResolved(t, w, "year")
	if !rendererHasText(w, "01") || !rendererHasText(w, "2026") {
		t.Error("year view did not render its monthly and period headers")
	}
}

func TestExampleAppliesConfiguredResourceHeader(t *testing.T) {
	t.Parallel()

	state := newAppState()
	w := gui.NewTestWindow(gui.WindowCfg{
		State: state, Width: 1060, Height: 560,
	})
	w.TestRender(mainView)
	if !rendererHasText(w, state.resourceHeader) {
		t.Errorf("resource header %q was not rendered", state.resourceHeader)
	}
}

func TestResourcePaginationChangesRowsAndOnlyResetsVerticalViewport(t *testing.T) {
	t.Parallel()

	state := newLargeAppState()
	state.database.delay = 0
	w := gui.NewTestWindow(gui.WindowCfg{
		State: state, Width: 1060, Height: 700,
	})
	w.TestRender(mainView)
	clickResolved(t, w, "resources-next")
	waitForLoad(t, w, state)

	if state.resourcePage != 1 {
		t.Errorf("resource page = %d, want 1", state.resourcePage)
	}
	if !rendererHasText(w, "Resource 021") {
		t.Error("second resource page did not render Resource 021")
	}
	if rendererHasText(w, "Resource 001") {
		t.Error("first-page resource remained rendered on second page")
	}
}

func TestFakeDatabaseLoadsOnlyRequestedResourcesAndRange(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	database := newFakeTimelineDatabase(anchor)
	database.delay = 0
	query := timelineQuery{
		Range:        timeline.DayView(anchor).Range,
		ResourcePage: 0,
		PageSize:     6,
	}
	result, err := database.load(query)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(result.Resources) != 6 || result.PageCount != 2 {
		t.Fatalf("resource result = %d resources across %d pages, want 6 across 2",
			len(result.Resources), result.PageCount)
	}
	loaded := make(map[timeline.ResourceID]bool, len(result.Resources))
	for _, resource := range result.Resources {
		loaded[resource.ID] = true
	}
	for _, event := range result.Events {
		if !loaded[event.ResourceID] {
			t.Errorf("event %q references resource %q outside the loaded page",
				event.ID, event.ResourceID)
		}
		if !event.End.After(query.Range.Start) || !event.Start.Before(query.Range.End) {
			t.Errorf("event %q does not intersect requested range %v", event.ID, query.Range)
		}
		if !event.Color.IsSet() {
			t.Errorf("event %q has no color after database mapping", event.ID)
		}
	}
}

func TestLargeDataModeLoadsOneResourcePage(t *testing.T) {
	t.Parallel()

	state := newLargeAppState()
	if len(state.database.resources) != 200 || len(state.database.events) != 24_000 {
		t.Fatalf("stored data = %d resources and %d events, want 200 and 24000",
			len(state.database.resources), len(state.database.events))
	}
	if len(state.resources) != resourcePageSize || len(state.events) != 2_400 {
		t.Errorf("loaded data = %d resources and %d events, want %d and 2400",
			len(state.resources), len(state.events), resourcePageSize)
	}
	if state.resourcePageCount != 10 || state.calendarView != viewYear {
		t.Errorf("large mode = page count %d, view %v; want 10 and year",
			state.resourcePageCount, state.calendarView)
	}
}

func BenchmarkFakeDatabaseLargeYearPage(b *testing.B) {
	anchor := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	database := newLargeFakeTimelineDatabase(anchor)
	database.delay = 0
	query := timelineQuery{
		Range: timeline.YearView(anchor).Range, PageSize: resourcePageSize,
	}
	b.ReportAllocs()
	b.ReportMetric(float64(len(database.events)), "stored-events/op")

	for b.Loop() {
		if _, err := database.load(query); err != nil {
			b.Fatal(err)
		}
	}
}

func waitForLoad(t *testing.T, w *gui.Window, state *appState) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for state.loading {
		w.FrameFn()
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for fake database load")
		}
		time.Sleep(time.Millisecond)
	}
	w.TestRender(nil)
}

func clickResolved(t *testing.T, w *gui.Window, leaf string) {
	t.Helper()
	ids := w.ResolveID(leaf)
	if len(ids) != 1 {
		t.Fatalf("%s IDs = %v, want one", leaf, ids)
	}
	if err := w.TestClick(ids[0]); err != nil {
		t.Fatalf("TestClick(%s): %v", leaf, err)
	}
}

func rendererHasText(w *gui.Window, text string) bool {
	for _, cmd := range w.Renderers() {
		if cmd.Kind == gui.RenderText && cmd.Text == text {
			return true
		}
	}
	return false
}
