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
- Keyboard event navigation and a single active-event accessibility proxy
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
	"fmt"
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
		A11YLabel:    "Studio schedule",
		A11YDescription: "Use the arrow keys to move between events. " +
			"Press Enter or Space to open an event.",
		EventA11YLabel: func(event timeline.Event, resource timeline.Resource) string {
			// The application owns localization, time zones, and terminology.
			return fmt.Sprintf("%s, %s, %s to %s",
				event.Title, resource.Label,
				event.Start.Format("15:04"), event.End.Format("15:04"))
		},
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

## Accessibility

The timeline keeps canvas rendering for performance and exposes one transparent
semantic element for the active event. Tab moves focus into or out of that
element. Left and Right move chronologically within a resource; Up and Down
move to the nearest event in another resource; Home and End move to the first
or last event in the resource. Ctrl+Home and Ctrl+End move to the first or last
event in the loaded scene. Enter and Space invoke `OnEventClick`.

Keyboard focus is drawn on the canvas and the active event is automatically
scrolled into view. Painted canvases are omitted from the accessibility tree so
they do not duplicate the semantic event and scroll area.

`EventA11YLabel` should return the complete event announcement in the user's
language. When it is nil, the event title is used as a minimal fallback. The
application is responsible for localized dates, times, time zones, resource
terminology, and any other event state that users need to hear.

Screen-reader exposure depends on the selected Go GUI backend. In the pinned Go
GUI v0.86.0 release, the native Windows GL backend does not yet publish its
accessibility tree through Windows UI Automation, so Narrator cannot read these
nodes. The keyboard interaction and visual focus behavior still work on
Windows. Use the WebAssembly backend in a browser to exercise the semantic tree
on Windows until a release containing the native UI Automation bridge is
available.

## Running the example

```sh
go run ./examples/timeline
```

The example simulates asynchronous range-based loading and resource pagination.
Its large-data mode starts with 200 resources and 24,000 events:

```sh
go run ./examples/timeline -large
```

### Browser accessibility test on Windows

Go GUI v0.86.0's native Windows backend does not yet expose its controls to
Narrator. To exercise the WebAssembly backend's ARIA tree instead, run this from
the project root. If local scripts are already allowed:

```powershell
.\examples\timeline\run-web.ps1
```

If PowerShell's execution policy is `Restricted`, allow locally created scripts
for only the current PowerShell process, then run it:

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy RemoteSigned
.\examples\timeline\run-web.ps1
```

This does not use `Bypass`, does not require administrator privileges, and the
policy change disappears when that PowerShell window closes.

Leave that terminal open and visit <http://127.0.0.1:8080/> in Edge. Start or
stop Narrator with `Windows+Ctrl+Enter`. Once the timeline has loaded, use Tab
until Narrator announces "Timeline application"; clicking the painted canvas
does not move Narrator to its semantic counterpart. Continue with Tab to reach
the active timeline event. Use the arrow keys to change events and Enter or
Space to select one. Press `Ctrl+C` in the terminal to stop the server.

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

The project intentionally has no `go.work` file. Its Go GUI version is selected
by `go.mod`, like it will be for downstream users and CI builds.

## License

[MIT](LICENSE)
