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

//type GraderLang interface{
//	Lang
//
//	Compile(files []string, memory int)
//}

func Extension(lang Lang) string {
	if lang == nil || len(lang.Extensions()) == 0 {
		return ".txt"
	}
	return lang.Extensions()[len(lang.Extensions())-1]
}
