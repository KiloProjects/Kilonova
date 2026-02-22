package sudoapi

import (
	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RenderMarkdown(src []byte, ctx *kilonova.MarkdownRenderContext) ([]byte, error) {
	return s.rd.Render(src, ctx)
}
