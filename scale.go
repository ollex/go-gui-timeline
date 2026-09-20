package timeline

import (
	"errors"
	"time"
)

// TimeScale converts between times and horizontal timeline coordinates.
type TimeScale struct {
	rangeStart time.Time
	rangeEnd   time.Time
	duration   time.Duration
	width      float32
}

// NewTimeScale constructs a linear scale for a time range and content width.
func NewTimeScale(timeRange TimeRange, width float32) (TimeScale, error) {
	if !timeRange.End.After(timeRange.Start) {
		return TimeScale{}, errors.New("timeline: range end must be after start")
	}
	if width <= 0 {
		return TimeScale{}, errors.New("timeline: width must be positive")
	}
	return TimeScale{
		rangeStart: timeRange.Start,
		rangeEnd:   timeRange.End,
		duration:   timeRange.End.Sub(timeRange.Start),
		width:      width,
	}, nil
}

// X returns the horizontal coordinate corresponding to a time.
// Times outside the scale's range produce coordinates outside its width.
func (s TimeScale) X(at time.Time) float32 {
	return s.width * float32(float64(at.Sub(s.rangeStart))/float64(s.duration))
}

// Time returns the time corresponding to a horizontal coordinate.
// Coordinates outside the scale's width produce times outside its range.
func (s TimeScale) Time(x float32) time.Time {
	ratio := float64(x / s.width)
	return s.rangeStart.Add(time.Duration(ratio * float64(s.duration)))
}

// Width returns the horizontal distance between two times.
func (s TimeScale) Width(start, end time.Time) float32 {
	return s.X(end) - s.X(start)
}

func (s TimeScale) clip(start, end time.Time) (time.Time, time.Time, bool) {
	if !end.After(s.rangeStart) || !start.Before(s.rangeEnd) {
		return time.Time{}, time.Time{}, false
	}
	if start.Before(s.rangeStart) {
		start = s.rangeStart
	}
	if end.After(s.rangeEnd) {
		end = s.rangeEnd
	}
	return start, end, true
}
