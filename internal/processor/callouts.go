package processor

import (
	"regexp"
	"strings"
)

// calloutRegex matches Obsidian-style callouts inside blockquotes.
// Captures: [!type] optional title
var calloutRegex = regexp.MustCompile(
	`<blockquote>\s*<p>\[!(\w+)\]([^\n<]*)?(?:<br\s*/?>)?\s*`,
)

// TransformCallouts converts Obsidian-style callout blockquotes into styled divs.
//
// Input (goldmark output):
//
//	<blockquote><p>[!note] Title<br />
//	content</p></blockquote>
//
// Output:
//
//	<div class="callout callout-note"><div class="callout-title">Title</div><p>
//	content</p></div>
func TransformCallouts(html string) string {
	return calloutRegex.ReplaceAllStringFunc(html, func(match string) string {
		sub := calloutRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}

		calloutType := strings.ToLower(sub[1])
		title := strings.TrimSpace(sub[2])
		if title == "" {
			// Default title is the type capitalized
			title = strings.ToUpper(calloutType[:1]) + calloutType[1:]
		}

		return `<div class="callout callout-` + calloutType + `"><div class="callout-title">` + title + `</div><p>`
	})
}

// closeCalloutTags replaces the closing </blockquote> for converted callouts.
func CloseCalloutTags(html string) string {
	// After TransformCallouts, converted callouts start with <div class="callout
	// but still have </blockquote> as their closing tag. Replace those.
	// We do a simple scan: if we see </blockquote> and the most recent
	// unclosed block is a callout div, replace it.

	// Simple approach: replace </blockquote> that follows callout content
	result := html
	for {
		idx := strings.Index(result, `<div class="callout `)
		if idx == -1 {
			break
		}

		// Find the next </blockquote> after this callout
		closeIdx := strings.Index(result[idx:], "</blockquote>")
		if closeIdx == -1 {
			break
		}
		closeIdx += idx

		result = result[:closeIdx] + "</div>" + result[closeIdx+len("</blockquote>"):]
	}

	return result
}

// ProcessCallouts applies the full callout transformation pipeline.
func ProcessCallouts(html string) string {
	html = TransformCallouts(html)
	html = CloseCalloutTags(html)
	return html
}
