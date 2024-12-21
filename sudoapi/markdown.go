package sudoapi

import "github.com/KiloProjects/kilonova"

func (s *BaseAPI) RenderMarkdown(src []byte, ctx *kilonova.RenderContext) ([]byte, error) {
	out, err := s.rd.Render(src, ctx)
	if err != nil {
		return nil, WrapError(err, "Couldn't render markdown")
	}
	return out, nil
}
