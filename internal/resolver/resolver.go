package resolver

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstfish/pumice/internal/slug"
	"go.abhg.dev/goldmark/wikilink"
)

type LinkResolver struct {
	titleToFile   map[string]string
	contentDir    string
	buildDir      string
	currentFile   string
	outputDir     string
	fileCollector FileCollectorInterface
}

type FileCollectorInterface interface {
	FindFileInContent(filename string) string
}

func NewLinkResolver(contentDir, buildDir string, fileCollector FileCollectorInterface) *LinkResolver {
	return &LinkResolver{
		titleToFile:   make(map[string]string),
		contentDir:    contentDir,
		buildDir:      buildDir,
		fileCollector: fileCollector,
	}
}

func (r *LinkResolver) SetTitleMapping(titleToFile map[string]string) {
	r.titleToFile = titleToFile
}

func (r *LinkResolver) SetCurrentContext(currentFile, outputDir string) {
	r.currentFile = currentFile
	r.outputDir = outputDir
}

func (r *LinkResolver) ResolveWikilink(node *wikilink.Node) ([]byte, error) {
	target := string(node.Target)

	if targetFile, ok := r.titleToFile[target]; ok {
		if r.outputDir != "" {
			targetInBuild := filepath.Join(r.buildDir, targetFile+".html")
			relPath, err := filepath.Rel(r.outputDir, targetInBuild)
			if err == nil {
				relPath = strings.TrimSuffix(relPath, ".html")

				if !strings.HasPrefix(relPath, "./") && !strings.HasPrefix(relPath, "../") {
					relPath = "./" + relPath
				}
				return []byte(relPath), nil
			}
		}
		return []byte(targetFile), nil
	}

	if r.currentFile != "" && r.outputDir != "" {
		currentDir := filepath.Dir(r.currentFile)

		refPath := filepath.Join(currentDir, target)
		if _, err := os.Stat(refPath); err == nil {
			relPathFromContent, err := filepath.Rel(r.contentDir, refPath)
			if err == nil {
				targetInBuild := filepath.Join(r.buildDir, relPathFromContent)
				relPath, err := filepath.Rel(r.outputDir, targetInBuild)
				if err == nil {
					return []byte(relPath), nil
				}
			}
		}

		if foundPath := r.fileCollector.FindFileInContent(target); foundPath != "" {
			relPathFromContent, err := filepath.Rel(r.contentDir, foundPath)
			if err == nil {
				targetInBuild := filepath.Join(r.buildDir, relPathFromContent)
				relPath, err := filepath.Rel(r.outputDir, targetInBuild)
				if err == nil {
					return []byte(relPath), nil
				}
			}
		}
	}

	if r.currentFile != "" && r.outputDir != "" {
		targetSlug := slug.Slugify(target)

		currentDir := filepath.Dir(r.currentFile)

		possibleFile := filepath.Join(currentDir, targetSlug+".md")
		if _, err := os.Stat(possibleFile); err == nil {
			return []byte("./" + targetSlug), nil
		}

		var foundPath string
		filepath.WalkDir(r.contentDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".md") {
				baseName := strings.TrimSuffix(filepath.Base(path), ".md")
				if slug.Slugify(baseName) == targetSlug {
					foundPath = path
					return filepath.SkipAll
				}
			}
			return nil
		})

		if foundPath != "" {
			relPathFromContent, err := filepath.Rel(r.contentDir, foundPath)
			if err == nil {
				dir := filepath.Dir(relPathFromContent)
				htmlPath := filepath.Join(dir, targetSlug+".html")
				targetInBuild := filepath.Join(r.buildDir, htmlPath)
				relPath, err := filepath.Rel(r.outputDir, targetInBuild)
				if err == nil {
					if strings.HasSuffix(relPath, ".html") {
						relPath = strings.TrimSuffix(relPath, ".html")
					}
					if !strings.HasPrefix(relPath, "./") && !strings.HasPrefix(relPath, "../") {
						relPath = "./" + relPath
					}
					return []byte(relPath), nil
				}
			}
		}

		return []byte("./" + targetSlug), nil
	}

	return []byte(slug.Slugify(target)), nil
}