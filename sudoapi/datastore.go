package sudoapi

import (
	"io"

	"github.com/KiloProjects/kilonova"
)

var _ kilonova.DataStore = &BaseAPI{}

func (s *BaseAPI) TestInput(testID int) (io.ReadSeekCloser, error) {
	r, err := s.manager.TestInput(testID)
	if err != nil {
		return nil, WrapError(err, "Could not open test input")
	}
	return r, nil
}

func (s *BaseAPI) TestOutput(testID int) (io.ReadSeekCloser, error) {
	r, err := s.manager.TestOutput(testID)
	if err != nil {
		return nil, WrapError(err, "Could not open test input")
	}
	return r, nil
}

func (s *BaseAPI) SaveTestInput(testID int, input io.Reader) error {
	if err := s.manager.SaveTestInput(testID, input); err != nil {
		return WrapError(err, "Could not save test input")
	}
	return nil
}

func (s *BaseAPI) SaveTestOutput(testID int, output io.Reader) error {
	if err := s.manager.SaveTestOutput(testID, output); err != nil {
		return WrapError(err, "Could not save test output")
	}
	return nil
}

func (s *BaseAPI) SubtestWriter(subtest int) (io.WriteCloser, error) {
	w, err := s.manager.SubtestWriter(subtest)
	if err != nil {
		return nil, WrapError(err, "Could not open subtest writer")
	}
	return w, nil
}

func (s *BaseAPI) SubtestReader(subtest int) (io.ReadSeekCloser, error) {
	r, err := s.manager.SubtestReader(subtest)
	if err != nil {
		return nil, WrapError(err, "Could not open subtest reader")
	}
	return r, nil
}
