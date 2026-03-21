package sitebuilder

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/konstfish/pumice/internal/processor"
	"github.com/konstfish/pumice/internal/slug"
	"github.com/konstfish/pumice/internal/types"
)

type SiteBuilder struct {
	contentDir        string
	buildDir          string
	staticDir         string
	siteURL           string
	siteTitle         string
	fileCollector     FileCollectorInterface
	linkResolver      LinkResolverInterface
	assetManager      AssetManagerInterface
	markdownProcessor MarkdownProcessorInterface
	pageRenderer      PageRendererInterface
}

type FileCollectorInterface interface {
	BuildTitleMapping() error
	GetTitleMapping() map[string]string
	CollectReferencedFiles() error
	IsFileReferenced(path string) bool
	FindFileInContent(filename string) string
}

type LinkResolverInterface interface {
	SetTitleMapping(titleToFile map[string]string)
	SetCurrentContext(currentFile, outputDir string)
}

type AssetManagerInterface interface {
	CopyStaticAssets() error
	CopyContentFileIfReferenced(contentDir, path string) error
}

type MarkdownProcessorInterface interface {
	ProcessFile(path, contentDir, buildDir string, resolver interface{}) (string, string, *types.PageMetadata, error)
}

type PageRendererInterface interface {
	RenderPage(htmlContent, outputPath string, meta *types.PageMetadata) error
}

func NewSiteBuilder(contentDir, buildDir, staticDir, siteURL, siteTitle string,
	fileCollector FileCollectorInterface,
	linkResolver LinkResolverInterface,
	assetManager AssetManagerInterface,
	markdownProcessor MarkdownProcessorInterface,
	pageRenderer PageRendererInterface) *SiteBuilder {

	return &SiteBuilder{
		contentDir:        contentDir,
		buildDir:          buildDir,
		staticDir:         staticDir,
		siteURL:           strings.TrimRight(siteURL, "/"),
		siteTitle:         siteTitle,
		fileCollector:     fileCollector,
		linkResolver:      linkResolver,
		assetManager:      assetManager,
		markdownProcessor: markdownProcessor,
		pageRenderer:      pageRenderer,
	}
}

// pageEntry tracks a processed page for listings, feeds, and backlinks.
type pageEntry struct {
	Title       string
	Date        string
	Tags        []string
	HtmlPath    string // relative to buildDir, e.g. "blog/my-post.html"
	Dir         string // content-relative dir, e.g. "blog"
	HtmlContent string // rendered HTML content for backlink injection
	OutputPath  string // full output path
	Meta        *types.PageMetadata
}

