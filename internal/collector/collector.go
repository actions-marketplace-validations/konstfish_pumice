package collector

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konstfish/pumice/internal/slug"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type FileCollector struct {
	contentDir      string
	referencedFiles map[string]bool
	titleMapping    map[string]string
}

func NewFileCollector(contentDir string) *FileCollector {
	return &FileCollector{
		contentDir:      contentDir,
		referencedFiles: make(map[string]bool),
		titleMapping:    make(map[string]string),
	}
}

func (fc *FileCollector) FindFileInContent(filename string) string {
	baseName := filepath.Base(filename)
	var foundPath string
	filepath.WalkDir(fc.contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Base(path) == baseName {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	return foundPath
}

func (fc *FileCollector) BuildTitleMapping() error {
	fc.titleMapping = make(map[string]string)

	err := filepath.WalkDir(fc.contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", path, err)
		}

		context := parser.NewContext()
		goldmark.New(goldmark.WithExtensions(meta.Meta)).Parser().Parse(text.NewReader(content), parser.WithContext(context))
		metaData := meta.Get(context)

		relPath, err := filepath.Rel(fc.contentDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}

		baseName := strings.TrimSuffix(filepath.Base(relPath), ".md")
		slugifiedName := slug.Slugify(baseName)
		htmlPath := filepath.Join(filepath.Dir(relPath), slugifiedName)

		if title, ok := metaData["title"]; ok {
			if titleStr, ok := title.(string); ok {
				fc.titleMapping[titleStr] = htmlPath
			}
		}

		return nil
	})

	return err
}

func (fc *FileCollector) GetTitleMapping() map[string]string {
	return fc.titleMapping
}

func (fc *FileCollector) CollectReferencedFiles() error {
	fc.referencedFiles = make(map[string]bool)

	return filepath.WalkDir(fc.contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", path, err)
		}

		fc.extractReferences(string(content), filepath.Dir(path))
		return nil
	})
}

func (fc *FileCollector) IsFileReferenced(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return fc.referencedFiles[absPath]
}

func (fc *FileCollector) extractReferences(content, baseDir string) {
	// Markdown images: ![alt](path)
	imgRegex := regexp.MustCompile(`!\[.*?\]\(([^)]+)\)`)
	matches := imgRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "http") {
			fc.addReference(baseDir, match[1])
		}
	}

	// Wikilink embeds: ![[file]] or ![[file|size]]
	wikiImgRegex := regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	matches = wikiImgRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			target := stripWikilinkLabel(match[1])
			fc.addReferenceWithSearch(baseDir, target)
		}
	}

	// Markdown links: [text](path)
	linkRegex := regexp.MustCompile(`\[.*?\]\(([^)]+)\)`)
	matches = linkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "http") && !strings.HasPrefix(match[1], "#") {
			fc.addReference(baseDir, match[1])
		}
	}

	// Wikilinks: [[target]] or [[target|display]]
	wikiLinkRegex := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches = wikiLinkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "!") {
			target := stripWikilinkLabel(match[1])
			fc.addReferenceWithSearch(baseDir, target)
		}
	}

	// Raw HTML: <img src="path">, <source src="path">, etc.
	htmlSrcRegex := regexp.MustCompile(`(?i)<(?:img|source|video|audio)\s[^>]*src=["']([^"']+)["']`)
	matches = htmlSrcRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "http") && !strings.HasPrefix(match[1], "//") {
			fc.addReferenceWithSearch(baseDir, match[1])
		}
	}
}

// stripWikilinkLabel removes the display text from a wikilink target.
// "file.pdf|Display Text" → "file.pdf"
// "file.pdf" → "file.pdf"
func stripWikilinkLabel(target string) string {
	if idx := strings.Index(target, "|"); idx != -1 {
		return target[:idx]
	}
	return target
}

// addReference marks a file as referenced by resolving it relative to baseDir.
func (fc *FileCollector) addReference(baseDir, ref string) {
	refPath := filepath.Join(baseDir, ref)
	if absPath, err := filepath.Abs(refPath); err == nil {
		fc.referencedFiles[absPath] = true
	}
}

// addReferenceWithSearch marks a file as referenced, searching the content
// directory if the file isn't found relative to baseDir.
func (fc *FileCollector) addReferenceWithSearch(baseDir, ref string) {
	refPath := filepath.Join(baseDir, ref)
	if absPath, err := filepath.Abs(refPath); err == nil {
		if _, err := os.Stat(absPath); err == nil {
			fc.referencedFiles[absPath] = true
			return
		}
	}
	if foundPath := fc.FindFileInContent(ref); foundPath != "" {
		if absPath, err := filepath.Abs(foundPath); err == nil {
			fc.referencedFiles[absPath] = true
		}
	}
}

