package processor

import "regexp"

// mermaidInitRegex matches the inline mermaid init script injected by goldmark-mermaid.
var mermaidInitRegex = regexp.MustCompile(`<script[^>]*>mermaid\.initialize\(\{[^}]*\}\);?</script>`)

// mermaidSrcRegex matches the mermaid script src tag injected by goldmark-mermaid.
var mermaidSrcRegex = regexp.MustCompile(`<script src="[^"]*mermaid[^"]*"></script>`)

// StripMermaidScripts removes goldmark-mermaid's injected script tags.
// Pumice handles mermaid loading and initialization itself.
func StripMermaidScripts(html string) string {
	html = mermaidInitRegex.ReplaceAllString(html, "")
	html = mermaidSrcRegex.ReplaceAllString(html, "")
	return html
}
