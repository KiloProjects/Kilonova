package config

import (
	"context"
	"errors"
	"log/slog"

	"github.com/BurntSushi/toml"
)

// GraderConf is the remote grader's own configuration file.
type GraderConf struct {
	Grader GraderSection `toml:"grader"`
}

type GraderSection struct {
	Listen     string `toml:"listen"`      // host:port for the ConnectRPC server
	CertFile   string `toml:"cert_file"`   // TLS server cert
	KeyFile    string `toml:"key_file"`    // TLS server key
	ScratchDir string `toml:"scratch_dir"` // local scratch dir (also served over the HTTP /scratch endpoint)

	// Execution settings, moved off the platform in remote mode.
	NumConcurrent int   `toml:"num_concurrent"`
	GlobalMaxMem  int64 `toml:"global_max_mem_kb"`
	StartingBox   int   `toml:"starting_box"`

	// ScratchTTLSec is the orphan GC TTL; must be >> max eval duration.
	ScratchTTLSec int `toml:"scratch_ttl_sec"`

	Clients []GraderClientConf `toml:"client"`
}

// GraderClientConf is one entry in the token registry: a named platform client
// with its bearer token. Priority is reserved (documented, unconsumed).
type GraderClientConf struct {
	Name     string `toml:"name"`
	Token    string `toml:"token"`
	Priority string `toml:"priority"`
}

// LoadGrader reads a grader.toml.
func LoadGrader(ctx context.Context, configPath string) (*GraderConf, error) {
	if configPath == "" {
		return nil, errors.New("invalid grader config path")
	}
	var gc GraderConf
	md, err := toml.DecodeFile(configPath, &gc)
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't load grader config file", slog.Any("err", err))
		return nil, err
	}
	if len(md.Undecoded()) > 0 {
		slog.InfoContext(ctx, "Grader config: undecoded keys", slog.Any("keys", md.Undecoded()))
	}
	if gc.Grader.ScratchTTLSec <= 0 {
		gc.Grader.ScratchTTLSec = 3600 // 1h default: orders of magnitude over any eval
	}
	return &gc, nil
}
