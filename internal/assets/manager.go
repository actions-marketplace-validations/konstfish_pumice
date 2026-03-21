package assets

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstfish/pumice/internal/static"
	ui "github.com/konstfish/ui/core"
)

type Manager struct {
	buildDir      string
	basePath      string
	fileCollector FileCollectorInterface
}

type FileCollectorInterface interface {
	IsFileReferenced(path string) bool
}

func NewManager(buildDir, basePath string, fileCollector FileCollectorInterface) *Manager {
	return &Manager{
		buildDir:      buildDir,
		basePath:      basePath,
		fileCollector: fileCollector,
	}
}

func (m *Manager) CopyStaticAssets() error {
	pumiceDir := filepath.Join(m.buildDir, "_pumice")
	if err := os.MkdirAll(pumiceDir, 0755); err != nil {
		return fmt.Errorf("creating _pumice directory: %w", err)
	}

	return fs.WalkDir(static.Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := static.Files.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}

		outputPath := filepath.Join(pumiceDir, path)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", outputPath, err)
		}

		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", outputPath, err)
		}

		fmt.Printf("Copied static: %s\n", outputPath)
		return nil
	})
}

func (m *Manager) AddStaticAssetsToPage(page *ui.Page) error {
	return fs.WalkDir(static.Files, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		assetPath := m.basePath + "/_pumice/" + path

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
