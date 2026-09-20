package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/ollex/go-gui-timeline"
)

const fakeDatabaseDelay = 180 * time.Millisecond

type timelineQuery struct {
	Range        timeline.TimeRange
	ResourcePage int
	PageSize     int
}

type timelineResult struct {
	Resources []timeline.Resource
	Events    []timeline.Event
	PageCount int
}

// fakeTimelineDatabase stands in for application-owned storage. The widget
// never sees the complete data set; it receives only this query's result.
type fakeTimelineDatabase struct {
	resources []timeline.Resource
	events    []timeline.Event
	delay     time.Duration
}

func newFakeTimelineDatabase(anchor time.Time) *fakeTimelineDatabase {
	workday := anchor.Add(8 * time.Hour)
	return &fakeTimelineDatabase{
		resources: []timeline.Resource{
			{ID: "studio", Label: "Design Studio"},
			{ID: "workshop", Label: "Workshop"},
			{ID: "camera", Label: "Camera Team"},
			{ID: "edit", Label: "Editing Suite"},
			{ID: "sound", Label: "Sound Stage"},
			{ID: "lighting", Label: "Lighting Crew"},
			{ID: "delivery", Label: "Delivery Bay"},
		},
		events: []timeline.Event{
			{ID: "overnight", ResourceID: "sound", Title: "Day boundary: overnight mix", Start: anchor.Add(-2 * time.Hour), End: anchor.Add(2 * time.Hour)},
			{ID: "briefing", ResourceID: "studio", Title: "Morning briefing", Start: workday.Add(-30 * time.Minute), End: workday.Add(time.Hour)},
			{ID: "review", ResourceID: "studio", Title: "Design review", Start: workday.Add(2 * time.Hour), End: workday.Add(4 * time.Hour)},
			{ID: "vendor", ResourceID: "studio", Title: "Vendor call (overlap)", Start: workday.Add(3 * time.Hour), End: workday.Add(4*time.Hour + 30*time.Minute)},
			{ID: "build", ResourceID: "workshop", Title: "Prototype build", Start: workday.Add(time.Hour), End: workday.Add(5 * time.Hour)},
			{ID: "shoot", ResourceID: "camera", Title: "Product shoot", Start: workday.Add(2*time.Hour + 30*time.Minute), End: workday.Add(6 * time.Hour)},
			{ID: "cut", ResourceID: "edit", Title: "Rough cut", Start: workday.Add(5 * time.Hour), End: workday.Add(9 * time.Hour)},
			{ID: "voice", ResourceID: "sound", Title: "Voice-over session", Start: workday.Add(4 * time.Hour), End: workday.Add(6 * time.Hour)},
			{ID: "rig", ResourceID: "lighting", Title: "Lighting setup", Start: workday.Add(time.Hour), End: workday.Add(3*time.Hour + 30*time.Minute)},
			{ID: "pickup", ResourceID: "delivery", Title: "Equipment pickup", Start: workday.Add(7 * time.Hour), End: workday.Add(9*time.Hour + 30*time.Minute)},
			{ID: "week-handoff", ResourceID: "delivery", Title: "Week boundary: Sunday handoff", Start: anchor.AddDate(0, 0, 6).Add(20 * time.Hour), End: anchor.AddDate(0, 0, 7).Add(4 * time.Hour)},
			{ID: "rehearsal", ResourceID: "studio", Title: "Tuesday rehearsal", Start: workday.AddDate(0, 0, 1), End: workday.AddDate(0, 0, 1).Add(3 * time.Hour)},
			{ID: "assembly", ResourceID: "workshop", Title: "Wednesday assembly", Start: workday.AddDate(0, 0, 2).Add(time.Hour), End: workday.AddDate(0, 0, 2).Add(6 * time.Hour)},
			{ID: "final", ResourceID: "edit", Title: "Friday final edit", Start: workday.AddDate(0, 0, 4), End: workday.AddDate(0, 0, 4).Add(5 * time.Hour)},
			{ID: "campaign", ResourceID: "studio", Title: "Month boundary: campaign", Start: anchor.AddDate(0, 0, 6), End: anchor.AddDate(0, 0, 11)},
			{ID: "maintenance", ResourceID: "workshop", Title: "October maintenance", Start: anchor.AddDate(0, 1, -16), End: anchor.AddDate(0, 1, -2)},
			{ID: "archive", ResourceID: "edit", Title: "Year boundary: annual archive", Start: time.Date(2026, time.December, 28, 0, 0, 0, 0, anchor.Location()), End: time.Date(2027, time.January, 4, 0, 0, 0, 0, anchor.Location())},
		},
		delay: fakeDatabaseDelay,
	}
}

