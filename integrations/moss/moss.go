package moss

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net"
	"strings"

	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	MossUserID = config.GenFlag("integrations.moss.user_id", -1, "User ID for MOSS Plagiarism Checker")

	ErrUnauthed        = errors.New("unauthenticated to MOSS")
	ErrUnsupportedLang = errors.New("unsupported language")

	defaultDialer net.Dialer
)

const (
	serverAddr = "moss.stanford.edu:7690"
)

type Options struct {
	// "l" - default C++ - must be given MOSSName
	LanguageName string
	// "m" - default 10
	Sensitivity int
	// "c" - default empty
	Comment string
	// "x" - default false
	ExperimentalServer bool
	// "n" - default 250
	MatchingFileLimit int

	Files iter.Seq[*File]
}

type File struct {
	Lang     string
	Filename string
	Data     []byte
}

type Conn struct {
	ctx  context.Context
	conn net.Conn
	sc   *bufio.Scanner
}

func (m *Conn) recvLine() (string, error) {
	if !m.sc.Scan() {
		return "", m.sc.Err()
	}
	return strings.TrimSpace(m.sc.Text()), nil
}

// langName must be the MOSS name and not kilonova name
func NewFile(langName string, filename string, data []byte) *File {
	return &File{
		Lang:     langName,
		Filename: strings.ReplaceAll(filename, " ", "_"),
		Data:     data,
	}
}

func (m *Conn) Process(conf *Options) (string, error) {
	if _, err := fmt.Fprintf(m.conn, "moss %d\n", MossUserID.Value()); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(m.conn, "directory 0\n"); err != nil {
		return "", err
	}
	var exp int = 0
	if conf.ExperimentalServer {
		exp = 1
	}
	if _, err := fmt.Fprintf(m.conn, "X %d\n", exp); err != nil {
		return "", err
	}
	if conf.Sensitivity == 0 {
		conf.Sensitivity = 10
	}
	if _, err := fmt.Fprintf(m.conn, "maxmatches %d\n", conf.Sensitivity); err != nil {
		return "", err
	}
	if conf.MatchingFileLimit == 0 {
		conf.MatchingFileLimit = 250
	}
	if _, err := fmt.Fprintf(m.conn, "show %d\n", conf.MatchingFileLimit); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(m.conn, "language %s\n", conf.LanguageName); err != nil {
		return "", err
	}

	val, err := m.recvLine()
	if err != nil {
		return "", err
	}
	if val == "no" {
		return "", ErrUnsupportedLang
	}

	i := 1
	for file := range conf.Files {
		if _, err := fmt.Fprintf(m.conn, "file %d %s %d %s\n", i, file.Lang, len(file.Data), file.Filename); err != nil {
			return "", err
		}
		_, err := m.conn.Write(file.Data)
		if err != nil {
			return "", err
		}
		i++
	}

	if _, err := fmt.Fprintf(m.conn, "query 0 %s\n", conf.Comment); err != nil {
		return "", err
	}
	slog.DebugContext(m.ctx, "MOSS Query sent. Waiting for response")
	return m.recvLine()
}

func (m *Conn) Close() error {
	_, err := m.conn.Write([]byte("end\n"))
	if err1 := m.conn.Close(); err == nil && err1 != nil {
		err = err1
	}
	return err
}

func New(ctx context.Context) (*Conn, error) {
	if MossUserID.Value() <= 0 {
		return nil, ErrUnauthed
	}
	conn, err := defaultDialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	return &Conn{ctx, conn, bufio.NewScanner(conn)}, nil
}
