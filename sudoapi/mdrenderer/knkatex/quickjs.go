package knkatex

import (
	"errors"
	"html"
	"io"
	"runtime"
	"strings"

	"github.com/lithdew/quickjs"
)

func RenderQuickJS(w io.Writer, src []byte, display bool) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	runtime := quickjs.NewRuntime()
	defer runtime.Free()

	context := runtime.NewContext()
	defer context.Free()

	globals := context.Globals()

	result, err := context.Eval(code)
	if err != nil {
		return err
	}
	defer result.Free()

	globals.Set("_EqSrc3120", context.String(string(src)))
	if display {
		result, err = context.Eval("katex.renderToString(_EqSrc3120, { displayMode: true })")
	} else {
		result, err = context.Eval("katex.renderToString(_EqSrc3120)")
	}
	defer result.Free()
	if err != nil {
		var evalErr *quickjs.Error
		if errors.As(err, &evalErr) {
			io.WriteString(w, "<code>"+html.EscapeString(strings.TrimPrefix(evalErr.Cause, "ParseError: "))+"</code>")
		}
		return err
	}

	_, err = io.WriteString(w, result.String())
	return err
}
