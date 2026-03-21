package processor

import "strings"

// AddLazyLoading adds loading="lazy" to all <img> tags that don't already have it.
func AddLazyLoading(html string) string {
	var result strings.Builder
	result.Grow(len(html))

	for {
		idx := strings.Index(html, "<img ")
		if idx == -1 {
			result.WriteString(html)
			break
		}

		// Find end of this tag
		endIdx := strings.Index(html[idx:], ">")
		if endIdx == -1 {
			result.WriteString(html)
			break
		}
		endIdx += idx

		tag := html[idx : endIdx+1]

		// Skip if already has loading attribute
		if !strings.Contains(tag, "loading=") {
			tag = strings.Replace(tag, "<img ", `<img loading="lazy" `, 1)
		}

		result.WriteString(html[:idx])
		result.WriteString(tag)
		html = html[endIdx+1:]
	}

	return result.String()
}
