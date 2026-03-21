package processor

import (
	"regexp"
	"strings"
)

var linkRegex = regexp.MustCompile(`<a\s+href="([^"]*)"`)

// AnnotateInternalLinks adds htmx attributes and data-internal to internal links
// in rendered HTML. External links (http, //, #, mailto, javascript) are skipped.
func AnnotateInternalLinks(html string) string {
	return linkRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatch := linkRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		href := submatch[1]

		// Skip external and non-navigational links
		if strings.HasPrefix(href, "http") ||
			strings.HasPrefix(href, "//") ||
			strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "javascript:") {
			return match
		}

		// Skip file links (images, pdfs, etc.)
		for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".pdf", ".zip", ".xml"} {
			if strings.HasSuffix(strings.ToLower(href), ext) {
				return match
			}
		}

		// hx-get needs .html suffix for static file fetching
		hxGet := href
		if !strings.HasSuffix(hxGet, ".html") {
			hxGet = hxGet + ".html"
		}

		return `<a href="` + href + `"` +
			` data-internal` +
			` hx-get="` + hxGet + `"` +
			` hx-target="#content"` +
			` hx-select="#content"` +
			` hx-swap="outerHTML transition:true"` +
			` hx-push-url="` + href + `"`
	})
}