func newLargeFakeTimelineDatabase(anchor time.Time) *fakeTimelineDatabase {
	const (
		resourceCount     = 200
		eventsPerResource = 120
	)
	resources := make([]timeline.Resource, 0, resourceCount)
	events := make([]timeline.Event, 0, resourceCount*eventsPerResource)
	year, _, _ := anchor.Date()
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, anchor.Location())
	for resourceIndex := range resourceCount {
		resourceID := timeline.ResourceID(fmt.Sprintf("resource-%03d", resourceIndex+1))
		resources = append(resources, timeline.Resource{
			ID: resourceID, Label: fmt.Sprintf("Resource %03d", resourceIndex+1),
		})
		for eventIndex := range eventsPerResource {
			day := (eventIndex*37 + resourceIndex*11) % 365
			hour := (eventIndex*5 + resourceIndex*3) % 24
			start := yearStart.AddDate(0, 0, day).Add(time.Duration(hour) * time.Hour)
			duration := time.Duration(1+(eventIndex+resourceIndex)%12) * time.Hour
			if eventIndex%29 == 0 {
				duration = 72 * time.Hour
			}
			events = append(events, timeline.Event{
				ID: timeline.EventID(fmt.Sprintf(
					"event-%03d-%03d", resourceIndex+1, eventIndex+1,
				)),
				ResourceID: resourceID,
				Start:      start,
				End:        start.Add(duration),
				Title:      fmt.Sprintf("Event %03d.%03d", resourceIndex+1, eventIndex+1),
			})
		}
	}
	return &fakeTimelineDatabase{
		resources: resources,
		events:    events,
		delay:     fakeDatabaseDelay,
	}
}

func (db *fakeTimelineDatabase) load(query timelineQuery) (timelineResult, error) {
	if query.PageSize <= 0 {
		return timelineResult{}, errors.New("fake database: page size must be positive")
	}
	if query.ResourcePage < 0 {
		return timelineResult{}, errors.New("fake database: resource page must be non-negative")
	}
	if !query.Range.End.After(query.Range.Start) {
		return timelineResult{}, errors.New("fake database: range end must be after start")
	}
	if db.delay > 0 {
		time.Sleep(db.delay)
	}

	pageCount := 0
	if len(db.resources) > 0 {
		pageCount = (len(db.resources) + query.PageSize - 1) / query.PageSize
	}
	if query.ResourcePage >= pageCount && (query.ResourcePage != 0 || pageCount != 0) {
		return timelineResult{}, fmt.Errorf(
			"fake database: resource page %d is outside %d pages",
			query.ResourcePage, pageCount,
		)
	}

	start := min(query.ResourcePage*query.PageSize, len(db.resources))
	end := min(start+query.PageSize, len(db.resources))
	resources := append([]timeline.Resource(nil), db.resources[start:end]...)
	loadedResources := make(map[timeline.ResourceID]struct{}, len(resources))
	for _, resource := range resources {
		loadedResources[resource.ID] = struct{}{}
	}
	events := make([]timeline.Event, 0)
	for _, event := range db.events {
		if _, loaded := loadedResources[event.ResourceID]; !loaded {
			continue
		}
		if event.End.After(query.Range.Start) && event.Start.Before(query.Range.End) {
			event.Color = bgColor(event)
			events = append(events, event)
		}
	}

	return timelineResult{
		Resources: resources,
		Events:    events,
		PageCount: pageCount,
	}, nil
}
