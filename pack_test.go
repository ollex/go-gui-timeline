package timeline_test

import (
	"testing"
	"time"

	"github.com/ollex/go-gui-timeline"
)

func TestBuildScenePacksOverlapsIntoLanes(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	scene, err := timeline.BuildScene(
		[]timeline.Resource{{ID: "busy"}, {ID: "empty"}},
		[]timeline.Event{
			newEvent("third", "busy", start, 3*time.Hour, 5*time.Hour),
			newEvent("fourth", "busy", start, 5*time.Hour, 6*time.Hour),
			newEvent("first", "busy", start, time.Hour, 4*time.Hour),
			newEvent("second", "busy", start, 2*time.Hour, 3*time.Hour),
		},
		timeline.LayoutConfig{
			Range:      timeline.TimeRange{Start: start, End: start.Add(8 * time.Hour)},
			Width:      800,
			LaneHeight: 20,
			EventInset: 2,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	busy := scene.Rows[0]
	if busy.LaneCount != 2 {
		t.Fatalf("busy LaneCount = %d, want 2", busy.LaneCount)
	}
	if busy.Rect.Height != 40 {
		t.Errorf("busy row height = %v, want 40", busy.Rect.Height)
	}
	wantIDs := []timeline.EventID{"first", "second", "third", "fourth"}
	wantLanes := []int{0, 1, 1, 0}
	for i, event := range busy.Events {
		if event.ID != wantIDs[i] || event.Lane != wantLanes[i] {
			t.Errorf("event %d = (%q, lane %d), want (%q, lane %d)",
				i, event.ID, event.Lane, wantIDs[i], wantLanes[i])
		}
	}
	if busy.Events[0].Rect.Y != 2 || busy.Events[1].Rect.Y != 22 {
		t.Errorf("lane event Y coordinates = (%v, %v), want (2, 22)",
			busy.Events[0].Rect.Y, busy.Events[1].Rect.Y)
	}

	empty := scene.Rows[1]
	if empty.LaneCount != 1 || empty.Rect.Y != 40 || empty.Rect.Height != 20 {
		t.Errorf("empty row = %+v, want one lane at y=40 with height 20", empty)
	}
	if scene.Height != 60 {
		t.Errorf("scene height = %v, want 60", scene.Height)
	}

	if hit, ok := scene.HitTest(250, 5); !ok || hit.EventID != "first" {
		t.Errorf("first-lane HitTest = (%+v, %v), want first", hit, ok)
	}
	if hit, ok := scene.HitTest(250, 25); !ok || hit.EventID != "second" {
		t.Errorf("second-lane HitTest = (%+v, %v), want second", hit, ok)
	}
	if _, ok := scene.HitTest(250, 20); ok {
		t.Error("HitTest in the inset between lanes found an event")
	}
}

func TestBuildScenePackingIsIndependentOfInputOrder(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC)
	alpha := newEvent("alpha", "room", start, time.Hour, 2*time.Hour)
	beta := newEvent("beta", "room", start, time.Hour, 2*time.Hour)
	config := timeline.LayoutConfig{
		Range:      timeline.TimeRange{Start: start, End: start.Add(4 * time.Hour)},
		Width:      400,
		LaneHeight: 20,
	}

	forward, err := timeline.BuildScene(
		[]timeline.Resource{{ID: "room"}}, []timeline.Event{alpha, beta}, config)
	if err != nil {
		t.Fatalf("BuildScene forward: %v", err)
	}
	reverse, err := timeline.BuildScene(
		[]timeline.Resource{{ID: "room"}}, []timeline.Event{beta, alpha}, config)
	if err != nil {
		t.Fatalf("BuildScene reverse: %v", err)
	}

	for i, wantID := range []timeline.EventID{"alpha", "beta"} {
		forwardEvent := forward.Rows[0].Events[i]
		reverseEvent := reverse.Rows[0].Events[i]
		if forwardEvent.ID != wantID || reverseEvent.ID != wantID ||
			forwardEvent.Lane != i || reverseEvent.Lane != i {
			t.Errorf("event %d differs by input order: forward=%+v reverse=%+v",
				i, forwardEvent, reverseEvent)
		}
	}
}

func newEvent(
	id timeline.EventID,
	resourceID timeline.ResourceID,
	base time.Time,
	startOffset, endOffset time.Duration,
) timeline.Event {
	return timeline.Event{
		ID:         id,
		ResourceID: resourceID,
		Start:      base.Add(startOffset),
		End:        base.Add(endOffset),
	}
}
