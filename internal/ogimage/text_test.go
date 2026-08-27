package ogimage

import (
	"strings"
	"testing"
)

// fixed-width measure: every rune is 10px wide. Makes widths predictable.
func fixedMeasure(s string) float64 { return float64(len([]rune(s))) * 10 }

func TestWrapLinesBasic(t *testing.T) {
	lines := wrapLines(fixedMeasure, "one two three four five", 100, 3)
	// 100px = 10 chars per line, including spaces.
	want := []string{"one two", "three four", "five"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q, want %q", lines, want)
	}
}

func TestWrapLinesTruncatesWithEllipsis(t *testing.T) {
	lines := wrapLines(fixedMeasure, "alpha beta gamma delta epsilon zeta", 60, 2)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), lines)
	}
	if !strings.HasSuffix(lines[1], "…") {
		t.Errorf("expected last line to be ellipsized, got %q", lines[1])
	}
	for i, l := range lines {
		if fixedMeasure(l) > 60 {
			t.Errorf("line %d %q exceeds max width", i, l)
		}
	}
}

func TestWrapLinesEmpty(t *testing.T) {
	if got := wrapLines(fixedMeasure, "   ", 100, 3); got != nil {
		t.Errorf("expected nil for blank text, got %q", got)
	}
}

func TestWrapLinesOverlongWordKept(t *testing.T) {
	lines := wrapLines(fixedMeasure, "supercalifragilistic", 50, 2)
	if len(lines) != 1 || lines[0] != "supercalifragilistic" {
		t.Errorf("overlong single word should be kept on its own line, got %q", lines)
	}
}

func TestTruncateToWidth(t *testing.T) {
	if got := truncateToWidth(fixedMeasure, "short", 100); got != "short" {
		t.Errorf("fitting string should be unchanged, got %q", got)
	}
	got := truncateToWidth(fixedMeasure, "this is far too long to fit", 100)
	if !strings.HasSuffix(got, "…") || fixedMeasure(got) > 100 {
		t.Errorf("expected ellipsized result within width, got %q (%v)", got, fixedMeasure(got))
	}
}
