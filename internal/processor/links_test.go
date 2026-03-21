package processor

import (
	"strings"
	"testing"
)

func TestAnnotateInternalLinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:  "relative link gets annotated",
			input: `<a href="./some-page">Some Page</a>`,
			contains: []string{
				`data-internal`,
				`hx-get="./some-page.html"`,
				`hx-target="#content"`,
				`hx-select="#content"`,
				`hx-swap="outerHTML transition:true"`,
				`hx-push-url="./some-page"`,
			},
		},
		{
			name:  "parent-relative link gets annotated",
			input: `<a href="../other/page">Page</a>`,
			contains: []string{
				`data-internal`,
				`hx-get="../other/page.html"`,
			},
		},
		{
			name:     "external link not annotated",
			input:    `<a href="https://example.com">Example</a>`,
			excludes: []string{`data-internal`, `hx-get`},
		},
		{
			name:     "anchor link not annotated",
			input:    `<a href="#section">Section</a>`,
			excludes: []string{`data-internal`, `hx-get`},
		},
		{
			name:  "link already ending in .html",
			input: `<a href="./page.html">Page</a>`,
			contains: []string{
				`hx-get="./page.html"`,
			},
		},
		{
			name:  "mixed links",
			input: `<a href="./internal">Int</a> and <a href="https://ext.com">Ext</a>`,
			contains: []string{`data-internal`},
		},
		{
			name:  "bare relative path",
			input: `<a href="blog">blog</a>`,
			contains: []string{
				`data-internal`,
				`hx-get="blog.html"`,
				`hx-push-url="blog"`,
			},
		},
		{
			name:  "bare path with subdir",
			input: `<a href="blog/my-post">post</a>`,
			contains: []string{`data-internal`, `hx-get="blog/my-post.html"`},
		},
		{
			name:     "pdf link not annotated",
			input:    `<a href="file.pdf">PDF</a>`,
			excludes: []string{`data-internal`},
		},
		{
			name:     "image link not annotated",
			input:    `<a href="photo.jpg">Photo</a>`,
			excludes: []string{`data-internal`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnnotateInternalLinks(tt.input)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in result, got: %s", s, result)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("did not expect %q in result, got: %s", s, result)
				}
			}
		})
	}
}
