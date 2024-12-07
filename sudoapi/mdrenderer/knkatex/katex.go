// Code in katex.go has been mostly derived from [goldmark-mathjax](https://github.com/litao91/goldmark-mathjax).
package knkatex

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/Yiling-J/theine-go"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type mjRenderCtx string

var (
	mjRenderCtxNode mjRenderCtx = "node"
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

		val, err := r.inlineCache.Get(context.WithValue(context.Background(), mjRenderCtxNode, n), buf.String())
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
		val, err := r.displayCache.Get(context.WithValue(context.Background(), mjRenderCtxNode, n), buf.String())
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
		if err := Render(ctx, &buf, []byte(key), false); err != nil {
			// TODO: Better checking for ctx and stuff
			val, ok := ctx.Value(mjRenderCtxNode).(ast.Node)
			if ok && val.OwnerDocument().Meta()["ctx"] != nil {
				x, ok := val.OwnerDocument().Meta()["ctx"].(*kilonova.RenderContext)
				if x != nil && ok && x.Problem != nil {
					slog.DebugContext(ctx, "Markdown error", slog.Any("err", err), slog.Any("problem", x.Problem))
				}
			} else {
				slog.DebugContext(ctx, "Couldn't render markdown", slog.Any("err", err))
			}
		}
		return theine.Loaded[[]byte]{Value: buf.Bytes(), Cost: 1, TTL: 0}, nil
	})
	if err != nil {
		slog.ErrorContext(context.Background(), "Couldn't initialize KaTeX inline cache")
		os.Exit(1)
	}

	displayCache, err := theine.NewBuilder[string, []byte](5000).BuildWithLoader(func(ctx context.Context, key string) (theine.Loaded[[]byte], error) {
		var buf bytes.Buffer
		if err := Render(ctx, &buf, []byte(key), true); err != nil {
			val, ok := ctx.Value(mjRenderCtxNode).(ast.Node)
			if ok && val.OwnerDocument().Meta()["ctx"] != nil {
				x := val.OwnerDocument().Meta()["ctx"].(*kilonova.RenderContext)
				if x.Problem != nil {
					slog.DebugContext(ctx, "Markdown error", slog.Any("err", err), slog.Any("problem", x.Problem))
				}
			} else {
				slog.DebugContext(ctx, "Couldn't render markdown", slog.Any("err", err))
			}
		}
		return theine.Loaded[[]byte]{Value: buf.Bytes(), Cost: 1, TTL: 0}, nil
	})
	if err != nil {
		slog.ErrorContext(context.Background(), "Couldn't initialize KaTeX display cache")
		os.Exit(1)
	}

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&mathjaxRenderer{
			inlineCache:  inlineCache,
			displayCache: displayCache,
		}, 502),
	))
}
