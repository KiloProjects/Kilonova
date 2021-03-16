package kilonova

import "io"

type KNArchiver struct {
	problem      *Problem
	testServicer TestService
}

func (a *KNArchiver) GenArchive(w io.Writer) error {
	panic("TODO")
}

func NewKNArchiver(problem *Problem, testserv TestService) (*KNArchiver, error) {
	return &KNArchiver{problem, testserv}, nil
}
