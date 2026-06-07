package types

import "time"

type PageMetadata struct {
	Title             string
	Date              string
	Draft             bool
	Comments          bool
	Tags              []string
	ReadingTime       int
	SocialDescription string
}

// dateLayouts lists the frontmatter date formats pumice accepts, in priority
// order. DD-MM-YYYY is the convention used across the sample content.
var dateLayouts = []string{
	"02-01-2006", // DD-MM-YYYY
	"2006-01-02", // ISO 8601 / YYYY-MM-DD
	"01/02/2006", // MM/DD/YYYY
}

// ParseDate parses a frontmatter date string into a time.Time, trying each
// supported layout. The bool reports whether parsing succeeded.
func ParseDate(s string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
