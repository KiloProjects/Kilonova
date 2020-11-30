package config

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

// ConfigStruct is the glue for all configuration sections
type ConfigStruct struct {
	Common   Common   `toml:"common"`
	Database Database `toml:"database"`
}

// Common is the data required for all services
type Common struct {
	LogDir  string `toml:"log_dir"`
	DataDir string `toml:"data_dir"`
	Debug   bool   `toml:"debug"`
}

// Database is the data required to establish a PostgreSQL connection
type Database struct {
	DBname  string `toml:"dbname"`
	Host    string `toml:"host"`
	SSLmode string `toml:"sslmode"`
	User    string `toml:"user"`
}

// String returns a DSN with all information from the struct
func (d Database) String() string {
	return fmt.Sprintf("sslmode=%s host=%s user=%s dbname=%s", d.SSLmode, d.Host, d.User, d.DBname)
}

// C represents the loaded config
var C ConfigStruct

func Load(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, &C)
}
