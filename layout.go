package timeline

import (
	"errors"
	"fmt"
)

// BuildScene derives resource rows and visible event rectangles without
// modifying the supplied resources or events.
func BuildScene(
	resources []Resource,
	events []Event,
	config LayoutConfig,
) (Scene, error) {
	return buildScene(resources, events, config, nil)
}

// buildScene accepts an optional event-location buffer. The widget cache owns
// one and reuses it across geometry rebuilds; the public BuildScene function
// remains allocation-independent for callers retaining earlier scenes.
func buildScene(
	resources []Resource,
	events []Event,
	config LayoutConfig,
	eventLocationCache []eventLocation,
) (Scene, error) {
	if config.LaneHeight <= 0 {
		return Scene{}, errors.New("timeline: lane height must be positive")
	}
	if config.EventInset < 0 || config.EventInset*2 >= config.LaneHeight {
		return Scene{}, errors.New(
			"timeline: event inset must be non-negative and leave a positive event height",
		)
	}
	scale, err := NewTimeScale(config.Range, config.Width)
	if err != nil {
		return Scene{}, err
	}

	rows := make([]ResourceLayout, len(resources))
	rowByID := make(map[ResourceID]int, len(resources))
	for i, resource := range resources {
		if resource.ID == "" {
			return Scene{}, fmt.Errorf("timeline: resource at index %d has an empty ID", i)
		}
		if _, exists := rowByID[resource.ID]; exists {
			return Scene{}, fmt.Errorf("timeline: duplicate resource ID %q", resource.ID)
		}
		rows[i] = ResourceLayout{
			ID:          resource.ID,
			SourceIndex: i,
		}
		rowByID[resource.ID] = i
	}

	pendingByRow := make([][]pendingEvent, len(rows))
	eventIDs := make(map[EventID]struct{}, len(events))
	for i, event := range events {
		if event.ID == "" {
			return Scene{}, fmt.Errorf("timeline: event at index %d has an empty ID", i)
		}
		if _, exists := eventIDs[event.ID]; exists {
			return Scene{}, fmt.Errorf("timeline: duplicate event ID %q", event.ID)
		}
		eventIDs[event.ID] = struct{}{}
		if !event.End.After(event.Start) {
			return Scene{}, fmt.Errorf("timeline: event %q end must be after start", event.ID)
		}
		rowIndex, exists := rowByID[event.ResourceID]
		if !exists {
			return Scene{}, fmt.Errorf(
				"timeline: event %q references unknown resource %q",
				event.ID,
				event.ResourceID,
			)
		}
		start, end, visible := scale.clip(event.Start, event.End)
		if !visible {
			continue
		}
		pendingByRow[rowIndex] = append(pendingByRow[rowIndex], pendingEvent{
			event:       event,
			visibleFrom: start,
			visibleTo:   end,
			sourceIndex: i,
		})
	}

	rowBottoms := make([]float32, len(resources))
	eventLocations := resetEventLocations(eventLocationCache, len(events))
	var sceneHeight float32
	for rowIndex := range rows {
		row := &rows[rowIndex]
		pending := pendingByRow[rowIndex]
		row.LaneCount = packEvents(pending)
		row.Rect = Rect{
			Y:      sceneHeight,
			Width:  config.Width,
			Height: float32(row.LaneCount) * config.LaneHeight,
		}
		row.Events = make([]EventLayout, 0, len(pending))
		for _, placed := range pending {
			eventIndex := len(row.Events)
			row.Events = append(row.Events, EventLayout{
				ID:            placed.event.ID,
				ResourceID:    placed.event.ResourceID,
				SourceIndex:   placed.sourceIndex,
				ResourceIndex: rowIndex,
				Lane:          placed.lane,
				Rect: Rect{
					X:      scale.X(placed.visibleFrom),
					Y:      row.Rect.Y + float32(placed.lane)*config.LaneHeight + config.EventInset,
					Width:  scale.Width(placed.visibleFrom, placed.visibleTo),
					Height: config.LaneHeight - 2*config.EventInset,
				},
			})
			eventLocations[placed.sourceIndex] = eventLocation{
				rowIndexPlusOne: rowIndex + 1,
				eventIndex:      eventIndex,
			}
		}
		sceneHeight += row.Rect.Height
		rowBottoms[rowIndex] = sceneHeight
	}

	return Scene{
		Range:          config.Range,
		Width:          config.Width,
		Height:         sceneHeight,
		Rows:           rows,
		rowBottoms:     rowBottoms,
		eventLocations: eventLocations,
		scale:          scale,
	}, nil
}

func resetEventLocations(cache []eventLocation, size int) []eventLocation {
	if cap(cache) < size {
		return make([]eventLocation, size)
	}
	cache = cache[:size]
	clear(cache)
	return cache
}
