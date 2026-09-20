package timeline_test

import (
	"math"
	"testing"
	"time"

	"github.com/ollex/go-gui-timeline"
)

func TestBuildScenePositionsRowsAndClipsEvents(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	resources := []timeline.Resource{
		{ID: "room-a", Label: "Room A"},
		{ID: "room-b", Label: "Room B"},
	}
	events := []timeline.Event{
		{
			ID:         "starts-before",
			ResourceID: "room-a",
			Start:      start.Add(-time.Hour),
			End:        start.Add(2 * time.Hour),
		},
		{
			ID:         "middle",
			ResourceID: "room-b",
			Start:      start.Add(4 * time.Hour),
			End:        start.Add(6 * time.Hour),
		},
		{
			ID:         "ends-after",
			ResourceID: "room-a",
			Start:      start.Add(8 * time.Hour),
			End:        start.Add(12 * time.Hour),
		},
		{
			ID:         "outside",
			ResourceID: "room-b",
			Start:      start.Add(11 * time.Hour),
			End:        start.Add(12 * time.Hour),
		},
	}

	scene, err := timeline.BuildScene(resources, events, timeline.LayoutConfig{
		Range: timeline.TimeRange{
			Start: start,
			End:   start.Add(10 * time.Hour),
		},
		Width:      1000,
		LaneHeight: 40,
		EventInset: 5,
	})
	if err != nil {
		t.Fatalf("BuildScene: %v", err)
	}

	if scene.Width != 1000 || scene.Height != 80 {
		t.Fatalf("scene size = %vx%v, want 1000x80", scene.Width, scene.Height)
	}
	if len(scene.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(scene.Rows))
	}
	assertRect(t, scene.Rows[0].Rect, timeline.Rect{Width: 1000, Height: 40})
	assertRect(t, scene.Rows[1].Rect, timeline.Rect{Y: 40, Width: 1000, Height: 40})

	if len(scene.Rows[0].Events) != 2 {
		t.Fatalf("first row events = %d, want 2", len(scene.Rows[0].Events))
	}
	assertRect(t, scene.Rows[0].Events[0].Rect, timeline.Rect{
		Y: 5, Width: 200, Height: 30,
	})
	assertRect(t, scene.Rows[0].Events[1].Rect, timeline.Rect{
		X: 800, Y: 5, Width: 200, Height: 30,
	})
	if len(scene.Rows[1].Events) != 1 {
		t.Fatalf("second row events = %d, want 1", len(scene.Rows[1].Events))
	}
	assertRect(t, scene.Rows[1].Events[0].Rect, timeline.Rect{
		X: 400, Y: 45, Width: 200, Height: 30,
	})
	if scene.Rows[1].Events[0].SourceIndex != 1 {
		t.Errorf("SourceIndex = %d, want 1", scene.Rows[1].Events[0].SourceIndex)
	}
}

func TestBuildSceneRejectsInvalidModels(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	validConfig := timeline.LayoutConfig{
		Range:      timeline.TimeRange{Start: start, End: start.Add(time.Hour)},
		Width:      100,
		LaneHeight: 20,
	}
	validResource := timeline.Resource{ID: "resource"}
	validEvent := timeline.Event{
		ID:         "event",
		ResourceID: validResource.ID,
		Start:      start,
		End:        start.Add(time.Minute),
	}
	tests := []struct {
		name      string
		resources []timeline.Resource
		events    []timeline.Event
		config    timeline.LayoutConfig
	}{
		{
			name:      "non-positive lane height",
			resources: []timeline.Resource{validResource},
			config:    withLaneHeight(validConfig, 0),
		},
		{
			name:      "negative event inset",
			resources: []timeline.Resource{validResource},
			config:    withEventInset(validConfig, -1),
		},
		{
			name:      "event inset consumes row",
			resources: []timeline.Resource{validResource},
			config:    withEventInset(validConfig, 10),
		},
		{
			name:      "empty resource ID",
			resources: []timeline.Resource{{}},
			config:    validConfig,
		},
		{
			name: "duplicate resource ID",
			resources: []timeline.Resource{
				validResource,
				validResource,
			},
			config: validConfig,
		},
		{
			name:      "empty event ID",
			resources: []timeline.Resource{validResource},
			events: []timeline.Event{{
				ResourceID: validResource.ID,
				Start:      start,
				End:        start.Add(time.Minute),
			}},
			config: validConfig,
		},
		{
			name:      "duplicate event ID",
			resources: []timeline.Resource{validResource},
			events:    []timeline.Event{validEvent, validEvent},
			config:    validConfig,
		},
		{
			name:      "empty event interval",
			resources: []timeline.Resource{validResource},
			events: []timeline.Event{{
				ID:         "empty",
				ResourceID: validResource.ID,
				Start:      start,
				End:        start,
			}},
			config: validConfig,
		},
		{
			name:      "unknown resource",
			resources: []timeline.Resource{validResource},
			events: []timeline.Event{{
				ID:         "missing-resource",
				ResourceID: "missing",
				Start:      start,
				End:        start.Add(time.Minute),
			}},
			config: validConfig,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := timeline.BuildScene(test.resources, test.events, test.config); err == nil {
				t.Fatal("BuildScene succeeded, want an error")
			}
		})
	}
}

func assertRect(t *testing.T, got, want timeline.Rect) {
	t.Helper()
	const tolerance = 0.001
	if math.Abs(float64(got.X-want.X)) > tolerance ||
		math.Abs(float64(got.Y-want.Y)) > tolerance ||
		math.Abs(float64(got.Width-want.Width)) > tolerance ||
		math.Abs(float64(got.Height-want.Height)) > tolerance {
		t.Errorf("Rect = %+v, want %+v", got, want)
	}
}

func withLaneHeight(config timeline.LayoutConfig, height float32) timeline.LayoutConfig {
	config.LaneHeight = height
	return config
}

func withEventInset(config timeline.LayoutConfig, inset float32) timeline.LayoutConfig {
	config.EventInset = inset
	return config
}
