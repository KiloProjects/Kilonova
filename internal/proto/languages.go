package proto

// LANGUAGE DEFINITIONS
// The data in this file should be loosely based on https://github.com/bogdan2412/infoarena/blob/master/eval/utilities.php#L48-L133

// Directory represents a directory rule
type Directory struct {
	In      string
	Out     string
	Opts    string
	Removes bool
}

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
	Mounts []Directory
	// SourceName
	SourceName string

	CompiledName string
}

// Languages is the map with all languages
var Languages = map[string]Language{
	"cpp": {
		Extensions:     []string{".cpp", ".c++", ".cc", ".cxx"},
		IsCompiled:     true,
		CompileCommand: []string{"/usr/bin/g++", "-std=c++11", "-O2", "-s", "-static", "main.cpp", "-o", "output"},
		RunCommand:     []string{"/box/output"},
		Mounts: []Directory{
			{In: "/etc"},
		},
		SourceName:   "/box/main.cpp",
		CompiledName: "/box/output",
	},
	"c": {
		Extensions:     []string{".c", ".h"},
		IsCompiled:     true,
		CompileCommand: []string{"/usr/bin/gcc", "-std=c11", "-O2", "-s", "-static", "main.c", "-o", "/output"},
		RunCommand:     []string{"/output"},
		Mounts: []Directory{
			{In: "/etc"},
		},
		SourceName:   "/main.c",
		CompiledName: "/output",
	},
	"python": {
		Extensions:   []string{".py", ".py3"},
		IsCompiled:   false,
		Mounts:       []Directory{},
		RunCommand:   []string{"/usr/bin/python3", "/main.py"},
		SourceName:   "/main.py",
		CompiledName: "/main.py",
	}, /*
		"java": {
			Extensions: []string{".java"},
			IsCompiled: true,
			Mounts: []Directory{
				{In: "/etc"},
			},
			CompileCommand: []string{"/usr/bin/javac", "/Main.java"},
			RunCommand:     []string{"/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.252.b09-1.fc32.x86_64/jre/bin/java", "Main"},
			SourceName:     "/Main.java",
			CompiledName:   "/Main.class",
		},
		"haskell": {
			Extensions:     []string{".hs", ".lhs"},
			IsCompiled:     true,
			Mounts:         []Directory{},
			CompileCommand: []string{"/usr/bin/ghc", "-o", "/output", "/main.hs"},
			RunCommand:     []string{"/output"},
			SourceName:     "/main.hs",
			CompiledName:   "/output",
		},
		"golang": {
			Extensions:     []string{".go"},
			IsCompiled:     true,
			CommonEnv:      map[string]string{"GOMAXPROCS": "1"},
			BuildEnv:       map[string]string{"GOPATH": "/go", "GOCACHE": "/go/cache"},
			CompileCommand: []string{"/usr/lib/golang/bin/go", "build", "/main.go"},
			RunCommand:     []string{"/main"},
			SourceName:     "/main.go",
			CompiledName:   "/main",
		}, */
}
