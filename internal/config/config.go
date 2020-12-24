package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/go-redis/redis/v8"
)

// ConfigStruct is the glue for all configuration sections
type ConfigStruct struct {
	Cache    Cache    `toml:"cache"`
	Common   Common   `toml:"common"`
	Database Database `toml:"database"`
	Eval     Eval     `toml:"eval"`
}

// Cache is the data required for the redis part (when I eventually make it)
type Cache struct {
	Host     string `toml:"host"`
	Password string `toml:"password"`
	DB       int    `toml:"DB"`
}

func (c Cache) GenOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Host,
		Password: c.Password,
		DB:       c.DB,
	}
}

// Eval is the data required for the eval service
type Eval struct {
	IsolatePath string `toml:"isolatePath"`
	CompilePath string `toml:"compilePath"`
	Address     string `toml:"address"`
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
	_, err := toml.DecodeFile(path, &C)
	return err
}
