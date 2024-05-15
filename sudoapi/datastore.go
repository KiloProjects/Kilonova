package sudoapi

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"vimagination.zapto.org/dos2unix"
)

func (s *BaseAPI) PurgeTestData(testID int) error {
	if err := errors.Join(
		s.testBucket.RemoveFile(strconv.Itoa(testID)+".in"),
		s.testBucket.RemoveFile(strconv.Itoa(testID)+".out"),
	); err != nil {
		return WrapError(err, "Could not purge test data")
	}
	return nil
}

// NOTE: If changing filename format, make sure to also change when directly accessing
func (s *BaseAPI) TestInput(testID int) (io.ReadCloser, error) {
	return s.testBucket.Reader(strconv.Itoa(testID) + ".in")
}
func (s *BaseAPI) TestOutput(testID int) (io.ReadCloser, error) {
	return s.testBucket.Reader(strconv.Itoa(testID) + ".out")
}
func (s *BaseAPI) SubtestReader(subtest int) (io.ReadCloser, error) {
	return s.subtestBucket.Reader(strconv.Itoa(subtest))
}

func (s *BaseAPI) SaveTestInput(testID int, input io.Reader) error {
	if err := s.testBucket.WriteFile(strconv.Itoa(testID)+".in", dos2unix.DOS2Unix(input), 0644); err != nil {
		return WrapError(err, "Could not save test input")
	}
	return nil
}

func (s *BaseAPI) SaveTestOutput(testID int, output io.Reader) error {
	if err := s.testBucket.WriteFile(strconv.Itoa(testID)+".out", dos2unix.DOS2Unix(output), 0644); err != nil {
		return WrapError(err, "Could not save test output")
	}
	return nil
}

func (s *BaseAPI) GetAttachmentRender(attID int, renderType string) (io.ReadSeekCloser, error) {
	f, err := s.attachmentCacheBucket.ReadSeeker(attachmentCacheBucketName(attID, renderType))
	if err != nil {
		return nil, WrapError(err, "Couldn't get rendered attachment")
	}
	return f, nil
}

func (s *BaseAPI) DelAttachmentRenders(attID int) error {
	entries, err := s.attachmentCacheBucket.FileList()
	if err != nil {
		zap.S().Warn("Couldn't delete attachment renders: ", err)
		return WrapError(err, "Couldn't delete attachment renders")
	}
	for _, entry := range entries {
		prefix, _, _ := strings.Cut(entry.Name(), ".")
		id, err := strconv.Atoi(prefix)
		if err != nil {
			zap.S().Warn("Attachment renders should start with attachment ID:", entry.Name())
			continue
		}
		if id != attID {
			continue
		}
		if err := s.attachmentCacheBucket.RemoveFile(entry.Name()); err != nil {
			zap.S().Warn("Could not delete attachment render: ", err)
		}
	}
	return nil
}

func (s *BaseAPI) SaveAttachmentRender(attID int, renderType string, data []byte) error {
	if err := s.attachmentCacheBucket.WriteFile(attachmentCacheBucketName(attID, renderType), bytes.NewReader(data), 0644); err != nil {
		zap.S().Warn("Couldn't save rendered attachment: ", err)
		return WrapError(err, "Couldn't delete rendered attachment")
	}
	return nil
}

func attachmentCacheBucketName(attID int, renderType string) string {
	return strconv.Itoa(attID) + "." + renderType
}
