package eval

import (
	"strings"
)

const (
	MagicReplace  = "<REPLACE>"
	MemoryReplace = "<MEMORY>"
)

// NOTE: Last extension MUST be unique (for proper detection of submissions in problem archives)
// TODO: remove Disabled
var Langs = map[string]Language{
	"c": {
		Extensions:    []string{".c"},
		Compiled:      true,
		PrintableName: "C",
		InternalName:  "c",
		MOSSName:      "cc", // Treat C as C++. Not necessarily correct but might help

		CompileCommand: []string{"gcc", "-fuse-ld=mold", "-std=gnu11", "-O2", "-lm", "-s", "-static", "-DKNOVA", "-DONLINE_JUDGE", MagicReplace, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.c",
		CompiledName:   "/box/output",
		SimilarLangs:   []string{"c", "cpp", "cpp11", "cpp14", "cpp17", "cpp20"},

		VersionCommand: []string{"gcc", "--version"},
		VersionParser:  getFirstLine,

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp11": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx", ".cpp11"},
		Compiled:      true,
		PrintableName: "C++11",
		InternalName:  "cpp11",
		MOSSName:      "cc",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++11", "-O2", "-s", "-static", "-DKNOVA", "-DONLINE_JUDGE", MagicReplace, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",
		SimilarLangs:   []string{"c", "cpp", "cpp11", "cpp14", "cpp17", "cpp20"},

		VersionCommand: []string{"g++", "--version"},
		VersionParser:  getFirstLine,

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp14": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx", ".cpp14"},
		Compiled:      true,
		PrintableName: "C++14",
		InternalName:  "cpp14",
		MOSSName:      "cc",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++14", "-O2", "-s", "-static", "-DKNOVA", "-DONLINE_JUDGE", MagicReplace, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",
		SimilarLangs:   []string{"c", "cpp", "cpp11", "cpp14", "cpp17", "cpp20"},

		VersionCommand: []string{"g++", "--version"},
		VersionParser:  getFirstLine,

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp17": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx", ".cpp17"},
		Compiled:      true,
		PrintableName: "C++17",
		InternalName:  "cpp17",
		MOSSName:      "cc",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++17", "-O2", "-s", "-static", "-DKNOVA", "-DONLINE_JUDGE", MagicReplace, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",
		SimilarLangs:   []string{"c", "cpp", "cpp11", "cpp14", "cpp17", "cpp20"},

		VersionCommand: []string{"g++", "--version"},
		VersionParser:  getFirstLine,

		Mounts: []Directory{{In: "/etc"}},
	},
	"cpp20": {
		Extensions:    []string{".cpp", ".c++", ".cc", ".cxx", ".cpp20"},
		Compiled:      true,
		PrintableName: "C++20",
		InternalName:  "cpp20",
		MOSSName:      "cc",

		CompileCommand: []string{"g++", "-fuse-ld=mold", "-std=c++20", "-O2", "-s", "-static", "-DKNOVA", "-DONLINE_JUDGE", MagicReplace, "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.cpp",
		CompiledName:   "/box/output",
		SimilarLangs:   []string{"c", "cpp", "cpp11", "cpp14", "cpp17", "cpp20"},

		VersionCommand: []string{"g++", "--version"},
		VersionParser:  getFirstLine,

		Mounts: []Directory{{In: "/etc"}},
	},
	"pascal": {
		// NOTE: fpc compiler is in the `fp-compiler` package on Ubuntu.
		// The `fpc` package would also install the IDE, which depends on x11 and other unnecessary fluff

		Extensions:    []string{".pas"},
		Compiled:      true,
		PrintableName: "Pascal",
		InternalName:  "pascal",
		MOSSName:      "pascal",

		CompileCommand: []string{"fpc", "-O2", "-XSst", "-Mobjfpc", "-vw", "-dKNOVA", "-dONLINE_JUDGE", MagicReplace, "-o/box/output"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.pas",
		CompiledName:   "/box/output",

		VersionCommand: []string{"fpc", "-iWDSOSP"},
		VersionParser:  nil,

		Mounts: []Directory{{In: "/etc"}},
	},
	"golang": {
		// Disabled:      true, // Doesn't work
		Extensions:    []string{".go"},
		Compiled:      true,
		PrintableName: "Go",
		InternalName:  "golang",
		MOSSName:      "ascii", // MOSS doesn't support go

		CompileCommand: []string{"/usr/bin/go", "build", MagicReplace},
		RunCommand:     []string{"/box/main"},
		SourceName:     "/box/main.go",
		CompiledName:   "/box/main",

		VersionCommand: []string{"/usr/bin/go", "version"},
		VersionParser:  nil,

		BuildEnv: map[string]string{"GOMAXPROCS": "1", "CGO_ENABLED": "0", "GOCACHE": "/go/cache", "GOPATH": "/box", "GO111MODULE": "off"},
		RunEnv:   map[string]string{"GOMAXPROCS": "1"},

		// TODO: Find way to nicely mount compilation cache so it doesn't take 10 seconds to compile stdlib.
		Mounts: []Directory{{In: "/go", Opts: "tmp", Verbatim: true}},
	},
	"haskell": {
		Disabled:      true, // For now
		Extensions:    []string{".hs", ".lhs"},
		Compiled:      true,
		PrintableName: "Haskell",
		InternalName:  "haskell",
		MOSSName:      "haskell",

		CompileCommand: []string{"ghc", "-o", "/box/output", MagicReplace},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.hs",
		CompiledName:   "/box/output",

		VersionCommand: []string{"ghc", "--numeric-version"},
		VersionParser:  nil,
	},
	"java": {
		Disabled:      true, // For now
		Extensions:    []string{".java"},
		Compiled:      true,
		PrintableName: "Java",
		InternalName:  "java",
		MOSSName:      "java",

		CompileCommand: []string{"javac", MagicReplace},
		RunCommand:     []string{"java", "Main"},
		SourceName:     "/Main.java",
		CompiledName:   "/Main.class",

		VersionCommand: []string{"javac", "--version"},
		VersionParser:  nil,

		Mounts: []Directory{{In: "/etc"}},
	},
	"kotlin": {
		Extensions:    []string{".kt"},
		Compiled:      true,
		PrintableName: "Kotlin",
		InternalName:  "kotlin",
		MOSSName:      "ascii", // MOSS doesn't support kotlin

		CompileCommand: []string{"kotlinc", MagicReplace, "-include-runtime", "-d", "output.jar"},
		RunCommand:     []string{"java", "-Xmx" + MemoryReplace + "K", "-DKNOVA", "-DONLINE_JUDGE", "-jar", "/box/output.jar"},
		SourceName:     "/box/main.kt",
		CompiledName:   "/box/output.jar",

		VersionCommand: []string{"kotlinc", "-version"},
		VersionParser:  func(s string) string { return strings.TrimPrefix(s, "info:") },

		Mounts: []Directory{{In: "/etc"}},
	},
	"python3": {
		Extensions:    []string{".py", ".py3"},
		Compiled:      false,
		PrintableName: "Python 3",
		InternalName:  "python3",
		MOSSName:      "python",

		RunCommand:   []string{"python3", "/box/main.py"},
		SourceName:   "/box/main.py",
		CompiledName: "/box/main.py",

		VersionCommand: []string{"python3", "--version"},
		VersionParser:  nil,
	},
	"nodejs": {
		// Disabled: true, // For now

		Extensions:    []string{".js"},
		Compiled:      true,
		PrintableName: "Node.js",
		InternalName:  "nodejs",
		MOSSName:      "javascript",

		CompileCommand: []string{"node", "-c", MagicReplace},
		RunCommand:     []string{"node", "/box/index.js"},
		SourceName:     "/box/index.js",
		CompiledName:   "/box/index.js",

		VersionCommand: []string{"node", "--version"},
		VersionParser:  nil,
	},
	"php": {
		// NOTE: Requires the php-cli package
		Extensions:    []string{".php"},
		Compiled:      true,
		PrintableName: "PHP",
		InternalName:  "php",
		MOSSName:      "ascii", // MOSS doesn't support php

		CompileCommand: []string{"php", "-l", MagicReplace},
		RunCommand: []string{"php", "-n",
			"-d", "ONLINE_JUDGE=true", "-d", "KNOVA=true", "-d", "display_errors=Off", "-d", "error_reporting=0",
			"-d", "memory_limit=" + MemoryReplace + "K",
			"/box/index.php",
		},
		SourceName:   "/box/index.php",
		CompiledName: "/box/index.php",

		VersionCommand: []string{"php", "--version"},
		VersionParser:  getFirstLine,
	},
	"rust": {
		Extensions: []string{".rs"},

		Compiled:      true,
		PrintableName: "Rust",
		InternalName:  "rust",
		MOSSName:      "ascii", // MOSS doesn't support rust

		CompileCommand: []string{"rustc", "--edition", "2021", "-O", "-C", "strip=symbols",
			"--cfg", "ONLINE_JUDGE", "--cfg", "KNOVA", MagicReplace, "-o", "/box/output",
		},
		RunCommand:   []string{"/box/output"},
		SourceName:   "/box/main.rs",
		CompiledName: "/box/output",

		VersionCommand: []string{"rustc", "--version"},

		Mounts: []Directory{{In: "/etc"}},
	},
	"outputOnly": {
		Extensions:    []string{".output_only"},
		Compiled:      false,
		PrintableName: "Output Only",
		InternalName:  "outputOnly",
		MOSSName:      "ascii", // Though MOSS isn't required for output only problems

		RunCommand:   []string{"cat", "/box/output"},
		SourceName:   "/box/output_src",
		CompiledName: "/box/output",

		VersionCommand: []string{"echo", "N/A"},
		VersionParser:  nil,
	},
}

// Language is the data available for a language
type Language struct {
	Disabled bool `json:"disabled"`

	// Useful to categorize by file upload
	Extensions []string `json:"extensions"`
	Compiled   bool     `json:"compiled"`

	// SimilarLangs is used on resolution of grader files during evaluation
	// to decide which of the grader files to include for interactive problems
	SimilarLangs []string `json:"-"`

	PrintableName string `json:"printable_name"`
	InternalName  string `json:"internal_name"`

	// Reference: http://moss.stanford.edu/general/scripts/mossnet
	MOSSName string `json:"-"`

	CompileCommand []string `json:"compile_command"`
	RunCommand     []string `json:"run_command"`

	VersionCommand []string `json:"-"`
	// Function to process the output of the VersionCommand output.
	// If nil, command output will be returned as is
	VersionParser func(string) string `json:"-"`

	BuildEnv map[string]string `json:"-"`
	RunEnv   map[string]string `json:"-"`

	// Mounts represents all directories to be mounted
	Mounts     []Directory `json:"-"`
	SourceName string      `json:"-"`

	CompiledName string `json:"compiled_name"`
}

// Directory represents a directory rule
type Directory struct {
	In      string `toml:"in"`
	Out     string `toml:"out"`
	Opts    string `toml:"opts"`
	Removes bool   `toml:"removes"`

	// Verbatim doesn't set Out to In implicitly if it isn't set
	Verbatim bool `toml:"verbatim"`
}

func getFirstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	return s
}
