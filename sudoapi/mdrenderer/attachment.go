package mdrenderer

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Attachments are of the form ~[name.xyz]

var _ goldmark.Extender = &attNode{}
var _ renderer.NodeRenderer = &attachmentRenderer{}
var _ parser.InlineParser = &attachmentParser{}

var attNodeKind = ast.NewNodeKind("attachment")

type attachmentParser struct{}

func (attachmentParser) Trigger() []byte {
	return []byte{'~'}
}

func (attachmentParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	if len(line) < 2 {
		return nil
	}
	if line[1] != '[' {
		return nil
	}
	i := 2
	for ; i < len(line); i++ {
		if line[i] == ']' {
			break
		}
	}
	if i >= len(line) || line[i] != ']' {
		return nil
	}
	block.Advance(i + 1)
	fileName := line[2:i]
	return &AttachmentNode{Filename: string(fileName)}
}

type attachmentRenderer struct{}

func (att *attachmentRenderer) RegisterFuncs(rd renderer.NodeRendererFuncRegisterer) {
	rd.Register(attNodeKind, att.renderAttachment)
}

func (att *attachmentRenderer) renderAttachment(writer util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*AttachmentNode)
	parts := strings.Split(node.Filename, "|")
	name := parts[0]
	classes := ""
	styles := make([]string, 0, len(parts))
	if len(parts) > 1 {
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				if kv[0] == "class" {
					classes = kv[1]
				} else {
					styles = append(styles, fmt.Sprintf("%s:%s", kv[0], kv[1]))
				}
			}
		}
	}
	ctx, ok := n.OwnerDocument().Meta()["ctx"].(*kilonova.RenderContext)
	if !ok || ctx == nil || ctx.Problem == nil {
		fmt.Fprintf(
			writer,
			`<img src="%s" class="%s" style="%s"></img>`,
			url.PathEscape(name),
			html.EscapeString(classes),
			html.EscapeString(strings.Join(styles, ",")),
		)
		return ast.WalkContinue, nil
	}
	fmt.Fprintf(
		writer,
		`<img src="/problems/%d/attachments/%s" class="%s" style="%s"></img>`,
		ctx.Problem.ID,
		url.PathEscape(name),
		html.EscapeString(classes),
		html.EscapeString(strings.Join(styles, ";")),
	)
	return ast.WalkContinue, nil
}

type attNode struct{}

func (*attNode) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&attachmentRenderer{}, 900)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&attachmentParser{}, 900)))
}

type AttachmentNode struct {
	ast.BaseInline

	Filename string
}

func (yt *AttachmentNode) Dump(source []byte, level int) {
	ast.DumpHelper(yt, source, level, map[string]string{"filename": yt.Filename}, nil)
}

func (yt *AttachmentNode) Kind() ast.NodeKind {
	return attNodeKind
}
