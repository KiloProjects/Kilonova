package language

import (
	"path"

	"github.com/KiloProjects/kilonova/domain/config"
)

func Uv() GraderLang {
	return &uv{}
}

type uv struct {
}

func (u uv) InternalName() string {
	return "uv"
}

func (u uv) PrintableName() string {
	return "Python 3 (Uv)"
}

func (u uv) Extensions() []string {
	return []string{".py", ".toml", ".lock"}
}

func (u uv) DefaultFilename() string {
	return "/box/main.py"
}

func (u uv) MOSSName() string {
	return "python"
}

func (u uv) Compiled() bool {
	// TODO: Uv sync does not work due to permission issues
	// // For uv sync
	// return true
	return false
}

func (u uv) CompileCommand(_ []string) []string {
	return []string{"uv", "sync", "--locked"}
}

func (u uv) RunCommand(files []string, _ int) []string {
	if len(files) == 0 {
		return []string{"uv", "run", "-q", "main.py"}
	}
	return []string{"uv", "run", "-q", files[0]}
}

func (u uv) SourceName(userFilename string) string {
	if userFilename == "" {
		return "/box/main.py"
	}
	return "/box/" + userFilename
}

func (u uv) CompiledName(userFilename string) string {
	if userFilename == "" {
		return "/box/main.py"
	}
	return "/box/" + userFilename
}

func (u uv) ExecuteName(userFilename string) string {
	if userFilename == "" {
		return "/box/main.py"
	}
	return "/box/" + userFilename
}

func (u uv) VersionCommand() []string {
	return []string{"--version"}
}

func (u uv) ParseVersion(version []byte) string {
	return string(version)
}

func (u uv) BuildEnv() map[string]string {
	return map[string]string{
		"XDG_CACHE_HOME":  "/mnt/uv/cache",
		"XDG_DATA_HOME":   "/mnt/uv/data",
		"XDG_CONFIG_HOME": "/mnt/uv/config",
		"XDG_CONFIG_DIRS": "/mnt/uv/sconfig",
		"XDG_BIN_HOME":    "/mnt/uv/bin",
	}
}

func (u uv) RunEnv() map[string]string {
	return map[string]string{
		"XDG_CACHE_HOME":  "/mnt/uv/cache",
		"XDG_DATA_HOME":   "/mnt/uv/data",
		"XDG_CONFIG_HOME": "/mnt/uv/config",
		"XDG_CONFIG_DIRS": "/mnt/uv/sconfig",
		"XDG_BIN_HOME":    "/mnt/uv/bin",
	}
}

func (u uv) Mounts() []Directory {
	return []Directory{
		{In: "/etc"},
		{In: "/run"}, // For resolv.conf
		{In: "/mnt/uv", Out: path.Join(config.Common.DataDir, "uv"), Opts: "rw"},
		{In: "/opt"}, // MacOS resolv.conf
	}
}

func (u uv) TimeLimitMultiplier() float64 {
	return 2
}

func (u uv) MemoryLimitMultiplier() float64 {
	return 2
}

func (u uv) SimilarLanguages() []string {
	return []string{"python3"}
}
