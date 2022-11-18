package sudoapi

func (s *BaseAPI) RenderMarkdown(src []byte) ([]byte, error) {
	out, err := s.rd.Render(src)
	if err != nil {
		return nil, WrapError(err, "Couldn't render markdown")
	}
	return out, nil
}
