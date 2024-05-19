package mdrenderer

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var _ parser.ASTTransformer = &HeadingTransformer{}

// HeadingTransformer is used for accesibility reasons.
// It transforms all <h_n>s in <h_(n+1)> headings
type HeadingTransformer struct{}

func (lt *HeadingTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		if h.Level != 6 {
			h.Level++
		}
		return ast.WalkContinue, nil
	})
}

type HeadingConv struct{}

func (conv *HeadingConv) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(util.Prioritized(&HeadingTransformer{}, 100)),
	)
}
