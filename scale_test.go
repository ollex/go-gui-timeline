package timeline_test

import (
	"testing"
	"time"

	"github.com/ollex/go-gui-timeline"
)

func TestTimeScaleConvertsTimesAndCoordinates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	scale, err := timeline.NewTimeScale(timeline.TimeRange{
		Start: start,
		End:   start.Add(10 * time.Hour),
	}, 200)
	if err != nil {
		t.Fatalf("NewTimeScale: %v", err)
	}

	if got := scale.X(start.Add(5 * time.Hour)); got != 100 {
		t.Errorf("X(midpoint) = %v, want 100", got)
	}
	if got := scale.Width(start.Add(2*time.Hour), start.Add(5*time.Hour)); got != 60 {
		t.Errorf("Width(three hours) = %v, want 60", got)
	}
	if got := scale.Time(150); !got.Equal(start.Add(7*time.Hour + 30*time.Minute)) {
		t.Errorf("Time(150) = %v, want %v", got, start.Add(7*time.Hour+30*time.Minute))
	}
}

func TestNewTimeScaleRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		timeRange timeline.TimeRange
		width     float32
	}{
		{
			name:      "empty range",
			timeRange: timeline.TimeRange{Start: start, End: start},
			width:     100,
		},
		{
			name:      "backwards range",
			timeRange: timeline.TimeRange{Start: start, End: start.Add(-time.Hour)},
			width:     100,
		},
		{
			name:      "zero width",
			timeRange: timeline.TimeRange{Start: start, End: start.Add(time.Hour)},
			width:     0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := timeline.NewTimeScale(test.timeRange, test.width); err == nil {
				t.Fatal("NewTimeScale succeeded, want an error")
			}
		})
	}
}
