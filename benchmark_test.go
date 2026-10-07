package timeline_test

import (
	"fmt"
	"testing"
	"time"

	gg "github.com/go-gui-org/go-gui/gui"
	"github.com/ollex/go-gui-timeline"
)

var benchmarkScene timeline.Scene

func BenchmarkBuildSceneLarge(b *testing.B) {
	for _, size := range []struct {
		name              string
		resources         int
		eventsPerResource int
	}{
		{name: "loaded_page_2400_events", resources: 20, eventsPerResource: 120},
		{name: "all_storage_24000_events", resources: 200, eventsPerResource: 120},
	} {
		b.Run(size.name, func(b *testing.B) {
			resources, events, start := benchmarkData(size.resources, size.eventsPerResource)
			config := timeline.LayoutConfig{
				Range: timeline.TimeRange{Start: start, End: start.AddDate(1, 0, 0)},
				Width: 2400, LaneHeight: 48, EventInset: 7,
			}
			b.ReportAllocs()
			b.ReportMetric(float64(len(events)), "events/op")
			b.ResetTimer()
			for range b.N {
				scene, err := timeline.BuildScene(resources, events, config)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkScene = scene
			}
		})
	}
}

func BenchmarkTimelineWidgetLargeYearFrame(b *testing.B) {
	benchmarkTimelineWidgetLargeYearFrame(b, true, 0, false)
}

func BenchmarkTimelineWidgetLargeYearCachedFrame(b *testing.B) {
	benchmarkTimelineWidgetLargeYearFrame(b, false, 0, false)
}

func BenchmarkTimelineWidgetLargeYearExplicitVersionCachedFrame(b *testing.B) {
	benchmarkTimelineWidgetLargeYearFrame(b, false, 1, false)
}

func BenchmarkTimelineWidgetLargeYearExplicitVersionScrollFrame(b *testing.B) {
	benchmarkTimelineWidgetLargeYearFrame(b, false, 1, true)
}

func benchmarkTimelineWidgetLargeYearFrame(
	b *testing.B,
	forceRedraw bool,
	contentVersion uint64,
	scroll bool,
) {
	b.Helper()
	resources, events, start := benchmarkData(20, 120)
	cfg := timeline.Config{
		ID: "timeline", Resources: resources, Events: events,
		View: timeline.YearView(start), Width: 970, Height: 350,
		ContentWidth: 2400, ResourceWidth: 170,
		HeaderHeight: 48, LaneHeight: 48, EventInset: 7,
		ContentVersion: contentVersion,
	}
	w := gg.NewTestWindow(b, gg.WindowCfg{Width: 1060, Height: 560})
	w.TestRender(func(w *gg.Window) gg.View { return timeline.New(w, cfg) })
	b.ReportAllocs()
	b.ReportMetric(float64(len(events)), "events/frame")
	b.ResetTimer()
	for i := range b.N {
		if forceRedraw {
			cfg.SelectedEventID = events[i%len(events)].ID
		}
		if scroll {
			offset := -float32(i%2) * 100
			w.ScrollX().Set("timeline", offset)
			w.ScrollY().Set("timeline", offset)
		}
		w.TestRender(nil)
	}
}

func benchmarkData(resourceCount, eventsPerResource int) (
	[]timeline.Resource,
	[]timeline.Event,
	time.Time,
) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	resources := make([]timeline.Resource, 0, resourceCount)
	events := make([]timeline.Event, 0, resourceCount*eventsPerResource)
	for resourceIndex := range resourceCount {
		resourceID := timeline.ResourceID(fmt.Sprintf("resource-%03d", resourceIndex))
		resources = append(resources, timeline.Resource{ID: resourceID})
		for eventIndex := range eventsPerResource {
			day := (eventIndex*37 + resourceIndex*11) % 365
			eventStart := start.AddDate(0, 0, day).Add(
				time.Duration((eventIndex*5+resourceIndex*3)%24) * time.Hour,
			)
			events = append(events, timeline.Event{
				ID: timeline.EventID(fmt.Sprintf(
					"event-%03d-%03d", resourceIndex, eventIndex,
				)),
				ResourceID: resourceID,
				Start:      eventStart,
				End:        eventStart.Add(time.Duration(1+eventIndex%12) * time.Hour),
				Title:      "Benchmark event",
			})
		}
	}
	return resources, events, start
}
