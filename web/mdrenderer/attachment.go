package mdrenderer

import (
	"fmt"
	"net/url"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Attachment quick image embeds are of the form ~[att.jpg]

var _ goldmark.Extender = &attNode{}
var _ renderer.NodeRenderer = &attachmentRenderer{}
var _ parser.InlineParser = &attachmentParser{}

var attNodeKind = ast.NewNodeKind("attachment")

type attachmentParser struct{}

func (_ attachmentParser) Trigger() []byte {
	return []byte{'~'}
}

func (_ attachmentParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
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
	fmt.Fprintf(writer, `<problem-attachment attname="%s"></problem-attachment>`, url.PathEscape(node.Filename))
	return ast.WalkContinue, nil
}

type attNode struct{}

func (_ *attNode) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&attachmentRenderer{}, 900)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&attachmentParser{}, 900)))
}

type AttachmentNode struct {
	ast.BaseInline

	Filename string
}

func (att *AttachmentNode) Dump(source []byte, level int) {
	ast.DumpHelper(att, source, level, map[string]string{"filename": att.Filename}, nil)
}

func (_ AttachmentNode) Kind() ast.NodeKind {
	return attNodeKind
}
