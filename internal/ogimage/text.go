package ogimage

import "strings"

// measureFunc reports the rendered width of a string in pixels. It is injected
// so the wrapping logic can be unit-tested without a real font face.
type measureFunc func(string) float64

// wrapLines greedily wraps text to at most maxWidth pixels per line, capped at
// maxLines. When the text doesn't fit, the final line is ellipsized so the
// truncation is visible. A single word wider than maxWidth is kept on its own
// line rather than dropped.
func wrapLines(measure measureFunc, text string, maxWidth float64, maxLines int) []string {
	text = strings.TrimSpace(text)
	if text == "" || maxLines <= 0 {
		return nil
	}

	words := strings.Fields(text)
	var lines []string
	cur := ""

	for _, w := range words {
		cand := w
		if cur != "" {
			cand = cur + " " + w
		}

		if measure(cand) <= maxWidth || cur == "" {
			cur = cand
			continue
		}

		// Current word doesn't fit — commit the line we have.
		lines = append(lines, cur)
		cur = w

		if len(lines) == maxLines {
			// No room left but words remain: mark the last line truncated.
			lines[maxLines-1] = ellipsize(measure, lines[maxLines-1], maxWidth)
			return lines
		}
	}

	if cur != "" {
		if len(lines) < maxLines {
			lines = append(lines, cur)
		} else {
			lines[maxLines-1] = ellipsize(measure, lines[maxLines-1], maxWidth)
		}
	}
	return lines
}

// ellipsize trims s until it plus a trailing ellipsis fits within maxWidth.
func ellipsize(measure measureFunc, s string, maxWidth float64) string {
	s = strings.TrimRight(strings.TrimSpace(s), " ")
	if measure(s+"…") <= maxWidth {
		return s + "…"
	}
	for len(s) > 0 {
		s = strings.TrimRight(s[:len(s)-1], " ")
		if measure(s+"…") <= maxWidth {
			return s + "…"
		}
	}
	return "…"
}

// truncateToWidth shortens a single line (no wrapping) to fit maxWidth,
// ellipsizing if needed. Used for the bottom-row tag list.
func truncateToWidth(measure measureFunc, s string, maxWidth float64) string {
	if measure(s) <= maxWidth {
		return s
	}
	return ellipsize(measure, s, maxWidth)
}
