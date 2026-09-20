package timeline_test

import (
	"math"
	"testing"
	"time"

	"github.com/ollex/go-gui-timeline"
)

func TestDayViewUsesCalendarMidnightsAcrossDST(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skipf("load timezone: %v", err)
	}
	view := timeline.DayView(time.Date(2026, time.March, 29, 12, 0, 0, 0, location))
	if got := view.Range.End.Sub(view.Range.Start); got != 23*time.Hour {
		t.Errorf("DST day duration = %v, want 23h", got)
	}
	if len(view.Ticks) != 24 {
		t.Fatalf("tick count = %d, want 23 hourly ticks plus end boundary", len(view.Ticks))
	}
	for _, tick := range view.Ticks {
		if tick.Label == "02:00" {
			t.Error("spring-forward day unexpectedly contains a 02:00 tick")
		}
	}
	if view.Ticks[len(view.Ticks)-1].Label != "" {
		t.Error("end boundary should not have a label")
	}
}

func TestWeekViewAlignsToRequestedFirstDay(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.September, 23, 15, 30, 0, 0, time.UTC)
	view := timeline.WeekView(anchor, time.Monday)
	wantStart := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)
	if !view.Range.Start.Equal(wantStart) || !view.Range.End.Equal(wantEnd) {
		t.Errorf("week range = %v to %v, want %v to %v",
			view.Range.Start, view.Range.End, wantStart, wantEnd)
	}
	if len(view.Ticks) != 8 {
		t.Fatalf("tick count = %d, want 8", len(view.Ticks))
	}
	if view.Ticks[0].Label != "21.09." || view.Ticks[6].Label != "27.09." {
		t.Errorf("week labels = %q ... %q, want 21.09. ... 27.09.",
			view.Ticks[0].Label, view.Ticks[6].Label)
	}
	if view.Ticks[7].Label != "" || !view.Ticks[7].At.Equal(wantEnd) {
		t.Errorf("end tick = %+v, want unlabeled %v", view.Ticks[7], wantEnd)
	}
}

func TestMonthViewUsesCalendarMonthAndDailyTicks(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2028, time.February, 20, 15, 30, 0, 0, time.UTC)
	view := timeline.MonthView(anchor)
	wantStart := time.Date(2028, time.February, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2028, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !view.Range.Start.Equal(wantStart) || !view.Range.End.Equal(wantEnd) {
		t.Errorf("month range = %v to %v, want %v to %v",
			view.Range.Start, view.Range.End, wantStart, wantEnd)
	}
	if len(view.Ticks) != 30 {
		t.Fatalf("leap-February tick count = %d, want 29 days plus end", len(view.Ticks))
	}
	if view.Ticks[0].Label != "01.02." || view.Ticks[28].Label != "29.02." {
		t.Errorf("month labels = %q ... %q, want 01.02. ... 29.02.",
			view.Ticks[0].Label, view.Ticks[28].Label)
	}
	if view.Ticks[29].Label != "" || !view.Ticks[29].At.Equal(wantEnd) {
		t.Errorf("end tick = %+v, want unlabeled %v", view.Ticks[29], wantEnd)
	}
}

func TestYearViewUsesCalendarYearAndMonthlyTicks(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2028, time.September, 20, 15, 30, 0, 0, time.UTC)
	view := timeline.YearView(anchor)
	wantStart := time.Date(2028, time.January, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2029, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !view.Range.Start.Equal(wantStart) || !view.Range.End.Equal(wantEnd) {
		t.Errorf("year range = %v to %v, want %v to %v",
			view.Range.Start, view.Range.End, wantStart, wantEnd)
	}
	if len(view.Ticks) != 13 {
		t.Fatalf("year tick count = %d, want 12 months plus end", len(view.Ticks))
	}
	if view.Ticks[0].Label != "01" || view.Ticks[11].Label != "12" {
		t.Errorf("year labels = %q ... %q, want 01 ... 12",
			view.Ticks[0].Label, view.Ticks[11].Label)
	}
	if view.Ticks[12].Label != "" || !view.Ticks[12].At.Equal(wantEnd) {
		t.Errorf("end tick = %+v, want unlabeled %v", view.Ticks[12], wantEnd)
	}
}

func TestCalendarViewsClipEventsAcrossBothBoundaries(t *testing.T) {
	t.Parallel()

	anchor := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	views := map[string]timeline.ViewSpec{
		"day":   timeline.DayView(anchor),
		"week":  timeline.WeekView(anchor, time.Monday),
		"month": timeline.MonthView(anchor),
		"year":  timeline.YearView(anchor),
	}
	for name, view := range views {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			overlap := view.Range.End.Sub(view.Range.Start) / 20
			scene, err := timeline.BuildScene(
				[]timeline.Resource{{ID: "resource"}},
				[]timeline.Event{
					{ID: "crosses-start", ResourceID: "resource",
						Start: view.Range.Start.Add(-overlap), End: view.Range.Start.Add(overlap)},
					{ID: "crosses-end", ResourceID: "resource",
						Start: view.Range.End.Add(-overlap), End: view.Range.End.Add(overlap)},
				},
				timeline.LayoutConfig{
					Range: view.Range, Width: 1000, LaneHeight: 40, EventInset: 2,
				},
			)
			if err != nil {
				t.Fatalf("BuildScene: %v", err)
			}
			if len(scene.Rows) != 1 || len(scene.Rows[0].Events) != 2 {
				t.Fatalf("event layouts = %+v, want two visible boundary fragments", scene.Rows)
			}
			left := scene.Rows[0].Events[0].Rect
			right := scene.Rows[0].Events[1].Rect
			if left.X != 0 || left.Width <= 0 {
				t.Errorf("left fragment = %+v, want positive width at x=0", left)
			}
			if right.Width <= 0 || math.Abs(float64(right.X+right.Width-1000)) > 0.001 {
				t.Errorf("right fragment = %+v, want positive width ending at x=1000", right)
			}
		})
	}
}
