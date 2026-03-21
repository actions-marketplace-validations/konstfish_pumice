package processor

import (
	"strings"
	"testing"
)

func TestProcessCallouts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:  "note callout with title",
			input: "<blockquote>\n<p>[!note] Important\n<br />\ncontent here</p>\n</blockquote>",
			contains: []string{
				`class="callout callout-note"`,
				`class="callout-title">Important</div>`,
				"content here",
			},
			excludes: []string{"<blockquote>", "[!note]"},
		},
		{
			name:  "warning callout default title",
			input: "<blockquote>\n<p>[!warning]<br />\nbe careful</p>\n</blockquote>",
			contains: []string{
				`callout-warning`,
				`callout-title">Warning</div>`,
			},
		},
		{
			name:  "regular blockquote unchanged",
			input: "<blockquote>\n<p>just a regular quote</p>\n</blockquote>",
			contains: []string{
				"<blockquote>",
				"just a regular quote",
				"</blockquote>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessCallouts(tt.input)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in result, got:\n%s", s, result)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("did not expect %q in result, got:\n%s", s, result)
				}
			}
		})
	}
}
