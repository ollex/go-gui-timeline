package timeline

import "sort"

// HitTest returns the topmost event containing a point in content coordinates.
func (s Scene) HitTest(x, y float32) (Hit, bool) {
	if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
		return Hit{}, false
	}
	rowIndex := sort.Search(len(s.rowBottoms), func(i int) bool {
		return y < s.rowBottoms[i]
	})
	if rowIndex == len(s.Rows) || !s.Rows[rowIndex].Rect.Contains(x, y) {
		return Hit{}, false
	}

	row := &s.Rows[rowIndex]
	laneHeight := row.Rect.Height / float32(row.LaneCount)
	lane := min(int((y-row.Rect.Y)/laneHeight), row.LaneCount-1)
	// BuildScene stores events in chronological order, so Rect.X is
	// nondecreasing. Discard every event that starts to the right of x.
	eventEnd := sort.Search(len(row.Events), func(i int) bool {
		return row.Events[i].Rect.X > x
	})
	for i := eventEnd - 1; i >= 0; i-- {
		event := &row.Events[i]
		if event.Lane != lane {
			continue
		}
		if !event.Rect.Contains(x, y) {
			// This is the latest event in the pointer's lane that starts at
			// or before x. Events within a lane never overlap, so no earlier
			// event in that lane can contain x.
			return Hit{}, false
		}
		return Hit{
			EventID:       event.ID,
			ResourceID:    event.ResourceID,
			EventIndex:    event.SourceIndex,
			ResourceIndex: event.ResourceIndex,
		}, true
	}
	return Hit{}, false
}
