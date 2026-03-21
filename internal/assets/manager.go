package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ui "github.com/konstfish/ui/core"
)

type Manager struct {
	buildDir      string
	staticDir     string
	basePath      string
	fileCollector FileCollectorInterface
}

type FileCollectorInterface interface {
	IsFileReferenced(path string) bool
}

func NewManager(buildDir, staticDir, basePath string, fileCollector FileCollectorInterface) *Manager {
	return &Manager{
		buildDir:      buildDir,
		staticDir:     staticDir,
		basePath:      basePath,
		fileCollector: fileCollector,
	}
}

func (m *Manager) CopyStaticAssets() error {
	if _, err := os.Stat(m.staticDir); os.IsNotExist(err) {
		return nil
	}

	pumiceDir := filepath.Join(m.buildDir, "_pumice")
	if err := os.MkdirAll(pumiceDir, 0755); err != nil {
		return fmt.Errorf("creating _pumice directory: %w", err)
	}

	return filepath.WalkDir(m.staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(m.staticDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}

		outputPath := filepath.Join(pumiceDir, relPath)
		outputDir := filepath.Dir(outputPath)

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", outputDir, err)
		}

		if err := m.copyFile(path, outputPath); err != nil {
			return err
		}

		fmt.Printf("Copied static: %s\n", outputPath)
		return nil
	})
}

func (m *Manager) AddStaticAssetsToPage(page *ui.Page) error {
	if _, err := os.Stat(m.staticDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(m.staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(m.staticDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}
		assetPath := m.basePath + "/_pumice/" + relPath

		if strings.HasSuffix(path, ".css") {
			page.AddStyleSheet(assetPath)
		} else if strings.HasSuffix(path, ".js") {
			page.AddScript(assetPath)
		}

		return nil
	})
}

func (m *Manager) CopyContentFileIfReferenced(contentDir, path string) error {
	filename := filepath.Base(path)
	if filename == "pumice.yaml" {
		return nil
	}

	if !m.fileCollector.IsFileReferenced(path) {
		return nil
	}

	relPath, err := filepath.Rel(contentDir, path)
	if err != nil {
		return fmt.Errorf("getting relative path for %s: %w", path, err)
	}

	outputPath := filepath.Join(m.buildDir, relPath)
	outputDir := filepath.Dir(outputPath)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	if err := m.copyFile(path, outputPath); err != nil {
		return err
	}

	fmt.Printf("Copied: %s\n", outputPath)
	return nil
}

func (m *Manager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying file %s: %w", src, err)
	}

	return nil
}