package sudoapi

import (
	"fmt"
	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RenderMarkdown(src []byte, ctx *kilonova.RenderContext) ([]byte, error) {
	out, err := s.rd.Render(src, ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't render markdown: %w", err)
	}
	return out, nil
}
