package datastore

import (
	"io"
	"os"

	"github.com/klauspost/compress/zstd"
)

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
