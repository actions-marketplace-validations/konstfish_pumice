package processor

import (
	"strings"
	"testing"
)

func TestEstimateReadingTime(t *testing.T) {
	tests := []struct {
		name     string
		words    int
		expected int
	}{
		{"empty", 0, 0},
		{"short", 50, 1},
		{"one minute", 238, 1},
		{"two minutes", 400, 2},
		{"long post", 2380, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := strings.Repeat("word ", tt.words)
			got := estimateReadingTime(text)
			if got != tt.expected {
				t.Errorf("estimateReadingTime(%d words) = %d, want %d", tt.words, got, tt.expected)
			}
		})
	}
}
