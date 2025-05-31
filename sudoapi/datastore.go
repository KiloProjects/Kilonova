package sudoapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"vimagination.zapto.org/dos2unix"
)

func (s *BaseAPI) PurgeTestData(testID int) error {
	if err := errors.Join(
		s.testBucket.RemoveFile(strconv.Itoa(testID)+".in"),
		s.testBucket.RemoveFile(strconv.Itoa(testID)+".out"),
	); err != nil {
		return fmt.Errorf("could not purge test data: %w", err)
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
		return fmt.Errorf("could not save test input: %w", err)
	}
	return nil
}

func (s *BaseAPI) SaveTestOutput(testID int, output io.Reader) error {
	if err := s.testBucket.WriteFile(strconv.Itoa(testID)+".out", dos2unix.DOS2Unix(output), 0644); err != nil {
		return fmt.Errorf("could not save test output: %w", err)
	}
	return nil
}

func (s *BaseAPI) GetAttachmentRender(attID int, renderType string) (io.ReadSeekCloser, error) {
	f, err := s.attachmentCacheBucket.ReadSeeker(attachmentCacheBucketName(attID, renderType))
	if err != nil {
		return nil, fmt.Errorf("couldn't get rendered attachment: %w", err)
	}
	return f, nil
}

func (s *BaseAPI) DelAttachmentRenders(attID int) error {
	entries, err := s.attachmentCacheBucket.FileList()
	if err != nil {
		slog.WarnContext(context.Background(), "Couldn't list attachment renders", slog.Any("err", err))
		return fmt.Errorf("couldn't delete attachment renders: %w", err)
	}
	for _, entry := range entries {
		prefix, _, _ := strings.Cut(entry.Name(), ".")
		id, err := strconv.Atoi(prefix)
		if err != nil {
			slog.WarnContext(
				context.Background(),
				"Attachment renders should start with attachment ID",
				slog.String("name", entry.Name()),
				slog.Any("err", err),
			)
			continue
		}
		if id != attID {
			continue
		}
		if err := s.attachmentCacheBucket.RemoveFile(entry.Name()); err != nil {
			slog.WarnContext(context.Background(), "Couldn't delete attachment render", slog.Any("err", err))
		}
	}
	return nil
}

func (s *BaseAPI) SaveAttachmentRender(attID int, renderType string, data []byte) error {
	if err := s.attachmentCacheBucket.WriteFile(attachmentCacheBucketName(attID, renderType), bytes.NewReader(data), 0644); err != nil {
		slog.WarnContext(context.Background(), "Couldn't save rendered attachment", slog.Any("err", err))
		return fmt.Errorf("couldn't delete rendered attachment: %w", err)
	}
	return nil
}

func attachmentCacheBucketName(attID int, renderType string) string {
	return strconv.Itoa(attID) + "." + renderType
}
