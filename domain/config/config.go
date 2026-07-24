package config

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/KiloProjects/kilonova"
)

var (
	Common   CommonConf
	Eval     EvalConf
	Email    EmailConf
	Frontend FrontendConf
)

// configStruct is the glue for all configuration sections when unmarshaling
// After load, it will disperse all the data in variables
type configStruct struct {
	Common   CommonConf   `toml:"common"`
	Eval     EvalConf     `toml:"eval"`
	Email    EmailConf    `toml:"email"`
	Frontend FrontendConf `toml:"frontend"`
}

// EmailConf is the data required for the email part
type EmailConf struct {
	Enabled bool `toml:"enabled"`

	Host     string `toml:"host"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	SendAs   string `toml:"sendAs"`
}

// EvalConf is the data required for the eval service
type EvalConf struct {
	// Mode selects the grader: "local" (default, in-process) or "remote".
	// In local mode the fields below drive the in-process BoxManager and Remote
	// is ignored; in remote mode execution settings live in the grader's own
	// config file and only Remote is used.
	Mode string `toml:"mode"`

	NumConcurrent int   `toml:"num_concurrent"`
	GlobalMaxMem  int64 `toml:"global_max_mem_kb"`

	StartingBox int `toml:"starting_box"`

	Remote RemoteEvalConf `toml:"remote"`
}

// IsRemote reports whether the platform should talk to a remote grader.
func (e EvalConf) IsRemote() bool { return e.Mode == "remote" }

// RemoteEvalConf tells the platform how to reach a remote grader.
type RemoteEvalConf struct {
	Endpoint string   `toml:"endpoint"` // ConnectRPC base URL, e.g. https://grader:9000
	Token    string   `toml:"token"`    // grader-minted bearer token for this platform instance
	SFTP     SFTPConf `toml:"sftp"`
}

// SFTPConf is the data-plane connection to the grader's scratch directory.
type SFTPConf struct {
	Addr        string `toml:"addr"`          // grader host:port for the sftp subsystem
	User        string `toml:"user"`          // ssh user
	KeyPath     string `toml:"key_path"`      // path to the ssh private key
	HostKeyPath string `toml:"host_key_path"` // optional authorized host key for pinning
	ScratchBase string `toml:"scratch_base"`  // remote scratch dir prefix ("" if the user is chrooted to it)
	MaxConns    int    `toml:"max_conns"`     // pooled ssh connections (default 4)
	TimeoutSec  int    `toml:"timeout_sec"`   // per-operation deadline in seconds (default 30)
}

// CommonConf is the data required for all services
type CommonConf struct {
	LogDir  string `toml:"log_dir"`
	DataDir string `toml:"data_dir"`
	// Debug is deprecated: use [kilonova.DebugMode] for debug state instead
	Debug bool `toml:"debug"`
	// HostPrefix is deprecated: use [kilonova.HostPrefix] for host prefix instead
	HostPrefix string   `toml:"host_prefix"`
	HostURL    *url.URL `toml:"-"`
	// DefaultLang is deprecated: use [kilonova.DefaultLang] for default language instead
	DefaultLang string `toml:"default_language"`

	DBDSN string `toml:"db_dsn"`

	TestMaxMemKB int `toml:"test_max_mem_kb"`
}

type FrontendConf struct {
	// Note that BannedHotProblems only counts for problems that are sorted
	// using the hotness filter (that is, had submissions in the last 7 days)
	// Basically, banned problems are considered to have had 0 submissions in the last 7 days
	BannedHotProblems []int `toml:"banned_hot_problems"`
}

// c represents the loaded config
var c configStruct

func spread() {
	kilonova.SetDebugMode(c.Common.Debug)
	kilonova.SetHostPrefix(c.Common.HostPrefix)
	kilonova.SetDefaultLanguage(c.Common.DefaultLang)
	Common = c.Common
	Email = c.Email
	Eval = c.Eval
	Frontend = c.Frontend
}

func compactify() {
	c.Common.Debug = kilonova.DebugMode()
	c.Common.DefaultLang = kilonova.DefaultLanguage()
	c.Common.HostPrefix = kilonova.HostPrefix()
	c.Common = Common
	c.Email = Email
	c.Eval = Eval
	c.Frontend = Frontend
}

func Save(configPath string) error {
	compactify()
	if configPath == "" {
		return errors.New("invalid config path")
	}

	// Make the directories just in case they don't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0666); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}

	enc := toml.NewEncoder(file)
	enc.Indent = " "
	if err := enc.Encode(c); err != nil {
		file.Close() // We don't care if it errors out, it's over anyway
		return err
	}

	return file.Close()
}

func Load(ctx context.Context, configPath string) error {
	if configPath == "" {
		return errors.New("invalid config path")
	}
	md, err := toml.DecodeFile(configPath, &c)
	if err != nil {
		slog.ErrorContext(ctx, "Couldn't load config file", slog.Any("err", err))
		return err
	}
	if len(md.Undecoded()) > 0 {
		slog.InfoContext(ctx, "There were some undecoded keys: ", slog.Any("keys", md.Undecoded()))
	}
	if c.Common.DefaultLang == "" {
		slog.WarnContext(ctx, "No default language set, defaulting to English")
		c.Common.DefaultLang = "en"
	}
	spread()
	return nil
}
