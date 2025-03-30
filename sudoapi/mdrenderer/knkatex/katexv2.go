package knkatex

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/Yiling-J/theine-go"
	"github.com/gohugoio/hugo-goldmark-extensions/passthrough"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type passthroughRenderCtx string

var (
	passthroughRenderCtxNode passthroughRenderCtx = "node"

	inlineDelimiters = []passthrough.Delimiters{
		{Open: "$", Close: "$"},
		{Open: "\\(", Close: "\\)"},
	}

	blockDelimiters = []passthrough.Delimiters{
		{Open: "$$", Close: "$$"},
		{Open: "\\[", Close: "\\]"},
	}
)

var ExtensionV2 goldmark.Extender = katexV2Extension{}
var _ renderer.NodeRenderer = &katexV2Renderer{}

type katexV2Extension struct{}

type katexV2Renderer struct {
	inlineCache  *theine.LoadingCache[string, []byte]
	displayCache *theine.LoadingCache[string, []byte]
}

func (r *katexV2Renderer) renderInlineMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, ok := n.(*passthrough.PassthroughInline)
		if !ok {
			slog.ErrorContext(context.Background(), "Invalid node type for inline math")
			return ast.WalkSkipChildren, nil
		}

		rawVal := string(n.Text(source))
		rawVal = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rawVal, n.Delimiters.Open), n.Delimiters.Close))

		val, err := r.inlineCache.Get(context.WithValue(context.Background(), mjRenderCtxNode, n), rawVal)
		if err != nil {
			w.WriteString(fmt.Sprintf("ERROR rendering latex: %v", err))
			return ast.WalkSkipChildren, nil
		}
		w.Write(val)
	}
	return ast.WalkContinue, nil
}

func (r *katexV2Renderer) renderBlockMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n, ok := n.(*passthrough.PassthroughBlock)
		if !ok {
			slog.ErrorContext(context.Background(), "Invalid node type for block math")
			return ast.WalkSkipChildren, nil
		}

		var buf bytes.Buffer
		l := n.Lines().Len()
		for i := range l {
			line := n.Lines().At(i)
			buf.WriteString(string(line.Value(source)))
		}

		rawVal := string(buf.String())
		rawVal = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rawVal, n.Delimiters.Open), n.Delimiters.Close))
		val, err := r.displayCache.Get(context.WithValue(context.Background(), mjRenderCtxNode, n), rawVal)
		if err != nil {
			w.WriteString(fmt.Sprintf("ERROR rendering latex: %v", err))
			return ast.WalkSkipChildren, nil
		}
		w.Write(val)
	}
	return ast.WalkSkipChildren, nil
}

func (r *katexV2Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(passthrough.KindPassthroughInline, r.renderInlineMath)
	reg.Register(passthrough.KindPassthroughBlock, r.renderBlockMath)
}

func (katexV2Extension) Extend(m goldmark.Markdown) {
	passthrough.New(passthrough.Config{
		InlineDelimiters: inlineDelimiters,
		BlockDelimiters:  blockDelimiters,
	}).Extend(m)

	inlineCache, err := theine.NewBuilder[string, []byte](5000).BuildWithLoader(cacheLoader(false))
	if err != nil {
		slog.ErrorContext(context.Background(), "Couldn't initialize KaTeX inline cache")
		os.Exit(1)
	}

	displayCache, err := theine.NewBuilder[string, []byte](5000).BuildWithLoader(cacheLoader(true))
	if err != nil {
		slog.ErrorContext(context.Background(), "Couldn't initialize KaTeX display cache")
		os.Exit(1)
	}

	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&katexV2Renderer{
			inlineCache:  inlineCache,
			displayCache: displayCache,
		}, 98),
	))
}

func cacheLoader(display bool) func(ctx context.Context, key string) (theine.Loaded[[]byte], error) {
	return func(ctx context.Context, key string) (theine.Loaded[[]byte], error) {
		var buf bytes.Buffer
		if err := Render(ctx, &buf, []byte(key), display); err != nil {
			// TODO: Better checking for ctx and stuff
			val, ok := ctx.Value(passthroughRenderCtxNode).(ast.Node)
			if ok && val.OwnerDocument().Meta()["ctx"] != nil {
				x, ok := val.OwnerDocument().Meta()["ctx"].(*kilonova.MarkdownRenderContext)
				if x != nil && ok && x.Problem != nil {
					slog.DebugContext(ctx, "Markdown error", slog.Any("err", err), slog.Any("problem", x.Problem))
				}
			} else {
				slog.DebugContext(ctx, "Couldn't render markdown", slog.Any("err", err))
			}
		}
		return theine.Loaded[[]byte]{Value: buf.Bytes(), Cost: 1, TTL: 0}, nil
	}
}
