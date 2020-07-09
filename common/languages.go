package common

import "github.com/KiloProjects/Kilonova/grader/box"

// The data in this file should be loosely based on https://github.com/bogdan2412/infoarena/blob/master/eval/utilities.php#L48-L133

// Language is a struct for a language
type Language struct {
	// Useful to categorize by file upload
	Extensions []string
	IsCompiled bool

	CompileCommand []string
	RunCommand     []string

	BuildEnv map[string]string
	RunEnv   map[string]string
	// CommonEnv will be added at both compile-time and runtime, and can be replaced by BuildEnv/RunEnv
	CommonEnv map[string]string

	// Mounts represents all directories to be mounted
	Mounts []box.Directory
	// SourceName
	SourceName string

	CompiledName string
}

// List of languages that are harder to containerize
// C# (Building something requires running inside a project, why can't it build just a single file?!??!?!?!)

// Languages is the map with all languages
var Languages = map[string]Language{
	"cpp": {
		Extensions:     []string{".cpp", ".c++", ".cc", ".cxx"},
		IsCompiled:     true,
		CompileCommand: []string{"/usr/bin/g++", "-std=c++11", "-O2", "-pipe", "-static", "-s", "/box/main.cpp", "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		Mounts: []box.Directory{
			{In: "/etc"},
		},
		SourceName:   "/box/main.cpp",
		CompiledName: "/box/output",
	},
	"c": {
		Extensions:     []string{".c", ".h"},
		IsCompiled:     true,
		CompileCommand: []string{"/usr/bin/gcc", "-std=c11", "-O2", "-pipe", "-static", "-s", "/box/main.c", "-o", "/box/output"},
		RunCommand:     []string{"/box/output"},
		Mounts: []box.Directory{
			{In: "/etc"},
		},
		SourceName:   "/box/main.c",
		CompiledName: "/box/output",
	},
	"python": {
		Extensions: []string{".py", ".py3"},
		IsCompiled: false,
		Mounts:     []box.Directory{},
		RunCommand: []string{"/usr/bin/python3", "/box/main.py"},
		SourceName: "/box/main.py",
	},
	"java": {
		Extensions: []string{".java"},
		IsCompiled: true,
		Mounts: []box.Directory{
			{In: "/etc"},
		},
		CompileCommand: []string{"/usr/bin/javac", "/box/Main.java"},
		RunCommand:     []string{"/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.252.b09-1.fc32.x86_64/jre/bin/java", "Main"},
		SourceName:     "/box/Main.java",
		CompiledName:   "/box/Main.class",
	},
	"haskell": {
		Extensions:     []string{".hs", ".lhs"},
		IsCompiled:     true,
		Mounts:         []box.Directory{},
		CompileCommand: []string{"/usr/bin/ghc", "-o", "/box/output", "/box/main.hs"},
		RunCommand:     []string{"/box/output"},
		SourceName:     "/box/main.hs",
		CompiledName:   "/box/output",
	},
	"golang": {
		Extensions:     []string{".go"},
		IsCompiled:     true,
		CommonEnv:      map[string]string{"GOMAXPROCS": "1"},
		BuildEnv:       map[string]string{"GOPATH": "/box/go", "GOCACHE": "/box/go/cache"},
		CompileCommand: []string{"/usr/lib/golang/bin/go", "build", "/box/main.go"},
		RunCommand:     []string{"/box/main"},
		SourceName:     "/box/main.go",
		CompiledName:   "/box/main",
	},
}
