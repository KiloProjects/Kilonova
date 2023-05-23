package knkatex

import (
	"bytes"
	"context"
	"fmt"

	katex "github.com/FurqanSoftware/goldmark-katex"
	"github.com/Yiling-J/theine-go"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"go.uber.org/zap"
)

var Extension goldmark.Extender = katexExtension{}
var _ renderer.NodeRenderer = &mathjaxRenderer{}

type katexExtension struct{}
type mathjaxRenderer struct {
	inlineCache  *theine.LoadingCache[string, []byte]
	displayCache *theine.LoadingCache[string, []byte]
}

func (r *mathjaxRenderer) renderInlineMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		var buf bytes.Buffer
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			segment := c.(*ast.Text).Segment
			value := segment.Value(source)
			if bytes.HasSuffix(value, []byte("\n")) {
				buf.Write(value[:len(value)-1])
				if c != n.LastChild() {
					buf.Write([]byte(" "))
				}
			} else {
				buf.Write(value)
			}
		}

		val, err := r.inlineCache.Get(context.Background(), buf.String())
		if err != nil {
			w.WriteString(fmt.Sprintf("ERROR rendering latex: %v", err))
			return ast.WalkSkipChildren, nil
		}
		w.Write(val)

		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

func (r *mathjaxRenderer) writeLines(w *bytes.Buffer, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		w.Write(line.Value(source))
	}
}

func (r *mathjaxRenderer) renderBlockMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	node := n.(*mathjax.MathBlock)
	if entering {
		var buf bytes.Buffer
		r.writeLines(&buf, source, node)
		val, err := r.displayCache.Get(context.Background(), buf.String())
		if err != nil {
			w.WriteString(fmt.Sprintf("ERROR rendering latex: %v", err))
			return ast.WalkContinue, nil
		}
		w.Write(val)

		return ast.WalkContinue, nil
	}

	return ast.WalkContinue, nil
}

func (r *mathjaxRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mathjax.KindInlineMath, r.renderInlineMath)
	reg.Register(mathjax.KindMathBlock, r.renderBlockMath)
}

func (katexExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(mathjax.NewMathJaxBlockParser(), 701),
	), parser.WithInlineParsers(
		util.Prioritized(mathjax.NewInlineMathParser(), 501),
	))

	inlineCache, err := theine.NewBuilder[string, []byte](5000).BuildWithLoader(func(ctx context.Context, key string) (theine.Loaded[[]byte], error) {
		var buf bytes.Buffer
		katex.Render(&buf, []byte(key), false)
		return theine.Loaded[[]byte]{Value: buf.Bytes(), Cost: 1, TTL: 0}, nil
	})
	if err != nil {
		zap.S().Fatal(err)
	}

	displayCache, err := theine.NewBuilder[string, []byte](5000).BuildWithLoader(func(ctx context.Context, key string) (theine.Loaded[[]byte], error) {
		var buf bytes.Buffer
		katex.Render(&buf, []byte(key), true)
		return theine.Loaded[[]byte]{Value: buf.Bytes(), Cost: 1, TTL: 0}, nil
	})
	if err != nil {
		zap.S().Fatal(err)
	}

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&mathjaxRenderer{
			inlineCache:  inlineCache,
			displayCache: displayCache,
		}, 502),
	))
}
