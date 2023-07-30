package mdrenderer

import (
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Links are also converted automatically if they can link to an attachment

var _ parser.ASTTransformer = &LinkTransformer{}

type LinkTransformer struct{}

func (lt *LinkTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		if len(string(h.Destination)) == 0 || string(h.Destination)[0] == '#' {
			return ast.WalkContinue, nil
		}

		if path.Base(path.Clean(string(h.Destination))) == string(h.Destination) {
			ctx, ok := pc.Get(rctxKey).(*kilonova.RenderContext)
			h.Destination = []byte(attachmentURL(ctx, ok, string(h.Destination)))
			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})
}

type LinkConv struct{}

func (conv *LinkConv) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(
		parser.WithASTTransformers(util.Prioritized(&LinkTransformer{}, 100)),
	)
}
