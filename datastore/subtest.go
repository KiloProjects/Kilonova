package datastore

import (
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"

	"go.uber.org/zap"
)

// SubtestWriter should be used by the eval server
func (m *StorageManager) SubtestWriter(subtest int) (io.WriteCloser, error) {
	f, err := os.OpenFile(m.SubtestPath(subtest)+".gz", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	gz, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	return &gzipFileWriter{f, gz}, nil
}

// SubtestReader should be used by the grader
func (m *StorageManager) SubtestReader(subtest int) (io.ReadCloser, error) {
	return openNormalOrGzip(m.SubtestPath(subtest))
}

func (m *StorageManager) SubtestPath(subtest int) string {
	return path.Join(m.RootPath, "subtests", strconv.Itoa(subtest))
}

type gzipFileWriter struct {
	f  *os.File
	gz *gzip.Writer
}

func (fw *gzipFileWriter) Write(p []byte) (int, error) {
	return fw.gz.Write(p)
}

func (fw *gzipFileWriter) Close() error {
	err2 := fw.gz.Close()
	err := fw.f.Close()
	if err == nil && err2 != nil {
		err = err2
		zap.S().Warn(err2)
	}
	return err
}

type gzipFileReader struct {
	f  *os.File
	gz *gzip.Reader
}

func (fw *gzipFileReader) Read(p []byte) (int, error) {
	return fw.gz.Read(p)
}

func (fw *gzipFileReader) Close() error {
	err2 := fw.gz.Close()
	err := fw.f.Close()
	if err == nil && err2 != nil {
		err = err2
		zap.S().Warn(err2)
	}
	return err
}

func openNormalOrGzip(fpath string) (io.ReadCloser, error) {
	f, err := os.Open(fpath)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		f2, err := os.Open(fpath + ".gz")
		if err != nil {
			return f2, err
		}
		gz, err := gzip.NewReader(f2)
		if err != nil {
			return nil, err
		}
		return &gzipFileReader{f2, gz}, nil
	}
	return f, err
}
