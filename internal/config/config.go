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
	Features   FeaturesConf
)

// configStruct is the glue for all configuration sections when unmarshaling
// After load, it will disperse all the data in variables
type configStruct struct {
	Common   CommonConf   `toml:"common"`
	Eval     EvalConf     `toml:"eval"`
	Email    EmailConf    `toml:"email"`
	Features FeaturesConf `toml:"features"`
}

// EmailConf is the data required for the email part
type EmailConf struct {
	Host     string `toml:"host"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// EvalConf is the data required for the eval service
type EvalConf struct {
	IsolatePath   string `toml:"isolatePath"`
	CompilePath   string `toml:"compilePath"`
	Address       string `toml:"address"`
	NumConcurrent int    `toml:"num_concurrent"`
}

// CommonConf is the data required for all services
type CommonConf struct {
	LogDir      string `toml:"log_dir"`
	DataDir     string `toml:"data_dir"`
	Debug       bool   `toml:"debug"`
	HostPrefix  string `toml:"host_prefix"`
	Port        int    `toml:"port"`
	DefaultLang string `toml:"default_language"`

	DBDSN string `toml:"db_dsn"`
}

type FeaturesConf struct {
	Grader bool `toml:"grader"`
	Signup bool `toml:"manual_signup"`
	Pastes bool `toml:"pastes"`

	AllSubs bool `toml:"all_subs"`

	CCDisclaimer bool `toml:"cc_disclaimer"`
}

// c represents the loaded config
var c configStruct

func spread() {
	Common = c.Common
	Email = c.Email
	Eval = c.Eval
	Features = c.Features
}

func compactify() {
	c.Common = Common
	c.Email = Email
	c.Eval = Eval
	c.Features = Features
}

func SetConfigPath(path string) {
	configPath = path
}

func Save() error {
	compactify()
	if configPath == "" {
		return errors.New("Invalid config path")
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
		return errors.New("Invalid config path")
	}
	md, err := toml.DecodeFile(configPath, &c)
	if len(md.Undecoded()) > 0 {
		zap.S().Warn("NOTE: There were a few undecoded keys")
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
