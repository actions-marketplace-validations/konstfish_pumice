package sitebuilder

import (
	"strings"
	"testing"
)

func TestRenderListingHidesTagsMatchingListingFolder(t *testing.T) {
	sb := &SiteBuilder{basePath: ""}
	entries := []pageEntry{{
		Title:    "A post",
		HtmlPath: "blog/a-post.html",
		Dir:      "blog",
		Tags:     []string{"blog", "kubernetes"},
	}}

	html := sb.renderListing("blog", entries)

	if strings.Contains(html, `<code class="page-tag">blog</code>`) {
		t.Fatalf("expected listing to hide tag matching folder name, got %s", html)
	}
	if !strings.Contains(html, `<code class="page-tag">kubernetes</code>`) {
		t.Fatalf("expected non-matching tag to remain, got %s", html)
	}
}

func TestRenderListingHidesMatchingTagBySlug(t *testing.T) {
	sb := &SiteBuilder{basePath: ""}
	entries := []pageEntry{{
		Title:    "A post",
		HtmlPath: "open-source/a-post.html",
		Dir:      "open-source",
		Tags:     []string{"Open Source", "go"},
	}}

	html := sb.renderListing("open-source", entries)

	if strings.Contains(html, `<code class="page-tag">Open Source</code>`) {
		t.Fatalf("expected listing to hide tag matching folder slug, got %s", html)
	}
	if !strings.Contains(html, `<code class="page-tag">go</code>`) {
		t.Fatalf("expected non-matching tag to remain, got %s", html)
	}
}
