package renderer

import "testing"

func TestParseDimensions(t *testing.T) {
	tests := []struct {
		input  string
		width  int
		height int
		ok     bool
	}{
		{"200", 200, 0, true},
		{"100x50", 100, 50, true},
		{"300X200", 300, 200, true},
		{"0", 0, 0, false},
		{"-1", 0, 0, false},
		{"abc", 0, 0, false},
		{"some alt text", 0, 0, false},
		{"200x", 0, 0, false},
		{"x200", 0, 0, false},
		{"200x0", 0, 0, false},
	}

	for _, tt := range tests {
		w, h, ok := parseDimensions(tt.input)
		if w != tt.width || h != tt.height || ok != tt.ok {
			t.Errorf("parseDimensions(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tt.input, w, h, ok, tt.width, tt.height, tt.ok)
		}
	}
}
