// Code in render.go has mostly been derived from [FurqanSoftware/goldmark-latex](https://github.com/FurqanSoftware/goldmark-katex/blob/master/katex.go).
package knkatex

import (
	_ "embed"
	"io"

	"github.com/KiloProjects/kilonova/internal/config"
)

//go:embed katex.min.js
var code string

var UseQuickJS = config.GenFlag[bool]("behavior.markdown.qjs_katex", false, "Use QuickJS for rendering KaTeX math (requires cgo)")

func Render(w io.Writer, src []byte, display bool) error {
	if UseQuickJS.Value() {
		return RenderQuickJS(w, src, display)
	}
	return RenderGoja(w, src, display)
}
