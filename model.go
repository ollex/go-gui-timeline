// Package timeline provides a resource timeline widget and its layout engine.
package timeline

import (
	"time"

	gg "github.com/go-gui-org/go-gui/gui"
)

// ResourceID uniquely identifies a resource in a timeline.
type ResourceID string

// EventID uniquely identifies an event in a timeline.
type EventID string

// Resource is one row in a timeline.
type Resource struct {
	ID    ResourceID
	Label string
}

// Event is a time interval assigned to a resource.
type Event struct {
	ID         EventID
	ResourceID ResourceID
	Start      time.Time
	End        time.Time
	Title      string
	// Color controls the event fill. Its zero value uses the timeline's
	// default event color.
	Color gg.Color
	// TextColor controls the event title. Its zero value uses the
	// timeline's default event text color.
	TextColor gg.Color
}

// TimeRange is the half-open interval displayed by a timeline.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// LayoutConfig contains the dimensions and time range needed to build a Scene.
type LayoutConfig struct {
	Range      TimeRange
	Width      float32
	LaneHeight float32
	EventInset float32
}

// Rect describes a rectangle in timeline content coordinates.
type Rect struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

// Contains reports whether the half-open rectangle contains the point.
func (r Rect) Contains(x, y float32) bool {
	return x >= r.X && x < r.X+r.Width &&
		y >= r.Y && y < r.Y+r.Height
}

// EventLayout is the derived geometry for one visible event.
type EventLayout struct {
	ID            EventID
	ResourceID    ResourceID
	SourceIndex   int
	ResourceIndex int
	Lane          int
	Rect          Rect
}

// ResourceLayout is the derived geometry for one resource row.
type ResourceLayout struct {
	ID          ResourceID
	SourceIndex int
	LaneCount   int
	Rect        Rect
	Events      []EventLayout
}

// Scene is the derived geometry shared by drawing and hit testing.
type Scene struct {
	Range          TimeRange
	Width          float32
	Height         float32
	Rows           []ResourceLayout
	rowBottoms     []float32
	eventLocations []eventLocation
	scale          TimeScale
}

// eventLocation maps an index in the source Events slice to its derived
// geometry. rowIndexPlusOne uses zero as the sentinel for an event outside the
// visible time range.
type eventLocation struct {
	rowIndexPlusOne int
	eventIndex      int
}

// Hit identifies an event found at a point in the scene.
type Hit struct {
	EventID       EventID
	ResourceID    ResourceID
	EventIndex    int
	ResourceIndex int
}
