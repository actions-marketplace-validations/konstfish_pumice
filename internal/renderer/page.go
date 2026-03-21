package renderer

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/konstfish/pumice/internal/config"
	"github.com/konstfish/pumice/internal/types"
	ui "github.com/konstfish/ui/core"
	"github.com/konstfish/ui/themes/kf"
)

type PageRenderer struct {
	configManager ConfigManagerInterface
	assetManager  AssetManagerInterface
}

type ConfigManagerInterface interface {
	GetPageTitle() string
	GetSiteURL() string
	GetBasePath() string
	GetOGImage() string
	GetFavicon() string
	GetFooter() config.FooterConfig
	GetGiscus() config.GiscusConfig
}

type AssetManagerInterface interface {
	AddStaticAssetsToPage(page *ui.Page) error
}

func NewPageRenderer(configManager ConfigManagerInterface, assetManager AssetManagerInterface) *PageRenderer {
	return &PageRenderer{
		configManager: configManager,
		assetManager:  assetManager,
	}
}

func (pr *PageRenderer) RenderPage(htmlContent, outputPath string, meta *types.PageMetadata) error {
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	pageSlug := strings.TrimSuffix(filepath.Base(outputPath), ".html")

	// Use frontmatter title if available, otherwise fall back to slug
	displayTitle := pageSlug
	if meta != nil && meta.Title != "" {
		displayTitle = meta.Title
	}

	// Browser tab title
	tabTitle := displayTitle
	if pr.configManager.GetPageTitle() != "" {
		tabTitle = displayTitle
	}

	page := ui.NewPage()
	page.SetTitle(tabTitle)

	// OG meta tags
	page.Head.AddChild(ui.NewElement("meta").
		SetAttribute("property", "og:title").
		SetAttribute("content", displayTitle))

	if meta != nil && meta.SocialDescription != "" {
		page.AddMeta("description", meta.SocialDescription)
		page.Head.AddChild(ui.NewElement("meta").
			SetAttribute("property", "og:description").
			SetAttribute("content", meta.SocialDescription))
	} else {
		page.AddMeta("description", displayTitle+" - "+pr.configManager.GetPageTitle())
	}

	page.Head.AddChild(ui.NewElement("meta").
		SetAttribute("property", "og:type").
		SetAttribute("content", "article"))

	if ogImage := pr.configManager.GetOGImage(); ogImage != "" {
		// Make absolute if site URL is set
		if siteURL := pr.configManager.GetSiteURL(); siteURL != "" && !strings.HasPrefix(ogImage, "http") {
			ogImage = strings.TrimRight(siteURL, "/") + "/" + strings.TrimLeft(ogImage, "/")
		}
		page.Head.AddChild(ui.NewElement("meta").
			SetAttribute("property", "og:image").
			SetAttribute("content", ogImage))
	}

	basePath := pr.configManager.GetBasePath()

	if favicon := pr.configManager.GetFavicon(); favicon != "" {
		faviconType := "image/x-icon"
		switch {
		case strings.HasSuffix(favicon, ".svg"):
			faviconType = "image/svg+xml"
		case strings.HasSuffix(favicon, ".png"):
			faviconType = "image/png"
		}
		page.AddLinkWithType("icon", basePath+favicon, faviconType)
	}

	pr.addExternalAssets(page, htmlContent)

	if err := pr.assetManager.AddStaticAssetsToPage(page); err != nil {
		return fmt.Errorf("adding static assets: %w", err)
	}

	// Persistent shell: body > #app > #content (swappable)
	body := kf.AppBody()

	// Content div — htmx swap target
	content := ui.NewElement("div").SetId("content").
		SetAttribute("data-title", tabTitle)

	// Page header with title
	title := ui.NewElement("h1").AddClass("page-header").SetContent(displayTitle)
	content.AddChild(title)

	// Tags
	if meta != nil && len(meta.Tags) > 0 {
		tagLine := ui.NewElement("div").AddClass("page-tags")
		for _, tag := range meta.Tags {
			tagLine.AddChild(
				ui.NewElement("code").AddClass("page-tag").SetContent(tag),
			)
		}
		content.AddChild(tagLine)
	}

	// Metadata line: date + reading time
	if meta != nil && (meta.Date != "" || meta.ReadingTime > 0) {
		metaLine := ui.NewElement("div").AddClass("page-meta")

		if meta.Date != "" {
			metaLine.AddChild(
				ui.NewElement("span").AddClass("page-date").SetContent(meta.Date),
			)
		}

		if meta.ReadingTime > 0 {
			readingText := fmt.Sprintf("%d min read", meta.ReadingTime)
			metaLine.AddChild(
				ui.NewElement("span").AddClass("page-reading-time").SetContent(readingText),
			)
		}

		content.AddChild(metaLine)
	}

	// Markdown content
	markdownDiv := ui.NewElement("div").AddClass("page-content")
	markdownDiv.Content = template.HTML(htmlContent)
	content.AddChild(markdownDiv)

	// Giscus comments
	if meta != nil && meta.Comments {
		if giscus := pr.renderGiscus(); giscus != nil {
			content.AddChild(giscus)
		}
	}

	body.AddChild(content)

	// Footer (outside #content so it persists across htmx swaps)
	if footer := pr.renderFooter(); footer != nil {
		body.AddChild(footer)
	}

	page.Body.AddChild(body)

	renderedHTML, err := page.Render()
	if err != nil {
		return fmt.Errorf("rendering page: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(renderedHTML), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", outputPath, err)
	}

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

func (pr *PageRenderer) renderFooter() *ui.Element {
	f := pr.configManager.GetFooter()
	if f.Text == "" && len(f.Links) == 0 {
		return nil
	}

	footer := ui.NewElement("footer").AddClass("site-footer")

	if len(f.Links) > 0 {
		nav := ui.NewElement("nav").AddClass("footer-links")
		for _, link := range f.Links {
			nav.AddChild(ui.NewElement("a").
				SetAttribute("href", link.URL).
				SetContent(link.Text))
		}
		footer.AddChild(nav)
	}

	if f.Text != "" {
		footer.AddChild(ui.NewElement("span").AddClass("footer-text").SetContent(f.Text))
	}

	return footer
}

func (pr *PageRenderer) renderGiscus() *ui.Element {
	g := pr.configManager.GetGiscus()
	if g.Repo == "" {
		return nil
	}

	if g.Mapping == "" {
		g.Mapping = "pathname"
	}
	if g.Theme == "" {
		g.Theme = "preferred_color_scheme"
	}
	if g.Lang == "" {
		g.Lang = "en"
	}

	wrapper := ui.NewElement("div").AddClass("giscus-comments")

	script := ui.NewElement("script").
		SetAttribute("src", "https://giscus.app/client.js").
		SetAttribute("data-repo", g.Repo).
		SetAttribute("data-repo-id", g.RepoID).
		SetAttribute("data-category", g.Category).
		SetAttribute("data-category-id", g.CategoryID).
		SetAttribute("data-mapping", g.Mapping).
		SetAttribute("data-strict", "0").
		SetAttribute("data-reactions-enabled", "1").
		SetAttribute("data-emit-metadata", "0").
		SetAttribute("data-input-position", "top").
		SetAttribute("data-theme", g.Theme).
		SetAttribute("data-lang", g.Lang).
		SetAttribute("data-loading", "lazy").
		SetAttribute("crossorigin", "anonymous").
		SetAttribute("async", "")

	wrapper.AddChild(script)
	return wrapper
}

func (pr *PageRenderer) addExternalAssets(page *ui.Page, htmlContent string) {
	page.AddStyleSheet("https://cdn.jsdelivr.net/gh/konstfish/ui@main/static/main.css")
	page.AddStyleSheet("https://cdn.jsdelivr.net/gh/konstfish/ui@main/static/prism.css")
	// pumice.css is auto-added by assetManager.AddStaticAssetsToPage
	if strings.Contains(htmlContent, `class="mermaid"`) {
		page.AddScript("https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js")
	}
	page.AddScript("https://unpkg.com/htmx.org@2.0.4")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js")
	page.AddStyleSheet("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/toolbar/prism-toolbar.min.css")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/toolbar/prism-toolbar.min.js")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/copy-to-clipboard/prism-copy-to-clipboard.min.js")
	// pumice.js is auto-added by assetManager.AddStaticAssetsToPage
}
