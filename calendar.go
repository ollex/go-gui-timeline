package timeline

import "time"

// Tick is a labeled vertical division in a timeline view. An empty label draws
// the division without header text, which is useful for the range end.
type Tick struct {
	At    time.Time
	Label string
}

type viewKind uint8

const (
	viewCustom viewKind = iota
	viewDay
	viewWeek
	viewMonth
	viewYear
)

// ViewSpec defines the visible calendar range and its header divisions.
type ViewSpec struct {
	Range TimeRange
	Ticks []Tick
	kind  viewKind
}

// DayView returns a midnight-to-midnight view with hourly divisions. Calendar
// midnights use anchor's location, so daylight-saving days may span 23 or 25
// elapsed hours.
func DayView(anchor time.Time) ViewSpec {
	start := startOfDay(anchor)
	end := start.AddDate(0, 0, 1)
	ticks := make([]Tick, 0, 26)
	for at := start; at.Before(end); at = at.Add(time.Hour) {
		ticks = append(ticks, Tick{At: at, Label: at.Format("15:04")})
	}
	ticks = append(ticks, Tick{At: end})
	return ViewSpec{Range: TimeRange{Start: start, End: end}, Ticks: ticks, kind: viewDay}
}

// WeekView returns a seven-day view with daily divisions. firstDay selects the
// weekday at the left edge, usually time.Monday or time.Sunday.
func WeekView(anchor time.Time, firstDay time.Weekday) ViewSpec {
	start := startOfDay(anchor)
	daysBack := (int(start.Weekday()) - int(firstDay) + 7) % 7
	start = start.AddDate(0, 0, -daysBack)
	end := start.AddDate(0, 0, 7)
	ticks := make([]Tick, 0, 8)
	for at := start; at.Before(end); at = at.AddDate(0, 0, 1) {
		ticks = append(ticks, Tick{At: at, Label: at.Format("02.01.")})
	}
	ticks = append(ticks, Tick{At: end})
	return ViewSpec{Range: TimeRange{Start: start, End: end}, Ticks: ticks, kind: viewWeek}
}

// MonthView returns the calendar month containing anchor with daily
// divisions. Month boundaries use anchor's location.
func MonthView(anchor time.Time) ViewSpec {
	year, month, _ := anchor.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, anchor.Location())
	end := start.AddDate(0, 1, 0)
	ticks := make([]Tick, 0, 32)
	for at := start; at.Before(end); at = at.AddDate(0, 0, 1) {
		ticks = append(ticks, Tick{At: at, Label: at.Format("02.01.")})
	}
	ticks = append(ticks, Tick{At: end})
	return ViewSpec{Range: TimeRange{Start: start, End: end}, Ticks: ticks, kind: viewMonth}
}

// YearView returns the calendar year containing anchor with monthly
// divisions. Year boundaries use anchor's location.
func YearView(anchor time.Time) ViewSpec {
	year, _, _ := anchor.Date()
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, anchor.Location())
	end := start.AddDate(1, 0, 0)
	ticks := make([]Tick, 0, 13)
	for at := start; at.Before(end); at = at.AddDate(0, 1, 0) {
		ticks = append(ticks, Tick{At: at, Label: at.Format("01")})
	}
	ticks = append(ticks, Tick{At: end})
	return ViewSpec{Range: TimeRange{Start: start, End: end}, Ticks: ticks, kind: viewYear}
}

func startOfDay(at time.Time) time.Time {
	year, month, day := at.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, at.Location())
}