func (sb *SiteBuilder) Build() error {
	if err := sb.prepareBuildDirectory(); err != nil {
		return fmt.Errorf("preparing build directory: %w", err)
	}

	if err := sb.fileCollector.BuildTitleMapping(); err != nil {
		return fmt.Errorf("building title mapping: %w", err)
	}

	titleMapping := sb.fileCollector.GetTitleMapping()
	sb.linkResolver.SetTitleMapping(titleMapping)

	if err := sb.assetManager.CopyStaticAssets(); err != nil {
		return fmt.Errorf("copying pumice static assets: %w", err)
	}

	if err := sb.copyUserStaticFiles(); err != nil {
		return fmt.Errorf("copying user static files: %w", err)
	}

	if err := sb.fileCollector.CollectReferencedFiles(); err != nil {
		return fmt.Errorf("collecting referenced files: %w", err)
	}

	// ── Pass 1: Process all markdown, collect page data ──────────
	dirsWithIndex := make(map[string]bool)
	var pages []pageEntry
	// Map from slug path to list of pages that link to it
	backlinks := make(map[string][]pageEntry)

	err := filepath.WalkDir(sb.contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return sb.assetManager.CopyContentFileIfReferenced(sb.contentDir, path)
		}

		relPath, _ := filepath.Rel(sb.contentDir, path)
		if filepath.Base(relPath) == "index.md" {
			dirsWithIndex[filepath.Dir(relPath)] = true
		}

		htmlContent, outputPath, meta, processErr := sb.markdownProcessor.ProcessFile(path, sb.contentDir, sb.buildDir, sb.linkResolver)
		if processErr != nil {
			return fmt.Errorf("processing markdown file %s: %w", path, processErr)
		}

		if meta != nil && meta.Draft {
			return nil
		}

		if filepath.Base(relPath) != "index.md" {
			baseName := strings.TrimSuffix(filepath.Base(relPath), ".md")
			slugName := slug.Slugify(baseName)
			htmlRel := filepath.Join(filepath.Dir(relPath), slugName+".html")

			entry := pageEntry{
				HtmlPath:    htmlRel,
				Dir:         filepath.Dir(relPath),
				HtmlContent: htmlContent,
				OutputPath:  outputPath,
				Meta:        meta,
			}
			if meta != nil {
				entry.Title = meta.Title
				entry.Date = meta.Date
				entry.Tags = meta.Tags
			}
			if entry.Title == "" {
				entry.Title = baseName
			}
			pages = append(pages, entry)
		} else {
			// Still render index pages immediately (no backlinks needed)
			if renderErr := sb.pageRenderer.RenderPage(htmlContent, outputPath, meta); renderErr != nil {
				return fmt.Errorf("rendering page %s: %w", outputPath, renderErr)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// ── Collect backlinks ────────────────────────────────────────
	for i := range pages {
		p := &pages[i]
		// Find all internal hrefs in this page's content
		targets := extractInternalLinks(p.HtmlContent)
		for _, target := range targets {
			// Normalize: strip .html, leading ./ or /
			target = strings.TrimSuffix(target, ".html")
			target = strings.TrimPrefix(target, "./")
			target = strings.TrimPrefix(target, "/")
			// If relative (e.g. "../self/foo"), resolve against page dir
			if strings.HasPrefix(target, "..") {
				target = filepath.Join(p.Dir, target)
				target = filepath.Clean(target)
			}
			backlinks[target] = append(backlinks[target], *p)
		}
	}

	// ── Build directory page map for listing embeds ──────────────
	dirPages := make(map[string][]pageEntry)
	for _, p := range pages {
		dirPages[p.Dir] = append(dirPages[p.Dir], p)
	}

	// ── Pass 2: Render pages with backlinks + embedded listings ──
	for _, p := range pages {
		content := p.HtmlContent

		// Replace {{listing:dir}} placeholders with actual listings
		content = sb.resolveListingPlaceholders(content, dirPages)

		// Find backlinks for this page
		pageSlug := strings.TrimSuffix(p.HtmlPath, ".html")
		if bl := backlinks[pageSlug]; len(bl) > 0 {
			content += sb.renderBacklinks(bl, pageSlug)
		}

		if err := sb.pageRenderer.RenderPage(content, p.OutputPath, p.Meta); err != nil {
			return fmt.Errorf("rendering page %s: %w", p.OutputPath, err)
		}
	}

	// Also resolve placeholders in already-rendered index pages
	sb.resolveIndexPlaceholders(dirPages)

	// ── Generate directory listings ──────────────────────────────
	if err := sb.generateDirectoryListings(dirsWithIndex, dirPages); err != nil {
		return err
	}

	// ── Generate tag pages ───────────────────────────────────────
	if err := sb.generateTagPages(pages); err != nil {
		return err
	}

	// ── Generate RSS feeds ───────────────────────────────────────
	if err := sb.generateRSSFeeds(pages); err != nil {
		return err
	}

	// ── Generate sitemap ─────────────────────────────────────────
	if err := sb.generateSitemap(pages); err != nil {
		return err
	}

	// ── Generate 404 page ────────────────────────────────────────
	if err := sb.generate404(); err != nil {
		return err
	}

	return nil
}

// ── Backlinks ────────────────────────────────────────────────────

func extractInternalLinks(html string) []string {
	var links []string
	// Match hx-get or href on data-internal links
	idx := 0
	for {
		marker := `data-internal`
		pos := strings.Index(html[idx:], marker)
		if pos == -1 {
			break
		}
		pos += idx

		// Find the href in this tag
		tagStart := strings.LastIndex(html[:pos], "<a ")
		if tagStart == -1 {
			idx = pos + len(marker)
			continue
		}
		tagEnd := strings.Index(html[tagStart:], ">")
		if tagEnd == -1 {
			idx = pos + len(marker)
			continue
		}
		tag := html[tagStart : tagStart+tagEnd]

		// Extract href value
		hrefIdx := strings.Index(tag, `href="`)
		if hrefIdx != -1 {
			hrefStart := hrefIdx + 6
			hrefEnd := strings.Index(tag[hrefStart:], `"`)
			if hrefEnd != -1 {
				links = append(links, tag[hrefStart:hrefStart+hrefEnd])
			}
		}

		idx = pos + len(marker)
	}
	return links
}

func (sb *SiteBuilder) renderBacklinks(bl []pageEntry, currentSlug string) string {
	// Deduplicate
	seen := make(map[string]bool)
	var unique []pageEntry
	for _, b := range bl {
		slug := strings.TrimSuffix(b.HtmlPath, ".html")
		if slug == currentSlug || seen[slug] {
			continue
		}
		seen[slug] = true
		unique = append(unique, b)
	}

	if len(unique) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(`<div class="backlinks"><h3>Backlinks</h3><ul>`)
	for _, b := range unique {
		absPath := "/" + strings.TrimSuffix(b.HtmlPath, ".html")
		absHtml := absPath + ".html"
		buf.WriteString(`<li><a href="`)
		buf.WriteString(absPath)
		buf.WriteString(`" data-internal hx-get="`)
		buf.WriteString(absHtml)
		buf.WriteString(`" hx-target="#content" hx-select="#content" hx-swap="outerHTML transition:true" hx-push-url="`)
		buf.WriteString(absPath)
		buf.WriteString(`">`)
		buf.WriteString(processor.EscapeHTML(b.Title))
		buf.WriteString(`</a></li>`)
	}
	buf.WriteString(`</ul></div>`)
	return buf.String()
}

// ── Listing embeds ───────────────────────────────────────────────

func (sb *SiteBuilder) resolveListingPlaceholders(html string, dirPages map[string][]pageEntry) string {
	for {
		idx := strings.Index(html, processor.ListingPlaceholder)
		if idx == -1 {
			break
		}

		endIdx := strings.Index(html[idx:], "-->")
		if endIdx == -1 {
			break
		}
		endIdx += idx + 3

		raw := html[idx+len(processor.ListingPlaceholder) : endIdx-3]

		// Parse "dir" or "dir:limit"
		dir := raw
		limit := 0
		if colonIdx := strings.LastIndex(raw, ":"); colonIdx != -1 {
			if n, err := fmt.Sscanf(raw[colonIdx+1:], "%d", &limit); err == nil && n == 1 {
				dir = raw[:colonIdx]
			}
		}

		entries := make([]pageEntry, len(dirPages[dir]))
		copy(entries, dirPages[dir])
		sortEntriesByDate(entries)

		if limit > 0 && limit < len(entries) {
			entries = entries[:limit]
		}

		listing := sb.renderListing(dir, entries)
		html = html[:idx] + listing + html[endIdx:]
	}
	return html
}

func (sb *SiteBuilder) resolveIndexPlaceholders(dirPages map[string][]pageEntry) {
	// Walk built index.html files and resolve any listing placeholders
	filepath.WalkDir(sb.buildDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(path) != "index.html" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if !strings.Contains(content, processor.ListingPlaceholder) {
			return nil
		}

		resolved := sb.resolveListingPlaceholders(content, dirPages)
		os.WriteFile(path, []byte(resolved), 0644)
		return nil
	})
}

// ── Directory listings ───────────────────────────────────────────

func (sb *SiteBuilder) generateDirectoryListings(dirsWithIndex map[string]bool, dirPages map[string][]pageEntry) error {
	dirs := make(map[string]bool)
	filepath.WalkDir(sb.contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "_attachments" {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(sb.contentDir, path)
			if rel != "." {
				dirs[rel] = true
			}
		}
		return nil
	})

	for dir := range dirs {
		if dirsWithIndex[dir] {
			continue
		}

		entries := dirPages[dir]
		if len(entries) == 0 {
			continue
		}

		sortEntriesByDate(entries)
		htmlContent := sb.renderListing(dir, entries)

		outputPath := filepath.Join(sb.buildDir, dir, "index.html")
		dirTitle := filepath.Base(dir)
		if len(dirTitle) > 0 {
			dirTitle = strings.ToUpper(dirTitle[:1]) + dirTitle[1:]
		}

		meta := &types.PageMetadata{Title: dirTitle}
		if err := sb.pageRenderer.RenderPage(htmlContent, outputPath, meta); err != nil {
			return fmt.Errorf("generating directory listing for %s: %w", dir, err)
		}
	}

	return nil
}

