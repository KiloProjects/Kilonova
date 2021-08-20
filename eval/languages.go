package eval

import "path"

func GetLangByFilename(filename string) string {
	fileExt := path.Ext(filename)
	if fileExt == "" {
		return ""
	}
	for k, v := range Langs {
		for _, ext := range v.Extensions {
			if ext == fileExt {
				return k
			}
		}
	}
	return ""
}

var Langs = map[string]Language{
	"c": {
		Extensions:    []string{".c"},
		Compiled:      true,
		PrintableName: "C",

		CompileCommand: []string{"gcc", "-std=c11", "-O2", "-s", "-static", "/box/main.c", "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.c",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx"},
		Compiled:      true,
		PrintableName: "C++",

		CompileCommand: []string{"g++", "-std=c++11", "-O2", "-s", "-static", "/box/main.cpp", "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"golang": {
		Extensions:    []string{".go"},
		Compiled:      true,
		PrintableName: "Go",

		CompileCommand: []string{"go", "build", "/main.go"},
		RunCommand:     []string{"/main"},
		SourceName:     "/main.go",
		CompiledName:   "/main",

		BuildEnv:  map[string]string{"GOCACHE": "/go/cache", "GOPATH": "/go"},
		CommonEnv: map[string]string{"GOMAXPROCS": "1"},
	},
	"haskell": {
		Disabled:      true, // For now
		Extensions:    []string{".hs", ".lhs"},
		Compiled:      true,
		PrintableName: "Haskell",

		CompileCommand: []string{"ghc", "-o", "/box/output", "/box/main.hs"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.hs",
		CompiledName:   "/box/output",
	},
	"java": {
		Disabled:      true, // For now
		Extensions:    []string{".java"},
		Compiled:      true,
		PrintableName: "Java",

		CompileCommand: []string{"javac", "/Main.java"},
		RunCommand:     []string{"java", "Main"},
		SourceName:     "/Main.java",
		CompiledName:   "/Main.class",

		Mounts: []Directory{{In: "/etc"}},
	},
	"python3": {
		Extensions:    []string{".py", ".py3"},
		Compiled:      false,
		PrintableName: "Python 3",

		RunCommand:   []string{"python3", "/box/main.py"},
		SourceName:   "/box/main.py",
		CompiledName: "/box/main.py",
	},
}

// Language is the data available for a language
type Language struct {
	Disabled bool

	// Useful to categorize by file upload
	Extensions []string
	Compiled   bool

	PrintableName string

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

// Directory represents a directory rule
type Directory struct {
	In      string `toml:"in"`
	Out     string `toml:"out"`
	Opts    string `toml:"opts"`
	Removes bool   `toml:"removes"`
}
