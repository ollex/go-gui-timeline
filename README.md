# Go GUI Timeline

`go-gui-timeline` is an independently maintained resource timeline widget for
[Go GUI](https://github.com/go-gui-org/go-gui). It displays time intervals
across resource rows and supports overlapping events, calendar views, scrolling,
selection, hit testing, and tooltips.

The project is currently in early development. Its API may change before the
first stable release.

## Features

- Day, week, month, year, and custom-range views
- Configurable Monday, Sunday, or other starts of the week
- Automatic lane packing for overlapping events
- Events clipped correctly at visible range boundaries
- Fixed time and resource headers
- Standard horizontal and vertical scrollbars
- Event click callbacks and selected-event styling
- Tooltips when event titles do not fit
- Per-event fill and text colors
- Cached layout, drawing, and indexed hit testing
- Application-owned data loading and resource pagination

## Installation

```sh
go get github.com/ollex/go-gui-timeline
```

## Basic usage

```go
package schedule

import (
	"time"

	"github.com/go-gui-org/go-gui/gui"
	"github.com/ollex/go-gui-timeline"
)

func View(w *gui.Window) gui.View {
	day := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.Local)
	room := timeline.ResourceID("room-a")

	return timeline.New(w, timeline.Config{
		ID:             "schedule",
		ResourceHeader: "Room",
		Resources: []timeline.Resource{
			{ID: room, Label: "Studio A"},
		},
		Events: []timeline.Event{
			{
				ID:         "planning",
				ResourceID: room,
				Start:      day.Add(9 * time.Hour),
				End:        day.Add(11 * time.Hour),
				Title:      "Planning session",
			},
		},
		View:         timeline.DayView(day),
		Width:        960,
		Height:       360,
		ContentWidth: 1440,
		OnEventClick: func(_ gui.EventCtx, event timeline.Event) {
			// Use event.ID to open details or update application state.
		},
	})
}
```

The application owns the resource and event data. Update that state using the
normal Go GUI redraw mechanisms and pass the current slices back to the widget.
By default, the timeline detects content changes automatically. Applications
with large data sets can set `ContentVersion` and increment it whenever a
resource or event changes to avoid hashing all content on cached frames.

## Running the example

```sh
go run ./examples/timeline
```

The example simulates asynchronous range-based loading and resource pagination.
Its large-data mode starts with 200 resources and 24,000 events:

```sh
go run ./examples/timeline -large
```

## Scope

The widget deliberately does not prescribe database access, event editing,
dragging, resource pagination, or navigation controls. Applications decide how
to load data and what an event click means; the timeline concentrates on
calendar geometry, rendering, scrolling, and interaction.

## Development

Run all library and example tests with:

```sh
go test ./...
```

Benchmarks can be run with:

```sh
go test -run '^$' -bench . -benchmem ./...
```

When developing beside a local Go GUI checkout, use an uncommitted `go.work`
file instead of adding a local `replace` directive to `go.mod`.

## License

[MIT](LICENSE)