// ── Tag pages ────────────────────────────────────────────────────

func (sb *SiteBuilder) generateTagPages(pages []pageEntry) error {
	tagPages := make(map[string][]pageEntry)
	for _, p := range pages {
		for _, tag := range p.Tags {
			tagPages[tag] = append(tagPages[tag], p)
		}
	}

	// Create tags directory listing
	if len(tagPages) > 0 {
		if err := sb.generateTagIndex(tagPages); err != nil {
			return err
		}
	}

	for tag, entries := range tagPages {
		sortEntriesByDate(entries)
		htmlContent := sb.renderListing("tags/"+slug.Slugify(tag), entries)

		tagSlug := slug.Slugify(tag)
		outputPath := filepath.Join(sb.buildDir, "tags", tagSlug, "index.html")

		meta := &types.PageMetadata{Title: "#" + tag}
		if err := sb.pageRenderer.RenderPage(htmlContent, outputPath, meta); err != nil {
			return fmt.Errorf("generating tag page for %s: %w", tag, err)
		}
		fmt.Printf("Generated: %s\n", outputPath)
	}

	return nil
}

func (sb *SiteBuilder) generateTagIndex(tagPages map[string][]pageEntry) error {
	// Sort tags by post count descending
	type tagCount struct {
		Tag   string
		Count int
	}
	var tags []tagCount
	for tag, entries := range tagPages {
		tags = append(tags, tagCount{tag, len(entries)})
	}
	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count == tags[j].Count {
			return tags[i].Tag < tags[j].Tag
		}
		return tags[i].Count > tags[j].Count
	})

	var buf strings.Builder
	buf.WriteString(`<div class="tag-index">`)
	for _, tc := range tags {
		tagSlug := slug.Slugify(tc.Tag)
		absPath := "/tags/" + tagSlug
		buf.WriteString(`<a href="`)
		buf.WriteString(absPath)
		buf.WriteString(`" data-internal hx-get="`)
		buf.WriteString(absPath + ".html")
		buf.WriteString(`" hx-target="#content" hx-select="#content" hx-swap="outerHTML transition:true" hx-push-url="`)
		buf.WriteString(absPath)
		buf.WriteString(`"><code class="page-tag">`)
		buf.WriteString(processor.EscapeHTML(tc.Tag))
		buf.WriteString(`</code><span class="tag-count">`)
		buf.WriteString(fmt.Sprintf("%d", tc.Count))
		buf.WriteString(`</span></a> `)
	}
	buf.WriteString(`</div>`)

	outputPath := filepath.Join(sb.buildDir, "tags", "index.html")
	meta := &types.PageMetadata{Title: "Tags"}
	if err := sb.pageRenderer.RenderPage(buf.String(), outputPath, meta); err != nil {
		return fmt.Errorf("generating tag index: %w", err)
	}
	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

