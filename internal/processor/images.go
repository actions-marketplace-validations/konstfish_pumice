package processor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var htmlImgSrcRegex = regexp.MustCompile(`(<(?:img|source|video|audio)\s[^>]*src=["'])([^"']+)(["'])`)

// ResolveImagePaths fixes bare image filenames in raw HTML by searching for
// them in the content directory. Obsidian resolves these by vault search,
// but the generated HTML needs correct relative paths.
func ResolveImagePaths(html, markdownFile, contentDir, outputDir string) string {
	mdDir := filepath.Dir(markdownFile)

	return htmlImgSrcRegex.ReplaceAllStringFunc(html, func(match string) string {
		sub := htmlImgSrcRegex.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}

		prefix := sub[1]
		src := sub[2]
		suffix := sub[3]

		// Skip external URLs and already-correct paths
		if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "//") || strings.HasPrefix(src, "/") {
			return match
		}

		// Check if the file exists at the direct relative path
		directPath := filepath.Join(mdDir, src)
		if _, err := os.Stat(directPath); err == nil {
			return match // Already correct
		}

		// Search for the file in the content directory
		filename := filepath.Base(src)
		var foundPath string
		filepath.WalkDir(contentDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == filename {
				foundPath = path
				return filepath.SkipAll
			}
			return nil
		})

		if foundPath == "" {
			return match // Not found, leave as-is
		}

		// Compute relative path from the output HTML file to where the
		// found file will be copied in the build dir
		relFromContent, err := filepath.Rel(contentDir, foundPath)
		if err != nil {
			return match
		}

		// The file will be at buildDir/relFromContent
		// The HTML page is at outputDir
		// We need the relative path from outputDir to buildDir/relFromContent
		// Content dir structure mirrors build dir for assets.
		// relFromContent = "self/_attachments/server.svg"
		// mdDir relative to contentDir = "self"
		// So from "self/" the relative path is "_attachments/server.svg"
		mdRelDir, err := filepath.Rel(contentDir, mdDir)
		if err != nil {
			return match
		}

		// Target relative to content root
		// Page output is in the same relative dir
		// So relative path from page to file = relative from mdRelDir to relFromContent
		relPath, err := filepath.Rel(mdRelDir, relFromContent)
		if err != nil {
			return match
		}

		return prefix + relPath + suffix
	})
}
