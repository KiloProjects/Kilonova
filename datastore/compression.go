package datastore

import (
	"io"
	"os"
	"sync"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"

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

type zstdFileReader struct {
	f  *os.File
	zr *zstd.Decoder
}

func (fr *zstdFileReader) Read(p []byte) (int, error) {
	return fr.zr.Read(p)
}

func (fr *zstdFileReader) WriteTo(w io.Writer) (int64, error) {
	return fr.zr.WriteTo(w)
}

func (fr *zstdFileReader) Close() error {
	err := fr.f.Close()
	fr.zr.Close()
	return err
}

// zstd.NewReader acts weirdly with sync.Pool, TODO: we need a better pooling mechanism
func newZstdReader(r io.Reader) *zstd.Decoder {
	zr, err := zstd.NewReader(r, zstd.WithDecoderConcurrency(1))
	if err != nil {
		panic(err) // There's something wrong with the options, nothing else can give an error
	}
	return zr
}
