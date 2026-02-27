// Package mdpng converts Markdown text into a PNG image.
//
// It uses goldmark to parse Markdown into an AST, then walks the AST
// to render styled text and shapes onto a canvas via fogleman/gg.
//
// Usage:
//
//	var buf bytes.Buffer
//	err := mdpng.Convert(markdownBytes, &buf)
//	// buf now contains PNG data
//
//	// With options:
//	err := mdpng.Convert(markdownBytes, &buf,
//	    mdpng.WithWidth(1024),
//	    mdpng.WithDarkMode(true),
//	)
package mdpng

import (
	"io"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Option configures the markdown-to-PNG conversion.
type Option func(*config)

type config struct {
	Width    int
	Height   int // 0 means auto-size to content
	Padding  int
	FontSize float64
	DarkMode bool
}

func defaultConfig() *config {
	return &config{
		Width:    800,
		Padding:  0,
		FontSize: 14,
		DarkMode: false,
	}
}

// WithWidth sets the image width in pixels. Default: 800.
func WithWidth(w int) Option {
	return func(c *config) { c.Width = w }
}

// WithHeight sets the image height in pixels. Default: 0 (auto-size to content).
func WithHeight(h int) Option {
	return func(c *config) { c.Height = h }
}

// WithPadding sets the padding around the content in pixels. Default: 20.
func WithPadding(p int) Option {
	return func(c *config) { c.Padding = p }
}

// WithFontSize sets the base font size in points. Default: 14.
func WithFontSize(pt float64) Option {
	return func(c *config) { c.FontSize = pt }
}

// WithDarkMode enables dark mode (light text on dark background). Default: false.
func WithDarkMode(on bool) Option {
	return func(c *config) { c.DarkMode = on }
}

// Convert converts Markdown bytes into a PNG image written to w.
func Convert(markdown []byte, w io.Writer, opts ...Option) error {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Parse markdown into AST using goldmark.
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(markdown)
	doc := md.Parser().Parse(reader)

	// Render AST to PNG.
	r := newRenderer(cfg, markdown)
	return r.render(doc, w)
}
