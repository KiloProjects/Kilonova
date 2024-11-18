package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

var (
	configPath string
	Common     CommonConf
	Eval       EvalConf
	Email      EmailConf
	Frontend   FrontendConf
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
}

// EvalConf is the data required for the eval service
type EvalConf struct {
	// Address       string `toml:"address"`
	NumConcurrent int   `toml:"num_concurrent"`
	GlobalMaxMem  int64 `toml:"global_max_mem_kb"`

	StartingBox int `toml:"starting_box"`
}

// CommonConf is the data required for all services
type CommonConf struct {
	LogDir      string `toml:"log_dir"`
	DataDir     string `toml:"data_dir"`
	Debug       bool   `toml:"debug"`
	HostPrefix  string `toml:"host_prefix"`
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
	Common = c.Common
	Email = c.Email
	Eval = c.Eval
	Frontend = c.Frontend
}

func compactify() {
	c.Common = Common
	c.Email = Email
	c.Eval = Eval
	c.Frontend = Frontend
}

func SetConfigPath(path string) {
	configPath = path
}

func Save() error {
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

func Load() error {
	if configPath == "" {
		return errors.New("invalid config path")
	}
	md, err := toml.DecodeFile(configPath, &c)
	if len(md.Undecoded()) > 0 {
		zap.S().Warn("NOTE: There were some undecoded keys")
		spew.Dump(md.Undecoded())
	}
	if err == nil {
		if c.Common.DefaultLang == "" {
			zap.S().Warn("No default language set, defaulting to English")
			c.Common.DefaultLang = "en"
		}
		if !(c.Common.DefaultLang == "en" || c.Common.DefaultLang == "ro") {
			zap.S().Warnf("Invalid language %q\n", c.Common.DefaultLang)
		}
		spread()
	}
	return err
}
