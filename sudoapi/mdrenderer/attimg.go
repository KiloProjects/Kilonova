package mdrenderer

import (
	"context"
	"fmt"
	"html"
	"log/slog"
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

// Image attachments are of the form ~[name.xyz]

var _ goldmark.Extender = &attNode{}
var _ renderer.NodeRenderer = &imgAttRenderer{}
var _ parser.InlineParser = &imgAttParser{}

var attNodeKind = ast.NewNodeKind("img_att")

type imgAttParser struct{}

func (imgAttParser) Trigger() []byte {
	return []byte{'~'}
}

func (imgAttParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
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
	return &ImageAttNode{Filename: string(fileName)}
}

type imgAttRenderer struct{}

func (att *imgAttRenderer) RegisterFuncs(rd renderer.NodeRendererFuncRegisterer) {
	rd.Register(attNodeKind, att.renderAttachment)
}

func (att *imgAttRenderer) renderAttachment(writer util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	align := "left"
	width := ""
	var inline bool
	var rmTransparency bool

	node := n.(*ImageAttNode)
	parts := strings.Split(node.Filename, "|")
	name := parts[0]
	if len(parts) > 1 {
		for _, part := range parts {
			key, val, found := strings.Cut(part, "=")
			if found {
				switch key {
				case "align":
					if align == "left" || align == "right" || align == "center" {
						align = val
					}
				case "width":
					width = val
				case "inline":
					inline = true
				case "noTransparency":
					rmTransparency = true
				}
			} else if key == "inline" {
				inline = true
			}
		}
	}
	ctx, ok := n.OwnerDocument().Meta()["ctx"].(*kilonova.MarkdownRenderContext)
	link := attachmentURL(ctx, ok, name)

	extra := ""
	if inline {
		extra += ` data-imginline="true" `
	}
	if rmTransparency {
		link += "?rmTransparency=true"
	}
	if width != "" {
		extra += ` style="width:` + html.EscapeString(width) + `" `
	}
	fmt.Fprintf(writer, `<img src="%s" data-imgalign="%s" %s></img>`, link, align, extra)
	return ast.WalkContinue, nil
}

type attNode struct{}

func (*attNode) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&imgAttRenderer{}, 300)))
	md.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&imgAttParser{}, 300)))
}

type ImageAttNode struct {
	ast.BaseInline

	Filename string
}

func (att *ImageAttNode) Dump(source []byte, level int) {
	ast.DumpHelper(att, source, level, map[string]string{"filename": att.Filename}, nil)
}

func (att *ImageAttNode) Kind() ast.NodeKind {
	return attNodeKind
}

func attachmentURL(ctx *kilonova.MarkdownRenderContext, okCtx bool, name string) string {
	if !okCtx || ctx == nil {
		return url.PathEscape(name)
	}
	if ctx.Problem != nil {
		return fmt.Sprintf("/assets/problem/%d/attachment/%s", ctx.Problem.ID, url.PathEscape(name))
	}

	if ctx.BlogPost != nil {
		return fmt.Sprintf("/assets/blogPost/%s/attachment/%s", ctx.BlogPost.Slug, url.PathEscape(name))
	}

	slog.WarnContext(context.TODO(), "Unexpected attachment URL state")
	return url.PathEscape(name)
}
