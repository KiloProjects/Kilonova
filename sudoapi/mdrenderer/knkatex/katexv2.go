package knkatex

import (
	"bytes"
	"context"
	"fmt"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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

	customBlockInfoKey = parser.NewContextKey()
)

var ExtensionV2 goldmark.Extender = katexV2Extension{}
var _ renderer.NodeRenderer = &katexV2Renderer{}
var _ parser.BlockParser = &katexBlockParser{}

type blockData struct {
	indent int
	length int
	node   ast.Node
}

type katexBlockParser struct {
}

func (b *katexBlockParser) Trigger() []byte {
	return []byte{'$'}
}

func (b *katexBlockParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || line[pos] != '$' {
		return nil, parser.NoChildren
	}
	findent := pos
	i := pos
	for ; i < len(line) && line[i] == '$'; i++ {
	}
	oFenceLength := i - pos
	if oFenceLength < 2 {
		return nil, parser.NoChildren
	}
	node := &passthrough.PassthroughBlock{Delimiters: &passthrough.Delimiters{Open: "$$", Close: "$$"}, BaseBlock: ast.BaseBlock{}}
	pc.Set(customBlockInfoKey, &blockData{findent, oFenceLength, node})
	return node, parser.NoChildren
}

func (b *katexBlockParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, segment := reader.PeekLine()
	fdata := pc.Get(customBlockInfoKey).(*blockData)

	w, pos := util.IndentWidth(line, fdata.indent)
	if w < 3 {
		i := pos
		for ; i < len(line) && line[i] == '$'; i++ {
		}
		length := i - pos
		if length >= fdata.length && util.IsBlank(line[i:]) {
			newline := 1
			if line[len(line)-1] != '\n' {
				newline = 0
			}
			reader.Advance(segment.Stop - segment.Start - newline + segment.Padding)
			return parser.Close
		}
	}
	pos, padding := util.IndentPositionPadding(line, reader.LineOffset(), segment.Padding, fdata.indent)
	if pos < 0 {
		pos = util.FirstNonSpacePosition(line)
		if pos < 0 {
			pos = 0
		}
		padding = 0
	}
	seg := text.NewSegmentPadding(segment.Start+pos, segment.Stop, padding)
	// if code block line starts with a tab, keep a tab as it is.
	if padding != 0 {
		preserveLeadingTabInCodeBlock(&seg, reader, fdata.indent)
	}
	seg.ForceNewline = true // EOF as newline
	node.Lines().Append(seg)
	reader.AdvanceAndSetPadding(segment.Stop-segment.Start-pos-1, padding)
	return parser.Continue | parser.NoChildren
}

func (b *katexBlockParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	fdata := pc.Get(customBlockInfoKey).(*blockData)
	if fdata.node == node {
		pc.Set(customBlockInfoKey, nil)
	}
}

func (b *katexBlockParser) CanInterruptParagraph() bool {
	return true
}

func (b *katexBlockParser) CanAcceptIndentedLine() bool {
	return false
}

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

		val, err := r.inlineCache.Get(context.WithValue(context.Background(), passthroughRenderCtxNode, n), rawVal)
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
		val, err := r.displayCache.Get(context.WithValue(context.Background(), passthroughRenderCtxNode, n), rawVal)
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

	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(&katexBlockParser{}, 201),
		),
	)

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

func preserveLeadingTabInCodeBlock(segment *text.Segment, reader text.Reader, indent int) {
	offsetWithPadding := reader.LineOffset() + indent
	sl, ss := reader.Position()
	reader.SetPosition(sl, text.NewSegment(ss.Start-1, ss.Stop))
	if offsetWithPadding == reader.LineOffset() {
		segment.Padding = 0
		segment.Start--
	}
	reader.SetPosition(sl, ss)
}
