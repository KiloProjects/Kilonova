package mdrenderer

import (
	"bytes"

	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer/knkatex"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// LocalRenderer is a local markdown renderer. It does not depend on any external services but it does not support mathjax rendering or any extensions to the markdown standard I intend to make.
type LocalRenderer struct {
	md goldmark.Markdown
}

func (r *LocalRenderer) Render(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := r.md.Convert(src, &buf)
	return buf.Bytes(), err
}

func NewLocalRenderer() *LocalRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, &attNode{}, knkatex.Extension,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions( // TODO: Keep in line with handlers.go:chromaCSS()
					chtml.TabWidth(4),
					chtml.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	return &LocalRenderer{md}
}