// ── RSS feeds ────────────────────────────────────────────────────

type rssChannel struct {
	XMLName     xml.Name  `xml:"channel"`
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate,omitempty"`
	GUID    string `xml:"guid"`
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

func (sb *SiteBuilder) generateRSSFeeds(pages []pageEntry) error {
	// Group by directory
	dirPages := make(map[string][]pageEntry)
	for _, p := range pages {
		dirPages[p.Dir] = append(dirPages[p.Dir], p)
	}

	// Also generate a root feed with all pages
	dirPages[""] = pages

	for dir, entries := range dirPages {
		sortEntriesByDate(entries)

		feedTitle := sb.siteTitle
		if dir != "" {
			dirName := filepath.Base(dir)
			feedTitle = strings.ToUpper(dirName[:1]) + dirName[1:] + " - " + sb.siteTitle
		}

		feedLink := sb.siteURL
		if dir != "" {
			feedLink = sb.siteURL + "/" + dir
		}

		var items []rssItem
		for _, e := range entries {
			pageURL := sb.siteURL + "/" + strings.TrimSuffix(e.HtmlPath, ".html")
			item := rssItem{
				Title: e.Title,
				Link:  pageURL,
				GUID:  pageURL,
			}
			if e.Date != "" {
				if t, err := time.Parse("2006-01-02", e.Date); err == nil {
					item.PubDate = t.Format(time.RFC1123Z)
				}
			}
			items = append(items, item)
		}

		feed := rssFeed{
			Version: "2.0",
			Channel: rssChannel{
				Title:       feedTitle,
				Link:        feedLink,
				Description: feedTitle,
				Items:       items,
			},
		}

		if len(items) > 0 {
			feed.Channel.PubDate = items[0].PubDate
		}

		feedPath := filepath.Join(sb.buildDir, dir, "feed.xml")
		if err := sb.writeXML(feedPath, feed); err != nil {
			return fmt.Errorf("generating RSS feed for %s: %w", dir, err)
		}
		fmt.Printf("Generated: %s\n", feedPath)
	}

	return nil
}

// ── Sitemap ──────────────────────────────────────────────────────

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapIndex struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (sb *SiteBuilder) generateSitemap(pages []pageEntry) error {
	var urls []sitemapURL

	// Add all pages
	for _, p := range pages {
		pageURL := sb.siteURL + "/" + strings.TrimSuffix(p.HtmlPath, ".html")
		u := sitemapURL{Loc: pageURL}
		if p.Date != "" {
			u.LastMod = p.Date
		}
		urls = append(urls, u)
	}

	// Add directory index pages
	filepath.WalkDir(sb.buildDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "index.html" {
			return nil
		}
		rel, _ := filepath.Rel(sb.buildDir, path)
		dir := filepath.Dir(rel)
		if dir == "." {
			urls = append(urls, sitemapURL{Loc: sb.siteURL + "/"})
		} else {
			urls = append(urls, sitemapURL{Loc: sb.siteURL + "/" + dir})
		}
		return nil
	})

	sitemap := sitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	sitemapPath := filepath.Join(sb.buildDir, "sitemap.xml")
	if err := sb.writeXML(sitemapPath, sitemap); err != nil {
		return fmt.Errorf("generating sitemap: %w", err)
	}
	fmt.Printf("Generated: %s\n", sitemapPath)
	return nil
}

