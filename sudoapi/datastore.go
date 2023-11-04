package sudoapi

import (
	"io"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

var _ kilonova.DataStore = &BaseAPI{}

func (s *BaseAPI) TestInput(testID int) (io.ReadCloser, error) {
	r, err := s.manager.TestInput(testID)
	if err != nil {
		return nil, WrapError(err, "Could not open test input")
	}
	return r, nil
}

func (s *BaseAPI) TestOutput(testID int) (io.ReadCloser, error) {
	r, err := s.manager.TestOutput(testID)
	if err != nil {
		return nil, WrapError(err, "Could not open test input")
	}
	return r, nil
}

func (s *BaseAPI) PurgeTestData(testID int) error {
	err := s.manager.PurgeTestData(testID)
	if err != nil {
		return WrapError(err, "Could not purge test data")
	}
	return nil
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

func (s *BaseAPI) SubtestReader(subtest int) (io.ReadCloser, error) {
	r, err := s.manager.SubtestReader(subtest)
	if err != nil {
		return nil, WrapError(err, "Could not open subtest reader")
	}
	return r, nil
}

func (s *BaseAPI) HasAttachmentRender(attID int) bool {
	return s.manager.HasAttachmentRender(attID)
}

func (s *BaseAPI) GetAttachmentRender(attID int) (io.ReadCloser, error) {
	f, err := s.manager.GetAttachmentRender(attID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get rendered attachment")
	}
	return f, nil
}

func (s *BaseAPI) DelAttachmentRender(attID int) error {
	if err := s.manager.DelAttachmentRender(attID); err != nil {
		zap.S().Warn("Couldn't delete rendered attachment: ", err)
		return WrapError(err, "Couldn't delete rendered attachment")
	}
	return nil
}

func (s *BaseAPI) SaveAttachmentRender(attID int, data []byte) error {
	if err := s.manager.SaveAttachmentRender(attID, data); err != nil {
		zap.S().Warn("Couldn't save rendered attachment: ", err)
		return WrapError(err, "Couldn't delete rendered attachment")
	}
	return nil
}

func (s *BaseAPI) InvalidateAllAttachments() error {
	if err := s.manager.InvalidateAllAttachments(); err != nil {
		zap.S().Warn("Couldn't invalidate all attachments: ", err)
		return WrapError(err, "Couldn't invalidate attachment renders")
	}
	return nil
}
