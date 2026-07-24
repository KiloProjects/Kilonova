package scratch

import (
	"errors"
	"fmt"
	"io"
	"net"
	"path"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPConfig configures the platform-side remote scratch over SFTP.
type SFTPConfig struct {
	Addr       string        // grader host:port for the sftp subsystem
	User       string        // ssh user
	PrivateKey []byte        // PEM-encoded ssh private key
	HostKey    []byte        // authorized grader host key; empty => unverified (rely on network segmentation)
	BaseDir    string        // remote scratch dir prefix; "" if the sftp user is chrooted to it
	MaxConns   int           // pooled ssh connections (>=1)
	Timeout    time.Duration // per-operation deadline
}

var _ eval.Scratch = (*sftpScratch)(nil)

// sftpScratch implements eval.Scratch against a remote grader's sftp subsystem.
// Bytes are streamed; the RPC control plane never carries file contents.
type sftpScratch struct {
	cfg  SFTPConfig
	pool *sftpPool
}

func NewSFTP(cfg SFTPConfig) (eval.Scratch, error) {
	if cfg.MaxConns < 1 {
		cfg.MaxConns = 4
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	signer, err := ssh.ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse ssh private key: %w", err)
	}
	hostKeyCallback := ssh.InsecureIgnoreHostKey() // ponytail: unverified host key; network segmentation (design D4) is the backstop. Upgrade: pin cfg.HostKey.
	if len(cfg.HostKey) > 0 {
		pk, err := ssh.ParsePublicKey(cfg.HostKey)
		if err != nil {
			return nil, fmt.Errorf("parse grader host key: %w", err)
		}
		hostKeyCallback = ssh.FixedHostKey(pk)
	}
	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.Timeout,
	}
	return &sftpScratch{
		cfg:  cfg,
		pool: newSFTPPool(cfg.Addr, clientCfg, cfg.MaxConns),
	}, nil
}

func (s *sftpScratch) path(identifier string) string {
	return path.Join(s.cfg.BaseDir, identifier)
}

func (s *sftpScratch) SaveFile(r io.Reader) (string, error) {
	identifier := uuid.Must(uuid.NewV7()).String()
	conn, err := s.pool.get(s.cfg.Timeout)
	if err != nil {
		return "", err
	}
	// The whole write (create -> copy -> close) must finish before we return so
	// the bytes are on the grader's disk before any RunBox3 references the id.
	err = conn.deadlined(s.cfg.Timeout, func() error {
		f, err := conn.sftp.Create(s.path(identifier))
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, r); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
	if err != nil {
		s.pool.discard(conn)
		return "", err
	}
	s.pool.put(conn)
	return identifier, nil
}

func (s *sftpScratch) ReadFile(identifier string) (io.ReadCloser, error) {
	conn, err := s.pool.get(s.cfg.Timeout)
	if err != nil {
		return nil, err
	}
	var f *sftp.File
	err = conn.deadlined(s.cfg.Timeout, func() error {
		f, err = conn.sftp.Open(s.path(identifier))
		return err
	})
	if err != nil {
		s.pool.discard(conn)
		return nil, err
	}
	// The pooled conn stays checked out until the caller closes the file.
	return &pooledFile{f: f, conn: conn, pool: s.pool, timeout: s.cfg.Timeout}, nil
}

func (s *sftpScratch) DeleteFile(identifier string) error {
	conn, err := s.pool.get(s.cfg.Timeout)
	if err != nil {
		return err
	}
	err = conn.deadlined(s.cfg.Timeout, func() error {
		return conn.sftp.Remove(s.path(identifier))
	})
	if err != nil {
		s.pool.discard(conn)
		return err
	}
	s.pool.put(conn)
	return nil
}

// pooledFile keeps its conn checked out for the read's lifetime, returning it
// on Close so ReadFile can stream without racing the pool.
type pooledFile struct {
	f       *sftp.File
	conn    *sftpConn
	pool    *sftpPool
	timeout time.Duration
}

func (p *pooledFile) Read(b []byte) (int, error) { return p.f.Read(b) }

func (p *pooledFile) Close() error {
	err := p.conn.deadlined(p.timeout, func() error { return p.f.Close() })
	if err != nil {
		p.pool.discard(p.conn)
		return err
	}
	p.pool.put(p.conn)
	return nil
}

// --- pool ---

type sftpConn struct {
	ssh  *ssh.Client
	sftp *sftp.Client
	raw  net.Conn
}

// deadlined bounds fn with a wall-clock deadline on the underlying conn. On
// timeout the conn errors out (and the caller discards it), so no op hangs
// indefinitely — the FUSE failure mode we rejected in design D2.
func (c *sftpConn) deadlined(timeout time.Duration, fn func() error) error {
	if err := c.raw.SetDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	err := fn()
	// Best-effort clear; if the conn is dead the caller discards it anyway.
	_ = c.raw.SetDeadline(time.Time{})
	return err
}

func (c *sftpConn) close() {
	if c.sftp != nil {
		c.sftp.Close()
	}
	if c.ssh != nil {
		c.ssh.Close()
	}
}

type sftpPool struct {
	addr string
	cfg  *ssh.ClientConfig

	sem  chan struct{} // caps concurrent live conns at max
	mu   sync.Mutex
	idle []*sftpConn
}

func newSFTPPool(addr string, cfg *ssh.ClientConfig, max int) *sftpPool {
	return &sftpPool{addr: addr, cfg: cfg, sem: make(chan struct{}, max)}
}

func (p *sftpPool) dial(timeout time.Duration) (*sftpConn, error) {
	raw, err := net.DialTimeout("tcp", p.addr, timeout)
	if err != nil {
		return nil, err
	}
	if err := raw.SetDeadline(time.Now().Add(timeout)); err != nil {
		raw.Close()
		return nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(raw, p.addr, p.cfg)
	if err != nil {
		raw.Close()
		return nil, err
	}
	_ = raw.SetDeadline(time.Time{})
	client := ssh.NewClient(sshConn, chans, reqs)
	sc, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &sftpConn{ssh: client, sftp: sc, raw: raw}, nil
}

func (p *sftpPool) get(timeout time.Duration) (*sftpConn, error) {
	select {
	case p.sem <- struct{}{}:
	case <-time.After(timeout):
		return nil, errors.New("sftp pool: timed out waiting for a connection slot")
	}
	p.mu.Lock()
	if n := len(p.idle); n > 0 {
		conn := p.idle[n-1]
		p.idle = p.idle[:n-1]
		p.mu.Unlock()
		return conn, nil
	}
	p.mu.Unlock()
	conn, err := p.dial(timeout)
	if err != nil {
		<-p.sem
		return nil, err
	}
	return conn, nil
}

func (p *sftpPool) put(conn *sftpConn) {
	p.mu.Lock()
	p.idle = append(p.idle, conn)
	p.mu.Unlock()
	<-p.sem
}

// discard drops a conn presumed dead so the next get redials a fresh one.
func (p *sftpPool) discard(conn *sftpConn) {
	conn.close()
	<-p.sem
}
