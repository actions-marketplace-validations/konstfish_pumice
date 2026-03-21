package renderer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	goldmarkRenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/wikilink"
)

type testResolver struct{}

func (r *testResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	return n.Target, nil
}

func newTestMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithParserOptions(
			parser.WithInlineParsers(
				util.Prioritized(&wikilink.Parser{}, 199),
			),
		),
		goldmark.WithRendererOptions(
			goldmarkRenderer.WithNodeRenderers(
				util.Prioritized(&WikilinkRenderer{
					Resolver: &testResolver{},
				}, 199),
			),
		),
	)
}

func TestImageScalingWidth(t *testing.T) {
	md := newTestMarkdown()
	var buf bytes.Buffer

	err := md.Convert([]byte("![[photo.jpg|200]]"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, `width="200"`) {
		t.Errorf("expected width attribute, got: %s", result)
	}
	if strings.Contains(result, `alt=`) {
		t.Errorf("should not have alt when label is a dimension, got: %s", result)
	}
}

func TestImageScalingWidthHeight(t *testing.T) {
	md := newTestMarkdown()
	var buf bytes.Buffer

	err := md.Convert([]byte("![[photo.png|300x200]]"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, `width="300"`) {
		t.Errorf("expected width=300, got: %s", result)
	}
	if !strings.Contains(result, `height="200"`) {
		t.Errorf("expected height=200, got: %s", result)
	}
}

func TestImageAltText(t *testing.T) {
	md := newTestMarkdown()
	var buf bytes.Buffer

	err := md.Convert([]byte("![[photo.jpg|a nice photo]]"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, `alt="a nice photo"`) {
		t.Errorf("expected alt text, got: %s", result)
	}
}

func TestImageNoLabel(t *testing.T) {
	md := newTestMarkdown()
	var buf bytes.Buffer

	err := md.Convert([]byte("![[photo.jpg]]"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, `<img src="photo.jpg">`) {
		t.Errorf("expected plain img tag, got: %s", result)
	}
}

func TestRegularWikilink(t *testing.T) {
	md := newTestMarkdown()
	var buf bytes.Buffer

	err := md.Convert([]byte("[[Some Page]]"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	result := buf.String()
	if !strings.Contains(result, `<a href="Some%20Page">`) {
		t.Errorf("expected link, got: %s", result)
	}
}
