package mdrenderer

import (
	"bytes"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer/knkatex"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

var (
	rctxKey = parser.NewContextKey()
)

// LocalRenderer is a local markdown renderer. It does not depend on any external services but it does not support mathjax rendering or any extensions to the markdown standard I intend to make.
type LocalRenderer struct {
	md goldmark.Markdown
}

func (r *LocalRenderer) Render(src []byte, ctx *kilonova.RenderContext) ([]byte, error) {
	var buf bytes.Buffer

	pctx := parser.NewContext()
	pctx.Set(rctxKey, ctx)

	doc := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(pctx))
	doc.OwnerDocument().Meta()["ctx"] = ctx
	err := r.md.Renderer().Render(&buf, src, doc)
	return buf.Bytes(), err
}

func NewLocalRenderer() *LocalRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, &attNode{}, knkatex.Extension,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions( // TODO: Keep in line with handlers.go:chromaCSS()
					chtml.TabWidth(4),
					chtml.WithClasses(true),
				),
			),
			&LinkConv{},
			&HeadingConv{},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	return &LocalRenderer{md}
}
