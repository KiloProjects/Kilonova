// Code generated by templ - DO NOT EDIT.

// templ: version: v0.3.920
package views

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import (
	"context"
	"github.com/alecthomas/chroma/v2"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"io"
	"strings"
	"unicode"
)

func Submission() templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		return nil
	})
}

var fmt = chtml.New(chtml.WithClasses(true), chtml.TabWidth(4))

func SyntaxHighlight(code []byte, lang string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if lang == "pascal" {
			lang = "pas"
		}
		if lang == "nodejs" {
			lang = "js"
		}

		lm := lexers.Get(strings.TrimFunc(lang, unicode.IsDigit))
		if lm == nil {
			lm = lexers.Fallback
		}

		lm = chroma.Coalesce(lm)
		it, err := lm.Tokenise(nil, string(code))
		if err != nil {
			return err
		}
		if err := fmt.Format(w, styles.Get("github"), it); err != nil {
			return err
		}

		return nil
	})
}

var _ = templruntime.GeneratedTemplate
