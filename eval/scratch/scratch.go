package scratch

import (
	"io"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

var _ eval.Scratch = (*scratch)(nil)

func New(fs afero.Fs) eval.Scratch {
	return &scratch{fs}
}

type scratch struct {
	fs afero.Fs
}

func (s *scratch) genIdentifier() string {
	return uuid.Must(uuid.NewV7()).String()
}

func (s *scratch) SaveFile(r io.Reader) (string, error) {
	identifier := s.genIdentifier()
	return identifier, afero.WriteReader(s.fs, identifier, r)
}

func (s *scratch) ReadFile(identifier string) (io.ReadCloser, error) {
	f, err := s.fs.Open(identifier)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (s *scratch) DeleteFile(identifier string) error {
	return s.fs.RemoveAll(identifier)
}
