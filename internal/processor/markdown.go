package processor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstfish/pumice/internal/renderer"
	"github.com/konstfish/pumice/internal/slug"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkRenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/wikilink"
)

type MarkdownProcessor struct {
	goldmark goldmark.Markdown
}

type WikilinkResolverInterface interface {
	ResolveWikilink(node *wikilink.Node) ([]byte, error)
	SetCurrentContext(currentFile, outputDir string)
}

func NewMarkdownProcessor(resolver WikilinkResolverInterface) *MarkdownProcessor {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			meta.Meta,
			&mermaid.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			// Wikilink parser (priority < 200 to fire before goldmark's link parser)
			parser.WithInlineParsers(
				util.Prioritized(&wikilink.Parser{}, 199),
			),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
			// Custom wikilink renderer with Obsidian image scaling support
			goldmarkRenderer.WithNodeRenderers(
				util.Prioritized(&renderer.WikilinkRenderer{
					Resolver: resolver,
				}, 199),
			),
		),
	)

	return &MarkdownProcessor{
		goldmark: md,
	}
}

func (mp *MarkdownProcessor) ProcessFile(path, contentDir, buildDir string, resolver interface{}) (string, string, *PageMetadata, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	relPath, err := filepath.Rel(contentDir, path)
	if err != nil {
		return "", "", nil, fmt.Errorf("getting relative path for %s: %w", path, err)
	}

	baseName := strings.TrimSuffix(filepath.Base(relPath), ".md")
	slugifiedName := slug.Slugify(baseName)
	htmlPath := filepath.Join(filepath.Dir(relPath), slugifiedName) + ".html"
	outputPath := filepath.Join(buildDir, htmlPath)
	outputDir := filepath.Dir(outputPath)

	if r, ok := resolver.(WikilinkResolverInterface); ok {
		r.SetCurrentContext(path, outputDir)
	}

	// Parse with context to extract frontmatter
	ctx := parser.NewContext()
	var buf bytes.Buffer
	if err := mp.goldmark.Convert(content, &buf, parser.WithContext(ctx)); err != nil {
		return "", "", nil, fmt.Errorf("converting markdown %s: %w", path, err)
	}

	// Extract metadata from frontmatter
	pageMeta := extractMetadata(ctx, string(content))

	// Strip goldmark-mermaid's injected scripts (we handle loading/init ourselves)
	html := StripMermaidScripts(buf.String())

	// Transform Obsidian-style callouts
	html = ProcessCallouts(html)

	// Fix bare image filenames (Obsidian vault search behavior)
	html = ResolveImagePaths(html, path, contentDir, outputDir)

	// Add line-numbers class to code blocks
	html = AnnotateCodeBlocks(html)

	// Mark {{listing:dir}} directives as placeholders
	html = MarkListingDirectives(html)

	// Lazy load images
	html = AddLazyLoading(html)

	// Annotate internal links with htmx attributes for SPA-style navigation
	html = AnnotateInternalLinks(html)

	return html, outputPath, pageMeta, nil
}

func extractMetadata(ctx parser.Context, rawContent string) *PageMetadata {
	metaData := meta.Get(ctx)
	pm := &PageMetadata{}

	if title, ok := metaData["title"]; ok {
		if s, ok := title.(string); ok {
			pm.Title = s
		}
	}

	if date, ok := metaData["date"]; ok {
		if s, ok := date.(string); ok {
			pm.Date = s
		}
	}

	if draft, ok := metaData["draft"]; ok {
		if b, ok := draft.(bool); ok {
			pm.Draft = b
		}
	}

	if comments, ok := metaData["comments"]; ok {
		if b, ok := comments.(bool); ok {
			pm.Comments = b
		}
	}

	if desc, ok := metaData["socialDescription"]; ok {
		if s, ok := desc.(string); ok {
			pm.SocialDescription = s
		}
	}

	if tags, ok := metaData["tags"]; ok {
		if tagList, ok := tags.([]interface{}); ok {
			for _, t := range tagList {
				if s, ok := t.(string); ok {
					pm.Tags = append(pm.Tags, s)
				}
			}
		}
	}

	pm.ReadingTime = estimateReadingTime(rawContent)

	return pm
}

