package mdrenderer

import (
	"bytes"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// LocalRenderer is a local markdown renderer. It does not depend on any external services but it does not support mathjax rendering or any extensions to the markdown standard I intend to make.
type LocalRenderer struct {
	md  goldmark.Markdown
	pol *bluemonday.Policy
}

func (r *LocalRenderer) Render(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := r.md.Convert(src, &buf)
	return r.pol.SanitizeReader(&buf).Bytes(), err
}

func NewLocalRenderer() *LocalRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, mathjax.MathJax, &attNode{}),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	pol := bluemonday.UGCPolicy()
	return &LocalRenderer{md, pol}
}
