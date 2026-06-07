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
	GetBuildDir() string
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

	siteURL := pr.configManager.GetSiteURL()
	siteName := pr.configManager.GetPageTitle()

	// Description: explicit socialDescription, else a sensible fallback.
	description := displayTitle + " - " + siteName
	if meta != nil && meta.SocialDescription != "" {
		description = meta.SocialDescription
	}
	page.AddMeta("description", description)

	// Canonical / og:url — the page's absolute address.
	canonical := pr.canonicalURL(outputPath, siteURL)
	if canonical != "" {
		page.AddLink("canonical", canonical)
	}

	// Absolute OG image URL.
	ogImage := pr.configManager.GetOGImage()
	if ogImage != "" && siteURL != "" && !strings.HasPrefix(ogImage, "http") {
		ogImage = strings.TrimRight(siteURL, "/") + "/" + strings.TrimLeft(ogImage, "/")
	}

	// Dated content is an "article"; everything else (home, listings, tags) a "website".
	ogType := "website"
	if meta != nil && meta.Date != "" {
		ogType = "article"
	}

	// Open Graph tags.
	addProperty := func(property, content string) {
		if content == "" {
			return
		}
		page.Head.AddChild(ui.NewElement("meta").
			SetAttribute("property", property).
			SetAttribute("content", content))
	}
	addProperty("og:title", displayTitle)
	addProperty("og:description", description)
	addProperty("og:type", ogType)
	addProperty("og:site_name", siteName)
	addProperty("og:url", canonical)
	addProperty("og:image", ogImage)
	if ogType == "article" {
		if t, ok := types.ParseDate(meta.Date); ok {
			addProperty("article:published_time", t.Format("2006-01-02"))
		}
	}

	// Twitter Card tags.
	twitterCard := "summary"
	if ogImage != "" {
		twitterCard = "summary_large_image"
	}
	page.AddMeta("twitter:card", twitterCard)
	page.AddMeta("twitter:title", displayTitle)
	page.AddMeta("twitter:description", description)
	if ogImage != "" {
		page.AddMeta("twitter:image", ogImage)
		// og:image:alt aids both Twitter/X cards and accessibility.
		addProperty("og:image:alt", description)
	}
	if canonical != "" {
		page.AddMeta("twitter:url", canonical)
	}
	if domain := urlHost(siteURL); domain != "" {
		page.AddMeta("twitter:domain", domain)
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

// urlHost extracts the bare host (no scheme, no path) from a site URL, e.g.
// "https://konst.fish/blog" -> "konst.fish". Used for twitter:domain.
func urlHost(siteURL string) string {
	host := siteURL
	if idx := strings.Index(host, "://"); idx != -1 {
		host = host[idx+3:]
	}
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// canonicalURL derives a page's absolute URL from its output path. Directory
// index pages map to the directory URL (with trailing slash); other pages drop
// the .html extension. Returns "" when no site URL is configured.
func (pr *PageRenderer) canonicalURL(outputPath, siteURL string) string {
	if siteURL == "" {
		return ""
	}

	rel, err := filepath.Rel(pr.configManager.GetBuildDir(), outputPath)
	if err != nil {
		return ""
	}

	urlPath := strings.TrimSuffix(filepath.ToSlash(rel), ".html")
	base := strings.TrimRight(siteURL, "/")

	switch {
	case urlPath == "index":
		return base + "/"
	case strings.HasSuffix(urlPath, "/index"):
		return base + "/" + strings.TrimSuffix(urlPath, "index")
	default:
		return base + "/" + urlPath
	}
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
	page.AddStyleSheet("https://cdn.jsdelivr.net/gh/konstfish/ui@0.0.2/static/main.css")
	page.AddStyleSheet("https://cdn.jsdelivr.net/gh/konstfish/ui@0.0.2/static/prism.css")
	// pumice.css is auto-added by assetManager.AddStaticAssetsToPage
	if strings.Contains(htmlContent, `class="mermaid"`) {
		page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/mermaid/10.9.1/mermaid.min.js")
	}
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/htmx/2.0.4/htmx.min.js")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/prism.min.js")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js")
	page.AddStyleSheet("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/toolbar/prism-toolbar.min.css")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/toolbar/prism-toolbar.min.js")
	page.AddScript("https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/copy-to-clipboard/prism-copy-to-clipboard.min.js")
	// pumice.js is auto-added by assetManager.AddStaticAssetsToPage
}
