package web

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Renderer will replace the current frontend markdown renderer, for easier extension development and faster load times
type Renderer struct {
	md goldmark.Markdown
}

func (r *Renderer) Render(src []byte) (bytes.Buffer, error) {
	var buf bytes.Buffer
	err := r.md.Convert(src, &buf)
	return buf, err
}

func NewRenderer() (*Renderer, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	return &Renderer{md: md}, nil
}
