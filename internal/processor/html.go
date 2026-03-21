package processor

import "html"

// EscapeHTML escapes special HTML characters in a string.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}
