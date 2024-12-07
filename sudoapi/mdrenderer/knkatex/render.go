// Code in render.go has mostly been derived from [FurqanSoftware/goldmark-latex](https://github.com/FurqanSoftware/goldmark-katex/blob/master/katex.go).
package knkatex

import (
	"context"
	_ "embed"
	"errors"
	"html"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

//go:embed katex.min.js
var code string

func initGoja() *goja.Program {
	prog, err := goja.Compile("katex.min.js", code, true)
	if err != nil {
		slog.WarnContext(context.Background(), "Could not init goja", slog.Any("err", err))
		return nil
	}
	return prog
}

var compiledKatex = sync.OnceValue(initGoja)

var renderToString = sync.OnceValue(func() *goja.Program { return goja.MustCompile("", "katex.renderToString(_EqSrc3120)", true) })
var displayRenderToString = sync.OnceValue(func() *goja.Program {
	return goja.MustCompile("", "katex.renderToString(_EqSrc3120, {displayMode: true})", true)
})

func Render(ctx context.Context, w io.Writer, src []byte, display bool) error {
	katex := compiledKatex()
	if katex == nil {
		slog.WarnContext(ctx, "No katex program provided")
		io.WriteString(w, "<code>Failed to render KaTeX</code>")
		return nil
	}
	vm := goja.New()
	if _, err := vm.RunProgram(katex); err != nil {
		return err
	}
	vm.Set("_EqSrc3120", string(src))

	cmd := renderToString()
	if display {
		cmd = displayRenderToString()
	}
	val, err := vm.RunProgram(cmd)
	if err != nil {
		var exception *goja.Exception
		if errors.As(err, &exception) {
			val := exception.Value()
			var msg string
			if val != nil {
				msg = val.String()
			} else {
				slog.WarnContext(ctx, "Exception value is nil")
				msg = exception.Error()
			}
			io.WriteString(w, "<code>"+html.EscapeString(strings.TrimPrefix(msg, "ParseError: "))+"</code>")
		}
		return err
	}

	_, err = io.WriteString(w, val.String())
	return err
}