// ── 404 page ─────────────────────────────────────────────────────

func (sb *SiteBuilder) generate404() error {
	htmlContent := `<div class="not-found">` +
		`<h2>Page not found</h2>` +
		`<p>The page you're looking for doesn't exist or has been moved.</p>` +
		`<p><a href="/" data-internal hx-get="/index.html" hx-target="#content" hx-select="#content" hx-swap="outerHTML transition:true" hx-push-url="/">← Back to home</a></p>` +
		`</div>`

	outputPath := filepath.Join(sb.buildDir, "404.html")
	meta := &types.PageMetadata{Title: "404"}
	if err := sb.pageRenderer.RenderPage(htmlContent, outputPath, meta); err != nil {
		return fmt.Errorf("generating 404 page: %w", err)
	}
	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

// ── Shared helpers ───────────────────────────────────────────────

func sortEntriesByDate(entries []pageEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Date == "" && entries[j].Date == "" {
			return entries[i].Title < entries[j].Title
		}
		if entries[i].Date == "" {
			return false
		}
		if entries[j].Date == "" {
			return true
		}
		return entries[i].Date > entries[j].Date
	})
}

func (sb *SiteBuilder) renderListing(dir string, entries []pageEntry) string {
	var buf strings.Builder

	for _, entry := range entries {
		entrySlug := strings.TrimSuffix(filepath.Base(entry.HtmlPath), ".html")
		absPath := "/" + strings.TrimSuffix(entry.HtmlPath, ".html")
		absHtml := absPath + ".html"
		_ = entrySlug

		buf.WriteString(`<div class="listing-entry">`)
		buf.WriteString(`<a href="`)
		buf.WriteString(absPath)
		buf.WriteString(`" class="listing-link" data-internal hx-get="`)
		buf.WriteString(absHtml)
		buf.WriteString(`" hx-target="#content" hx-select="#content" hx-swap="outerHTML transition:true" hx-push-url="`)
		buf.WriteString(absPath)
		buf.WriteString(`"><span class="listing-title">`)
		buf.WriteString(processor.EscapeHTML(entry.Title))
		buf.WriteString(`</span></a>`)

		if entry.Date != "" || len(entry.Tags) > 0 {
			buf.WriteString(`<div class="listing-meta">`)
			if entry.Date != "" {
				buf.WriteString(`<span class="listing-date">`)
				buf.WriteString(entry.Date)
				buf.WriteString(`</span>`)
			}
			for _, tag := range entry.Tags {
				buf.WriteString(`<code class="page-tag">`)
				buf.WriteString(processor.EscapeHTML(tag))
				buf.WriteString(`</code>`)
			}
			buf.WriteString(`</div>`)
		}

		buf.WriteString(`</div>`)
	}

	return buf.String()
}

func (sb *SiteBuilder) prepareBuildDirectory() error {
	if err := os.RemoveAll(sb.buildDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing build directory: %w", err)
	}
	return os.MkdirAll(sb.buildDir, 0755)
}

func (sb *SiteBuilder) writeXML(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(xml.Header)
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	return enc.Encode(v)
}

func (sb *SiteBuilder) copyUserStaticFiles() error {
	if sb.staticDir == "" {
		return nil
	}
	if _, err := os.Stat(sb.staticDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.WalkDir(sb.staticDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sb.staticDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for %s: %w", path, err)
		}

		outputPath := filepath.Join(sb.buildDir, relPath)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		src, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer src.Close()

		dst, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outputPath, err)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("copying %s: %w", path, err)
		}

		fmt.Printf("Copied static: %s\n", outputPath)
		return nil
	})
}
