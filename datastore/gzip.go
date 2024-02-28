package datastore

import (
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/gzip"

	"go.uber.org/zap"
)

var (
	NoCompression      = gzip.NoCompression
	DefaultCompression = gzip.DefaultCompression
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
		// If gzip close was successful, put the gzip.Writer back in the pool
		gzipWriterPool.Put(fw.gz)
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
