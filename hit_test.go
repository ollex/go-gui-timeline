package timeline_test

import (
	"testing"
	"time"

	"github.com/ollex/go-gui-timeline"
)

func TestSceneHitTestFindsEventsByResourceRow(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	scene, err := timeline.BuildScene(
		[]timeline.Resource{{ID: "a"}, {ID: "b"}},
		[]timeline.Event{
			{
				ID:         "first",
				ResourceID: "a",
				Start:      start,
				End:        start.Add(5 * time.Hour),
			},
			{
				ID:         "second",
				ResourceID: "b",
				Start:      start.Add(5 * time.Hour),
				End:        start.Add(10 * time.Hour),
			},
		},
		timeline.LayoutConfig{
			Range:      timeline.TimeRange{Start: start, End: start.Add(10 * time.Hour)},
			Width:      100,
			LaneHeight: 20,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	tests := []struct {
		name    string
		x       float32
		y       float32
		wantID  timeline.EventID
		wantHit bool
	}{
		{name: "first row", x: 25, y: 10, wantID: "first", wantHit: true},
		{name: "second row", x: 75, y: 30, wantID: "second", wantHit: true},
		{name: "empty part of first row", x: 75, y: 10},
		{name: "left of scene", x: -1, y: 10},
		{name: "right edge is exclusive", x: 100, y: 30},
		{name: "bottom edge is exclusive", x: 75, y: 40},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			hit, ok := scene.HitTest(test.x, test.y)
			if ok != test.wantHit {
				t.Fatalf("HitTest hit = %v, want %v", ok, test.wantHit)
			}
			if hit.EventID != test.wantID {
				t.Errorf("HitTest EventID = %q, want %q", hit.EventID, test.wantID)
			}
		})
	}
}

func TestSceneHitTestReturnsLastOverlappingEvent(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	scene, err := timeline.BuildScene(
		[]timeline.Resource{{ID: "room"}},
		[]timeline.Event{
			{
				ID:         "under",
				ResourceID: "room",
				Start:      start,
				End:        start.Add(10 * time.Hour),
			},
			{
				ID:         "over",
				ResourceID: "room",
				Start:      start,
				End:        start.Add(10 * time.Hour),
			},
		},
		timeline.LayoutConfig{
			Range:      timeline.TimeRange{Start: start, End: start.Add(10 * time.Hour)},
			Width:      100,
			LaneHeight: 20,
		},
	)
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	hit, ok := scene.HitTest(50, 10)
	if !ok {
		t.Fatal("HitTest missed overlapping events")
	}
	if hit.EventID != "over" {
		t.Errorf("HitTest EventID = %q, want %q", hit.EventID, "over")
	}
}
