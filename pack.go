package timeline

import (
	"cmp"
	"slices"
	"time"
)

type pendingEvent struct {
	event       Event
	visibleFrom time.Time
	visibleTo   time.Time
	sourceIndex int
	lane        int
}

// packEvents sorts events chronologically and greedily puts each event in the
// first lane whose previous event has ended. Intervals are half-open, so an
// event starting exactly when another ends can reuse its lane.
func packEvents(events []pendingEvent) int {
	slices.SortFunc(events, func(a, b pendingEvent) int {
		if order := a.event.Start.Compare(b.event.Start); order != 0 {
			return order
		}
		if order := a.event.End.Compare(b.event.End); order != 0 {
			return order
		}
		return cmp.Compare(a.event.ID, b.event.ID)
	})

	laneEnds := make([]time.Time, 0)
	for i := range events {
		lane := 0
		for ; lane < len(laneEnds); lane++ {
			if !events[i].event.Start.Before(laneEnds[lane]) {
				break
			}
		}
		if lane == len(laneEnds) {
			laneEnds = append(laneEnds, events[i].event.End)
		} else {
			laneEnds[lane] = events[i].event.End
		}
		events[i].lane = lane
	}
	return max(1, len(laneEnds))
}
