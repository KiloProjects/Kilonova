package db

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"

	"github.com/jackc/pgx/v4"
)

var _ io.ReadWriteCloser = &File{}

type File struct {
	ctx context.Context
	tx  *sql.Tx
	fd  int
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.fd == -1 {
		return -1, fs.ErrClosed
	}
	var res []byte
	err = f.tx.QueryRowContext(f.ctx, "SELECT loread($1, $2)", f.fd, len(p)).Scan(&res)
	copy(p, res)
	if err != nil {
		return len(res), err
	}

	if len(res) < len(p) {
		err = io.EOF
	}
	return len(res), err
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.fd == -1 {
		return -1, fs.ErrClosed
	}
	err = f.tx.QueryRowContext(f.ctx, "SELECT lowrite($1, $2)", f.fd, p).Scan(&n)
	if err != nil {
		return n, err
	}

	if n < 0 {
		return 0, errors.New("failed to write to large object")
	}

	return n, nil
}

func (f *File) Truncate(size int) (err error) {
	if f.fd == -1 {
		return fs.ErrClosed
	}
	_, err = f.tx.ExecContext(f.ctx, "SELECT lo_truncate($1, $2)", f.fd, size)
	return err
}

func (f *File) Close() error {
	f.fd = -1
	return f.tx.Commit()
}

func (s *DB) OpenFile(ctx context.Context, oid int, mode pgx.LargeObjectMode) (*File, error) {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	var fd int
	err = tx.QueryRowContext(ctx, "SELECT lo_open($1, $2)", oid, mode).Scan(&fd)
	if err != nil {
		return nil, err
	}
	return &File{ctx, tx, fd}, nil
}

/*
func (s *DB) CreateFile(ctx context.Context) (uint32, error) {
	var oid uint32
	err := s.conn.QueryRowContext(ctx, "SELECT lo_create(0)").Scan(&oid)
	return oid, err
}
*/
