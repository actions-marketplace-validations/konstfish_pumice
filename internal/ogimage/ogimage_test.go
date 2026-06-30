package ogimage

import (
	"os"
	"testing"
)

func TestRenderDimensions(t *testing.T) {
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cards := []Card{
		{Site: "konst.fish", Title: "Hello World", Tags: []string{"go"}, ReadingTime: 3},
		{Site: "konst.fish", Title: "No tags here"},
		{Site: "konst.fish", Title: "A very long title that will certainly need to wrap onto several lines and then be truncated for sure absolutely", Tags: []string{"a", "b", "c"}, Date: "01-01-2026", ReadingTime: 20},
		{}, // fully empty card must not panic
	}

	for i, c := range cards {
		img := g.Render(c)
		b := img.Bounds()
		if b.Dx() != Width || b.Dy() != Height {
			t.Errorf("card %d: got %dx%d, want %dx%d", i, b.Dx(), b.Dy(), Width, Height)
		}
	}
}

// TestWritePreviews emits sample PNGs when OG_PREVIEW_DIR is set, for manual
// visual inspection. Skipped in normal test runs.
func TestWritePreviews(t *testing.T) {
	dir := os.Getenv("OG_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set OG_PREVIEW_DIR to emit preview images")
	}
	g, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	previews := map[string]Card{
		"og_article.png": {
			Site:        "konst.fish",
			Title:       "Building a Static Site Generator in Go",
			Tags:        []string{"go", "web"},
			Date:        "30-06-2026",
			ReadingTime: 5,
		},
		"og_long.png": {
			Site:        "konst.fish",
			Title:       "A Really Quite Long Title That Should Wrap Across Multiple Lines And Then Get Truncated With An Ellipsis Eventually For Sure",
			Tags:        []string{"meta", "testing", "layout"},
			ReadingTime: 12,
		},
		"og_minimal.png": {
			Site:  "konst.fish",
			Title: "Tags",
		},
	}
	for name, c := range previews {
		if err := g.Save(c, dir+"/"+name); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
	}
}
