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

type Renderer struct {
	md goldmark.Markdown
}

func (r *Renderer) Render(src []byte, ctx *kilonova.MarkdownRenderContext) ([]byte, error) {
	var buf bytes.Buffer

	pctx := parser.NewContext()
	pctx.Set(rctxKey, ctx)

	doc := r.md.Parser().Parse(text.NewReader(src), parser.WithContext(pctx))
	doc.OwnerDocument().Meta()["ctx"] = ctx
	err := r.md.Renderer().Render(&buf, src, doc)
	return buf.Bytes(), err
}

func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, extension.Footnote, &attNode{},
			// knkatex.Extension,
			knkatex.ExtensionV2,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					HighlightFormatOptions()...,
				),
			),
			&LinkConv{},
			&HeadingConv{},
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	return &Renderer{md}
}

func HighlightFormatOptions() []chtml.Option {
	return []chtml.Option{
		chtml.TabWidth(4),
		chtml.WithClasses(true),
	}
}
