package datastore

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/klauspost/compress/gzip"

	"go.uber.org/zap"
	"vimagination.zapto.org/dos2unix"
)

type gzipFileReader struct {
	f  *os.File
	gz *gzip.Reader
}

func (fr *gzipFileReader) Read(p []byte) (int, error) {
	return fr.gz.Read(p)
}

func (fr *gzipFileReader) Close() error {
	err2 := fr.gz.Close()
	err := fr.f.Close()
	if err == nil && err2 != nil {
		err = err2
		zap.S().Warn(err2)
	}
	if err2 == nil {
		// If close was successful, put the gzip.Reader back in the pool
		gzipReaderPool.Put(fr.gz)
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
	gz, err := newGzipReader(f)
	if err != nil {
		return nil, err
	}
	return &gzipFileReader{f, gz}, nil
}

var gzipReaderPool = &sync.Pool{}

func newGzipReader(r io.Reader) (*gzip.Reader, error) {
	gra := gzipReaderPool.Get()
	if gra == nil {
		return gzip.NewReader(r)
	}
	gr := gra.(*gzip.Reader)
	if err := gr.Reset(r); err != nil {
		zap.S().Warn(err)
		return nil, err
	}
	return gr, nil
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
	if err2 == nil {
		// If close was successful, put the gzip.Writer back in the pool
		gzipWriterPool.Put(fw.gz)
	}
	return err
}

func writeCompressedFile(path string, r io.Reader, perms fs.FileMode) error {
	f, err := os.OpenFile(path+".gz", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	gz := newGzipWriter(f)
	_, err = io.Copy(gz, dos2unix.DOS2Unix(r))
	if err1 := gz.Close(); err1 != nil && err == nil {
		err = err1
	}
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

var gzipWriterPool = &sync.Pool{
	New: func() any {
		return gzip.NewWriter(nil)
	},
}

func newGzipWriter(w io.Writer) *gzip.Writer {
	gw := gzipWriterPool.Get().(*gzip.Writer)
	gw.Reset(w)
	return gw
}
