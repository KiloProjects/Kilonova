package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
)

var (
	configPath string
	Common     CommonConf
	Database   DBConf
	Eval       EvalConf
	Languages  map[string]Language
	Email      EmailConf
	Index      IndexConf
)

// configStruct is the glue for all configuration sections when unmarshaling
// After load, it will disperse all the data in variables
type configStruct struct {
	Common    CommonConf          `toml:"common"`
	Database  DBConf              `toml:"database"`
	Eval      EvalConf            `toml:"eval"`
	Languages map[string]Language `toml:"languages"`
	Email     EmailConf           `toml:"email"`
	Index     IndexConf           `toml:"index"`
}

type IndexConf struct {
	Lists        []int  `toml:"lists_to_show"`
	ShowProblems bool   `toml:"show_problems"`
	Description  string `toml:"description"`
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
	LogDir     string `toml:"log_dir"`
	DataDir    string `toml:"data_dir"`
	Debug      bool   `toml:"debug"`
	HostPrefix string `toml:"host_prefix"`
}

// DBConf is the data required to establish a PostgreSQL connection
type DBConf struct {
	Type string `toml:"dbtype"`
	DSN  string `toml:"dsn"`
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

	Printable string `toml:"printable"`

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
	Common = c.Common
	Database = c.Database
	Email = c.Email
	Eval = c.Eval
	Languages = c.Languages
	Index = c.Index
}

func compactify() {
	c.Common = Common
	c.Database = Database
	c.Email = Email
	c.Eval = Eval
	c.Languages = Languages
	c.Index = Index
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
		log.Println("NOTE: There were a few undecoded keys")
		spew.Dump(md.Undecoded())
	}
	if err == nil {
		spread()
	}
	return err
}
