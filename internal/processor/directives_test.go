package processor

import (
	"strings"
	"testing"
)

func TestMarkListingDirectives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "bare directive",
			input:    `{{listing:blog}}`,
			contains: `<!--pumice:listing:blog-->`,
		},
		{
			name:     "wrapped in p",
			input:    `<p>{{listing:blog/writing}}</p>`,
			contains: `<!--pumice:listing:blog/writing-->`,
		},
		{
			name:     "no directive unchanged",
			input:    `<p>just text</p>`,
			contains: `<p>just text</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MarkListingDirectives(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected %q in result, got: %s", tt.contains, result)
			}
		})
	}
}
