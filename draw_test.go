package timeline

import (
	"testing"
	"unicode/utf8"

	gg "github.com/go-gui-org/go-gui/gui"
)

func TestEventColor(t *testing.T) {
	t.Parallel()

	if got := eventColor(Event{}); got != defaultEventColor {
		t.Errorf("default color = %v, want %v", got, defaultEventColor)
	}
	if got := eventColor(Event{Color: gg.Red}); got != gg.Red {
		t.Errorf("event color = %v, want red", got)
	}
}

func TestEventTextColor(t *testing.T) {
	t.Parallel()

	fallback := gg.White
	if got := eventTextColor(Event{}, fallback); got != fallback {
		t.Errorf("default text color = %v, want %v", got, fallback)
	}
	if got := eventTextColor(Event{TextColor: gg.Black}, fallback); got != gg.Black {
		t.Errorf("event text color = %v, want black", got)
	}
}

func TestFitTextWithEllipsis(t *testing.T) {
	t.Parallel()

	measure := func(text string) float32 {
		return float32(utf8.RuneCountInString(text) * 10)
	}
	tests := []struct {
		name     string
		text     string
		maxWidth float32
		want     string
	}{
		{name: "fits", text: "Review", maxWidth: 60, want: "Review"},
		{name: "exact prefix and ellipsis", text: "Morning", maxWidth: 50, want: "Morn…"},
		{name: "trims space before ellipsis", text: "Morning briefing", maxWidth: 90, want: "Morning…"},
		{name: "unicode rune boundary", text: "Café meeting", maxWidth: 50, want: "Café…"},
		{name: "ellipsis only", text: "Meeting", maxWidth: 10, want: "…"},
		{name: "too narrow", text: "Meeting", maxWidth: 9, want: ""},
		{name: "no room", text: "Meeting", maxWidth: 0, want: ""},
		{name: "empty", text: "", maxWidth: 100, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := fitTextWithEllipsis(test.text, test.maxWidth, measure)
			if got != test.want {
				t.Errorf("fitTextWithEllipsis(%q, %v) = %q, want %q",
					test.text, test.maxWidth, got, test.want)
			}
			if got != "" && measure(got) > test.maxWidth {
				t.Errorf("result width %v exceeds maximum %v", measure(got), test.maxWidth)
			}
		})
	}
}

func TestBoxFullyVisible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		y, height, top, bot float32
		want                bool
	}{
		{name: "inside", y: 20, height: 10, top: 20, bot: 40, want: true},
		{name: "touches bottom", y: 30, height: 10, top: 20, bot: 40, want: true},
		{name: "crosses header", y: 19, height: 10, top: 20, bot: 40},
		{name: "crosses bottom", y: 31, height: 10, top: 20, bot: 40},
		{name: "zero height", y: 20, height: 0, top: 20, bot: 40},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := boxFullyVisible(test.y, test.height, test.top, test.bot)
			if got != test.want {
				t.Errorf("boxFullyVisible(%v, %v, %v, %v) = %v, want %v",
					test.y, test.height, test.top, test.bot, got, test.want)
			}
		})
	}
}

func TestBoxIntersects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		y, height, top, bot float32
		want                bool
	}{
		{name: "inside", y: 20, height: 10, top: 20, bot: 40, want: true},
		{name: "crosses top", y: 15, height: 10, top: 20, bot: 40, want: true},
		{name: "crosses bottom", y: 35, height: 10, top: 20, bot: 40, want: true},
		{name: "ends at top", y: 10, height: 10, top: 20, bot: 40},
		{name: "starts at bottom", y: 40, height: 10, top: 20, bot: 40},
		{name: "zero height", y: 20, height: 0, top: 20, bot: 40},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := boxIntersects(test.y, test.height, test.top, test.bot)
			if got != test.want {
				t.Errorf("boxIntersects(%v, %v, %v, %v) = %v, want %v",
					test.y, test.height, test.top, test.bot, got, test.want)
			}
		})
	}
}
