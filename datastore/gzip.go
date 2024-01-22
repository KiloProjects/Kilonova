package datastore

import (
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"

	"go.uber.org/zap"
	"vimagination.zapto.org/dos2unix"
)

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

func openGzipOrNormal(fpath string) (io.ReadCloser, error) {
	f, err := os.Open(fpath + ".gz")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return os.Open(fpath)
		}
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	return &gzipFileReader{f, gz}, nil
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

func writeCompressedFile(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path+".gz", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	_, err = io.Copy(gz, dos2unix.DOS2Unix(r))
	if err1 := gz.Close(); err1 != nil && err == nil {
		err = err1
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}
