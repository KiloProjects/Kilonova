// Package config holds the process-wide configuration: startup-only settings
// populated from KN_* environment variables (declared as CLI flags in cmd/kn,
// documented in .env.example) and the runtime-editable flag registry persisted
// to flags.json (config_v2.go). Nothing here is read from or written to a
// config file.
package config

import (
	"errors"
	"path/filepath"
)

var (
	Common CommonConf
	Eval   EvalConf
	Email  EmailConf
)

// EmailConf is the SMTP wiring for the mailer.
type EmailConf struct {
	Host     string // host:port
	Username string
	Password string
	SendAs   string
}

// Enabled reports whether a mailer should be constructed at all.
func (e EmailConf) Enabled() bool { return e.Host != "" }

// EvalConf is the data required for the eval service
type EvalConf struct {
	// Mode selects the grader: "local" (default, in-process) or "remote".
	// In local mode the sandbox fields below drive the in-process BoxManager and
	// Remote is ignored; in remote mode the grader process owns them and only
	// Remote is used.
	Mode string

	NumConcurrent int
	GlobalMaxMem  int64
	StartingBox   int

	Remote RemoteEvalConf
}

// IsRemote reports whether the platform should talk to a remote grader.
func (e EvalConf) IsRemote() bool { return e.Mode == "remote" }

// RemoteEvalConf tells the platform how to reach a remote grader. The JSON
// control plane and the /scratch data plane both live on Endpoint, behind the
// same TLS cert and bearer token — no separate data-plane connection.
type RemoteEvalConf struct {
	Endpoint string // grader base URL, e.g. https://grader:9000
	Token    string // grader-minted bearer token for this platform instance
}

// CommonConf is the data required for all services
type CommonConf struct {
	DataDir string
	DBDSN   string
}

// LogDir is where rotating log files go: always <DataDir>/logs.
func (c CommonConf) LogDir() string { return filepath.Join(c.DataDir, "logs") }

// ScratchDir is the remote grader's scratch root: always <DataDir>/scratch.
func (c CommonConf) ScratchDir() string { return filepath.Join(c.DataDir, "scratch") }

// RequireDataDir is the single check every process runs before touching disk.
func RequireDataDir() error {
	if !filepath.IsAbs(Common.DataDir) {
		return errors.New("KN_DATA_DIR must be set to an absolute path (run `kn config-migrate` to convert a legacy config.toml)")
	}
	return nil
}
