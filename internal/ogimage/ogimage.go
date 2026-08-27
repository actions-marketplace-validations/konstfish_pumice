// Package ogimage renders per-page Open Graph preview cards as PNGs: a clean
// card with the site brand top-left, the page title centered, and tags / date /
// reading time along the bottom. Drawn entirely in Go (no headless browser)
// using the Liberation Sans typeface (a Go module), so no font files live in
// the repo.
package ogimage

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/go-fonts/liberation/liberationsansbold"
	"github.com/go-fonts/liberation/liberationsansregular"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// Card dimensions — the de-facto Open Graph image size.
const (
	Width  = 1200
	Height = 630
)

// Nord palette (https://www.nord-theme.com/).
const (
	colBgTop = "#323846" // subtle top-of-gradient
	colBgBot = "#2B303B" // subtle bottom-of-gradient
	colTitle = "#ECEFF4" // nord6  — page title
	colTag   = "#81A1C1" // nord9  — tags
	colMeta  = "#7B889E" // dimmed — brand, date / reading time
)

// Card holds the page data rendered onto an OG image.
type Card struct {
	Site        string // brand label shown top-left, e.g. "konst.fish"
	Title       string
	Tags        []string
	Date        string
	ReadingTime int // minutes; 0 omits the reading-time label
}

// Generator renders Cards. It parses the fonts once and is safe to reuse
// across many pages.
type Generator struct {
	regular *truetype.Font
	bold    *truetype.Font
}

// New parses the Liberation Sans typeface and returns a ready Generator.
func New() (*Generator, error) {
	reg, err := truetype.Parse(liberationsansregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing regular font: %w", err)
	}
	bold, err := truetype.Parse(liberationsansbold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing bold font: %w", err)
	}
	return &Generator{regular: reg, bold: bold}, nil
}

// Save renders the card and writes it as a PNG to path, creating parent
// directories as needed.
func (g *Generator) Save(c Card, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating og image directory: %w", err)
	}
	dc := g.draw(c)
	if err := dc.SavePNG(path); err != nil {
		return fmt.Errorf("encoding og image %s: %w", path, err)
	}
	return nil
}

// Render returns the card as an image, primarily for testing.
func (g *Generator) Render(c Card) image.Image {
	return g.draw(c).Image()
}

func (g *Generator) face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func (g *Generator) draw(c Card) *gg.Context {
	dc := gg.NewContext(Width, Height)
	measureW := func(s string) float64 { w, _ := dc.MeasureString(s); return w }

	// Background — a subtle vertical gradient for a little depth.
	grad := gg.NewLinearGradient(0, 0, 0, Height)
	grad.AddColorStop(0, mustColor(colBgTop))
	grad.AddColorStop(1, mustColor(colBgBot))
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, Width, Height)
	dc.Fill()

	const pad = 80.0
	contentW := float64(Width) - 2*pad

	// Brand, top-left — lowercase, muted.
	if c.Site != "" {
		dc.SetFontFace(g.face(g.regular, 32))
		dc.SetHexColor(colMeta)
		dc.DrawStringAnchored(strings.ToLower(c.Site), pad, pad+8, 0, 0.5)
	}

	// Title — large, vertically centered in the card.
	dc.SetFontFace(g.face(g.bold, 76))
	titleLines := wrapLines(measureW, c.Title, contentW, 3)
	const titleLH = 92.0
	blockTop := float64(Height)/2 - titleLH*float64(len(titleLines))/2
	dc.SetHexColor(colTitle)
	for i, line := range titleLines {
		cy := blockTop + titleLH*(float64(i)+0.5)
		dc.DrawStringAnchored(line, pad, cy, 0, 0.5)
	}

	// Bottom row: tags (left) and "date · N min read" (right).
	rowY := float64(Height) - pad
	dc.SetFontFace(g.face(g.regular, 28))

	meta := metaLabel(c)
	metaW := 0.0
	if meta != "" {
		metaW = measureW(meta)
		dc.SetHexColor(colMeta)
		dc.DrawStringAnchored(meta, float64(Width)-pad, rowY, 1, 0.5)
	}

	if len(c.Tags) > 0 {
		tags := make([]string, len(c.Tags))
		for i, t := range c.Tags {
			tags[i] = "#" + t
		}
		maxTagW := contentW - metaW - 40
		label := truncateToWidth(measureW, strings.Join(tags, "  "), maxTagW)
		dc.SetHexColor(colTag)
		dc.DrawStringAnchored(label, pad, rowY, 0, 0.5)
	}

	return dc
}

// metaLabel builds the right-aligned "date · N min read" string, omitting
// whichever parts are absent.
func metaLabel(c Card) string {
	var parts []string
	if c.Date != "" {
		parts = append(parts, c.Date)
	}
	if c.ReadingTime > 0 {
		parts = append(parts, fmt.Sprintf("%d min read", c.ReadingTime))
	}
	return strings.Join(parts, " · ")
}

// mustColor parses a "#rrggbb" string into an opaque color for gradient stops.
// Inputs are compile-time constants, so parsing never fails in practice.
func mustColor(hex string) color.Color {
	var r, g, b uint8
	fmt.Sscanf(strings.TrimPrefix(hex, "#"), "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
