package knkatex

import (
	"errors"
	"html"
	"io"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"go.uber.org/zap"
)

func initGoja() *goja.Program {
	prog, err := goja.Compile("katex.min.js", code, true)
	if err != nil {
		zap.S().Warn(err)
		return nil
	}
	return prog
}

var compiledKatex = sync.OnceValue(initGoja)

var renderToString = sync.OnceValue(func() *goja.Program { return goja.MustCompile("", "katex.renderToString(_EqSrc3120)", true) })
var displayRenderToString = sync.OnceValue(func() *goja.Program {
	return goja.MustCompile("", "katex.renderToString(_EqSrc3120, {displayMode: true})", true)
})

func RenderGoja(w io.Writer, src []byte, display bool) error {
	katex := compiledKatex()
	if katex == nil {
		zap.S().Warn("No katex program provided")
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
				zap.S().Warn("Exception value is nil")
				msg = exception.Error()
			}
			io.WriteString(w, "<code>"+html.EscapeString(strings.TrimPrefix(msg, "ParseError: "))+"</code>")
		}
		return err
	}

	_, err = io.WriteString(w, val.String())
	return err
}
