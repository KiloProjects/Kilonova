package kilonova

import (
	"archive/zip"
	"io"
)

type KNArchiver struct {
	problem      *Problem
	testServicer TestService
}

func (a *KNArchiver) ReadArchive(r io.ReaderAt, size int64) (*Problem, []*Test, error) {
	rd, err := zip.NewReader(r, size)
	if err != nil {
		return nil, nil, err
	}
	_ = rd
	panic("TODO")
}

func (a *KNArchiver) GenArchive(w io.Writer) error {
	ar := zip.NewWriter(w)
	_ = ar
	// TODO
	return ar.Close()
}

func NewKNArchiver(problem *Problem, probserv ProblemService, testserv TestService) (*KNArchiver, error) {
	return &KNArchiver{problem, testserv}, nil
}
