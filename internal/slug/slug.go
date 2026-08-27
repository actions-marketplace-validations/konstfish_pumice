package slug

import (
	"path/filepath"
	"strings"
)

// Slugify converts a string into a URL-friendly slug.
func Slugify(text string) string {
	slug := strings.ToLower(text)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	var result strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result.WriteRune(char)
		}
	}
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}

// HiddenTagSlug returns the slug of a directory's base name, or "" for the
// root. A tag whose slug equals this is redundant inside that directory (e.g.
// the "blog" tag on pages under a "blog/" folder) and is hidden from listings
// and OG cards.
func HiddenTagSlug(dir string) string {
	cleanDir := filepath.Clean(dir)
	if cleanDir == "." || cleanDir == string(filepath.Separator) {
		return ""
	}
	return Slugify(filepath.Base(cleanDir))
}

// VisibleTags drops any tag made redundant by the containing directory name.
func VisibleTags(dir string, tags []string) []string {
	hiddenSlug := HiddenTagSlug(dir)
	if hiddenSlug == "" {
		return tags
	}

	visible := make([]string, 0, len(tags))
	for _, tag := range tags {
		if Slugify(tag) == hiddenSlug {
			continue
		}
		visible = append(visible, tag)
	}
	return visible
}
