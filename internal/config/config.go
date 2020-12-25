package config

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v8"
)

// ConfigStruct is the glue for all configuration sections
type ConfigStruct struct {
	Cache     Cache               `toml:"cache"`
	Common    Common              `toml:"common"`
	Database  Database            `toml:"database"`
	Eval      Eval                `toml:"eval"`
	Languages map[string]Language `toml:"languages"`
	Email     Email               `toml:"email"`
}

type Email struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
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

//  LANGUAGE DEFINITION STUFF --------------------

// Directory represents a directory rule
type Directory struct {
	In      string `toml:"in"`
	Out     string `toml:"out"`
	Opts    string `toml:"opts"`
	Removes bool   `toml:"removes"`
}

// Language is a struct for a language
type Language struct {
	Disabled bool `toml:"disabled"`

	// Useful to categorize by file upload
	Extensions []string `toml:"extensions"`
	IsCompiled bool     `toml:"is_compiled"`

	CompileCommand []string `toml:"compile_command"`
	RunCommand     []string `toml:"run_command"`

	BuildEnv map[string]string `toml:"build_env"`
	RunEnv   map[string]string `toml:"run_env"`
	// CommonEnv will be added at both compile-time and runtime, and can be replaced by BuildEnv/RunEnv
	CommonEnv map[string]string `toml:"common_env"`

	// Mounts represents all directories to be mounted
	Mounts []Directory `toml:"mounts"`
	// SourceName
	SourceName string `toml:"source_name"`

	CompiledName string `toml:"compiled_name"`
}

// /LANGUAGE DEFINITION STUFF --------------------

// C represents the loaded config
var C ConfigStruct

func Load(path string) error {
	md, err := toml.DecodeFile(path, &C)
	if len(md.Undecoded()) > 0 {
		log.Println("NOTE: There were a few undecoded keys")
		spew.Dump(md.Undecoded())
	}
	return err
}
