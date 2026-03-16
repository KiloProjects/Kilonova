package language

// Lang provides the base metadata for an evaluation language.
// It is safe to be used anywhere, not just the grader.
type Lang interface {
	// InternalName is the unique name for the given language
	InternalName() string
	// PrintableName is the human-readable name of the language
	PrintableName() string

	// Extensions lists all matching extensions for a language.
	// The last extension is unique to the language,
	// the rest can overlap with others.
	Extensions() []string

	// DefaultFilename is the sample name to use for files,
	// when no other info is provided.
	DefaultFilename() string

	// MOSSName is the language ID to be used for MOSS submissions
	// Multiple distinct languages can have the same MOSS name
	MOSSName() string
}

type GraderLang interface {
	Lang

	Compiled() bool
	CompileCommand(files []string) []string
	RunCommand(files []string, memoryLimit int) []string

	SourceName(userFilename string) string
	CompiledName(userFilename string) string
	ExecuteName(userFilename string) string

	VersionCommand() []string
	ParseVersion(version []byte) string

	// BuildEnv returns the environment variables to be set when compiling
	// These are safe to modify
	BuildEnv() map[string]string
	RunEnv() map[string]string
	Mounts() []Directory

	TimeLimitMultiplier() float64
	MemoryLimitMultiplier() float64

	// SimilarLanguages returns the list of recognized similar languages to this one
	SimilarLanguages() []string
}

func Extension(lang Lang) string {
	if lang == nil || len(lang.Extensions()) == 0 {
		return ".txt"
	}
	return lang.Extensions()[len(lang.Extensions())-1]
}

func FirstExtension(lang Lang) string {
	if lang == nil || len(lang.Extensions()) == 0 {
		return ".txt"
	}
	return lang.Extensions()[0]
}
