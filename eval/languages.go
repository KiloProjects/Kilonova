package eval

import "path"

const MAGIC_REPLACE = "<REPLACE>"

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
		InternalName:  "c",

		CompileCommand: []string{"gcc", "-fuse-ld=mold", "-std=c11", "-O2", "-s", "-static", "-DONLINE_JUDGE", MAGIC_REPLACE, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.c",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx"},
		Compiled:      true,
		PrintableName: "C++11",
		InternalName:  "cpp",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++11", "-O2", "-s", "-static", "-DONLINE_JUDGE", MAGIC_REPLACE, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp14": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx"},
		Compiled:      true,
		PrintableName: "C++14",
		InternalName:  "cpp14",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++14", "-O2", "-s", "-static", "-DONLINE_JUDGE", MAGIC_REPLACE, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp17": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx"},
		Compiled:      true,
		PrintableName: "C++17",
		InternalName:  "cpp17",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++17", "-O2", "-s", "-static", "-DONLINE_JUDGE", MAGIC_REPLACE, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp20": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx"},
		Compiled:      true,
		PrintableName: "C++20",
		InternalName:  "cpp20",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++20", "-O2", "-s", "-static", "-DONLINE_JUDGE", MAGIC_REPLACE, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",

		Mounts: []Directory{{In: "/etc"}},
	},
	"golang": {
		Extensions:    []string{".go"},
		Compiled:      true,
		PrintableName: "Go",
		InternalName:  "golang",

		CompileCommand: []string{"go", "build", "MAGIC_REPLACE"},
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
		InternalName:  "haskell",

		CompileCommand: []string{"ghc", "-o", "/box/output", MAGIC_REPLACE},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.hs",
		CompiledName:   "/box/output",
	},
	"java": {
		Disabled:      true, // For now
		Extensions:    []string{".java"},
		Compiled:      true,
		PrintableName: "Java",
		InternalName:  "java",

		CompileCommand: []string{"javac", MAGIC_REPLACE},
		RunCommand:     []string{"java", "Main"},
		SourceName:     "/Main.java",
		CompiledName:   "/Main.class",

		Mounts: []Directory{{In: "/etc"}},
	},
	"python3": {
		Extensions:    []string{".py", ".py3"},
		Compiled:      false,
		PrintableName: "Python 3",
		InternalName:  "python3",

		RunCommand:   []string{"python3", "/box/main.py"},
		SourceName:   "/box/main.py",
		CompiledName: "/box/main.py",
	},
	"outputOnly": {
		Extensions:    []string{".output_only"},
		Compiled:      false,
		PrintableName: "Output Only",
		InternalName:  "outputOnly",

		RunCommand:   []string{"cat", "/box/output"},
		SourceName:   "/box/output_src",
		CompiledName: "/box/output",
	},
}

// Language is the data available for a language
type Language struct {
	Disabled bool

	// Useful to categorize by file upload
	Extensions []string
	Compiled   bool

	PrintableName string
	InternalName  string

	CompileCommand []string `toml:"compile_command"`
	RunCommand     []string `toml:"run_command"`

	BuildEnv map[string]string `toml:"build_env"`
	RunEnv   map[string]string `toml:"run_env"`
	// CommonEnv will be added at both compile-time and runtime, and can be replaced by BuildEnv/RunEnv
	CommonEnv map[string]string `toml:"common_env"`

	// Mounts represents all directories to be mounted
	Mounts     []Directory `toml:"mounts"`
	SourceName string      `toml:"source_name"`

	CompiledName string `toml:"compiled_name"`
}

// Directory represents a directory rule
type Directory struct {
	In      string `toml:"in"`
	Out     string `toml:"out"`
	Opts    string `toml:"opts"`
	Removes bool   `toml:"removes"`
}
