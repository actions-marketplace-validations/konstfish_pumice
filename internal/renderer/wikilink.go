package renderer

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/yuin/goldmark/ast"
	goldmarkRenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

// WikilinkRenderer extends the default wikilink renderer with Obsidian-style
// image scaling support: ![[image.jpg|200]] renders as <img width="200">.
type WikilinkRenderer struct {
	Resolver wikilink.Resolver

	once    sync.Once
	hasDest sync.Map
}

func (r *WikilinkRenderer) init() {
	r.once.Do(func() {
		if r.Resolver == nil {
			r.Resolver = wikilink.DefaultResolver
		}
	})
}

// RegisterFuncs registers the wikilink rendering function with goldmark.
func (r *WikilinkRenderer) RegisterFuncs(reg goldmarkRenderer.NodeRendererFuncRegisterer) {
	reg.Register(wikilink.Kind, r.Render)
}

// Render renders a wikilink node as HTML.
func (r *WikilinkRenderer) Render(bw util.BufWriter, src []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	r.init()

	n, ok := node.(*wikilink.Node)
	if !ok {
		return ast.WalkStop, fmt.Errorf("unexpected node %T, expected *wikilink.Node", node)
	}

	if entering {
		return r.enter(bw, n, src)
	}

	r.exit(bw, n)
	return ast.WalkContinue, nil
}

func (r *WikilinkRenderer) enter(bw util.BufWriter, n *wikilink.Node, src []byte) (ast.WalkStatus, error) {
	dest, err := r.Resolver.ResolveWikilink(n)
	if err != nil {
		return ast.WalkStop, fmt.Errorf("resolve %q: %w", n.Target, err)
	}
	if len(dest) == 0 {
		return ast.WalkContinue, nil
	}

	// Regular wikilink (not an image embed)
	if !isImage(n) {
		r.hasDest.Store(n, struct{}{})
		_, _ = bw.WriteString(`<a href="`)
		_, _ = bw.Write(util.URLEscape(dest, true))
		_, _ = bw.WriteString(`">`)
		return ast.WalkContinue, nil
	}

	// Image embed
	_, _ = bw.WriteString(`<img src="`)
	_, _ = bw.Write(util.URLEscape(dest, true))
	_, _ = bw.WriteString(`"`)

	if n.ChildCount() == 1 {
		label := nodeText(src, n.FirstChild())
		if !bytes.Equal(label, n.Target) {
			labelStr := string(label)

			if imgW, imgH, ok := parseDimensions(labelStr); ok {
				// Obsidian dimension syntax: |200 or |200x150
				fmt.Fprintf(bw, ` width="%d"`, imgW)
				if imgH > 0 {
					fmt.Fprintf(bw, ` height="%d"`, imgH)
				}
			} else {
				// Regular alt text
				_, _ = bw.WriteString(` alt="`)
				_, _ = bw.Write(util.EscapeHTML(label))
				_, _ = bw.WriteString(`"`)
			}
		}
	}

	_, _ = bw.WriteString(`>`)
	return ast.WalkSkipChildren, nil
}

func (r *WikilinkRenderer) exit(bw util.BufWriter, n *wikilink.Node) {
	if _, ok := r.hasDest.LoadAndDelete(n); ok {
		_, _ = bw.WriteString("</a>")
	}
}

func isImage(n *wikilink.Node) bool {
	if !n.Embed {
		return false
	}
	switch filepath.Ext(string(n.Target)) {
	case ".apng", ".avif", ".gif", ".jpg", ".jpeg", ".jfif", ".pjpeg", ".pjp", ".png", ".svg", ".webp":
		return true
	default:
		return false
	}
}

// parseDimensions parses Obsidian-style dimension strings.
// "200" returns (200, 0, true) — width only.
// "200x150" returns (200, 150, true) — width and height.
func parseDimensions(s string) (width, height int, ok bool) {
	if w, err := strconv.Atoi(s); err == nil && w > 0 {
		return w, 0, true
	}

	for i, c := range s {
		if c == 'x' || c == 'X' {
			w, errW := strconv.Atoi(s[:i])
			h, errH := strconv.Atoi(s[i+1:])
			if errW == nil && errH == nil && w > 0 && h > 0 {
				return w, h, true
			}
		}
	}

	return 0, 0, false
}

func nodeText(src []byte, n ast.Node) []byte {
	var buf bytes.Buffer
	writeNodeText(src, &buf, n)
	return buf.Bytes()
}

func writeNodeText(src []byte, dst io.Writer, n ast.Node) {
	switch n := n.(type) {
	case *ast.Text:
		_, _ = dst.Write(n.Segment.Value(src))
	case *ast.String:
		_, _ = dst.Write(n.Value)
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			writeNodeText(src, dst, c)
		}
	}
}
