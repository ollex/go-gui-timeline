package timeline

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

var (
	benchmarkEventLayout EventLayout
	benchmarkEventFound  bool
	benchmarkBuiltScene  Scene
	benchmarkHit         Hit
	benchmarkHitFound    bool
)

func BenchmarkSceneEventLayoutLookupLarge(b *testing.B) {
	resources, events, start := benchmarkSceneData()
	scene, err := BuildScene(resources, events, LayoutConfig{
		Range: TimeRange{Start: start, End: start.AddDate(1, 0, 0)},
		Width: 2400, LaneHeight: 48, EventInset: 7,
	})
	if err != nil {
		b.Fatal(err)
	}
	target := len(events) - 1
	b.ReportMetric(float64(len(events)), "events")

	b.Run("indexed", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			benchmarkEventLayout, benchmarkEventFound = sceneEventLayout(scene, target)
		}
	})
	b.Run("previous_linear_scan", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			benchmarkEventLayout, benchmarkEventFound = linearSceneEventLayout(scene, target)
		}
	})
}

func BenchmarkBuildSceneEventLocationBuffer(b *testing.B) {
	resources, events, start := benchmarkSceneData()
	config := LayoutConfig{
		Range: TimeRange{Start: start, End: start.AddDate(1, 0, 0)},
		Width: 2400, LaneHeight: 48, EventInset: 7,
	}

	b.Run("fresh", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			scene, err := BuildScene(resources, events, config)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkBuiltScene = scene
		}
	})
	b.Run("reused_index", func(b *testing.B) {
		var cache []eventLocation
		b.ReportAllocs()
		for range b.N {
			scene, err := buildScene(resources, events, config, cache)
			if err != nil {
				b.Fatal(err)
			}
			cache = scene.eventLocations
			benchmarkBuiltScene = scene
		}
	})
}

func BenchmarkSceneHitTestChronologicalRow(b *testing.B) {
	resources, events, start := benchmarkSceneData()
	scene, err := BuildScene(resources, events, LayoutConfig{
		Range: TimeRange{Start: start, End: start.AddDate(1, 0, 0)},
		Width: 2400, LaneHeight: 48, EventInset: 7,
	})
	if err != nil {
		b.Fatal(err)
	}
	row := scene.Rows[len(scene.Rows)-1]
	target := row.Events[len(row.Events)/4]
	x := target.Rect.X + target.Rect.Width/2
	y := target.Rect.Y + target.Rect.Height/2

	b.Run("chronological_search", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			benchmarkHit, benchmarkHitFound = scene.HitTest(x, y)
		}
	})
	b.Run("previous_reverse_row_scan", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			benchmarkHit, benchmarkHitFound = reverseRowScanHitTest(scene, x, y)
		}
	})
}

func benchmarkSceneData() ([]Resource, []Event, time.Time) {
	const (
		resourceCount     = 20
		eventsPerResource = 120
	)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	resources := make([]Resource, 0, resourceCount)
	events := make([]Event, 0, resourceCount*eventsPerResource)
	for resourceIndex := range resourceCount {
		resourceID := ResourceID(fmt.Sprintf("resource-%03d", resourceIndex))
		resources = append(resources, Resource{ID: resourceID})
		for eventIndex := range eventsPerResource {
			eventStart := start.Add(time.Duration(eventIndex*48) * time.Hour)
			events = append(events, Event{
				ID: EventID(fmt.Sprintf(
					"event-%03d-%03d", resourceIndex, eventIndex,
				)),
				ResourceID: resourceID,
				Start:      eventStart,
				End:        eventStart.Add(time.Hour),
			})
		}
	}
	return resources, events, start
}

func linearSceneEventLayout(scene Scene, eventIndex int) (EventLayout, bool) {
	for _, row := range scene.Rows {
		for _, event := range row.Events {
			if event.SourceIndex == eventIndex {
				return event, true
			}
		}
	}
	return EventLayout{}, false
}

func reverseRowScanHitTest(scene Scene, x, y float32) (Hit, bool) {
	if x < 0 || x >= scene.Width || y < 0 || y >= scene.Height {
		return Hit{}, false
	}
	rowIndex := sort.Search(len(scene.rowBottoms), func(i int) bool {
		return y < scene.rowBottoms[i]
	})
	if rowIndex == len(scene.Rows) || !scene.Rows[rowIndex].Rect.Contains(x, y) {
		return Hit{}, false
	}
	row := &scene.Rows[rowIndex]
	for i := len(row.Events) - 1; i >= 0; i-- {
		event := &row.Events[i]
		if event.Rect.Contains(x, y) {
			return Hit{
				EventID: event.ID, ResourceID: event.ResourceID,
				EventIndex: event.SourceIndex, ResourceIndex: event.ResourceIndex,
			}, true
		}
	}
	return Hit{}, false
}
