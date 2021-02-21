package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-redis/redis/v8"
)

var (
	Cache     CacheConf
	Common    CommonConf
	Database  DBConf
	Eval      EvalConf
	Languages map[string]Language
	Email     EmailConf
)

// configStruct is the glue for all configuration sections when unmarshaling
// After load, it will disperse all the data in variables
type configStruct struct {
	Cache     CacheConf           `toml:"cache"`
	Common    CommonConf          `toml:"common"`
	Database  DBConf              `toml:"database"`
	Eval      EvalConf            `toml:"eval"`
	Languages map[string]Language `toml:"languages"`
	Email     EmailConf           `toml:"email"`
}

// EmailConf is the data required for the email part
type EmailConf struct {
	Host     string `toml:"host"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

// CacheConf is the data required for the redis part (when I eventually make it)
type CacheConf struct {
	Host     string `toml:"host"`
	Password string `toml:"password"`
	DB       int    `toml:"DB"`
}

func (c CacheConf) GenOptions() *redis.Options {
	return &redis.Options{
		Addr:     c.Host,
		Password: c.Password,
		DB:       c.DB,
	}
}

// EvalConf is the data required for the eval service
type EvalConf struct {
	IsolatePath string `toml:"isolatePath"`
	CompilePath string `toml:"compilePath"`
	Address     string `toml:"address"`
}

// CommonConf is the data required for all services
type CommonConf struct {
	LogDir     string `toml:"log_dir"`
	DataDir    string `toml:"data_dir"`
	Debug      bool   `toml:"debug"`
	HostPrefix string `toml:"host_prefix"`
}

// DBConf is the data required to establish a PostgreSQL connection
type DBConf struct {
	DBname  string `toml:"dbname"`
	Host    string `toml:"host"`
	SSLmode string `toml:"sslmode"`
	User    string `toml:"user"`
}

// String returns a DSN with all information from the struct
func (d DBConf) String() string {
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

	Printable string `tmol:"printable"`

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

// c represents the loaded config
var c configStruct

func spread() {
	Cache = c.Cache
	Common = c.Common
	Database = c.Database
	Email = c.Email
	Eval = c.Eval
	Languages = c.Languages
}

func compactify() {
	c.Cache = Cache
	c.Common = Common
	c.Database = Database
	c.Email = Email
	c.Eval = Eval
	c.Languages = Languages
}

func Save(path string) error {
	compactify()

	// Make the directories just in case they don't exist
	if err := os.MkdirAll(filepath.Dir(path), 0666); err != nil {
		return err
	}

	file, err := os.Create(path)
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

func Load(path string) error {
	md, err := toml.DecodeFile(path, &c)
	if len(md.Undecoded()) > 0 {
		log.Println("NOTE: There were a few undecoded keys")
		spew.Dump(md.Undecoded())
	}
	if err == nil {
		spread()
	}
	return err
}
