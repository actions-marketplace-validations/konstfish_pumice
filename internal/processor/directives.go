package processor

import "regexp"

// listingDirectiveRegex matches {{listing:path}} or {{listing:path:limit}} directives.
// Goldmark wraps them in <p> tags: <p>{{listing:blog:5}}</p>
var listingDirectiveRegex = regexp.MustCompile(`(?:<p>)?\{\{listing:([^}]+)\}\}(?:</p>)?`)

// ListingPlaceholder is the prefix used to identify listing placeholders in HTML.
const ListingPlaceholder = `<!--pumice:listing:`

// MarkListingDirectives replaces {{listing:path}} or {{listing:path:5}} with
// HTML comment placeholders that the sitebuilder resolves with actual listings.
func MarkListingDirectives(html string) string {
	return listingDirectiveRegex.ReplaceAllStringFunc(html, func(match string) string {
		sub := listingDirectiveRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		// sub[1] is "blog" or "blog:5" — pass through as-is
		return ListingPlaceholder + sub[1] + `-->`
	})
}
