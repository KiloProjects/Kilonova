package scheduler

import (
	"context"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova/eval/language"
)

// SupportedLanguages returns the map of languages available in the current environment.
// It checks that each language's compiler/interpreter binary exists in $PATH.
// This is exported so it can be reused by the remote client.
func SupportedLanguages(ctx context.Context) map[string]language.GraderLang {
	return supportedLanguages(ctx)
}

// LanguageFromFilename picks the best language for a given filename from langs.
// The logic mirrors BoxManager.LanguageFromFilename and is exported for use by
// the remote client, which has its own local language map.
func LanguageFromFilename(langs map[string]language.GraderLang, filename string) language.GraderLang {
	fileExt := path.Ext(filename)
	if fileExt == "" {
		return nil
	}
	// Heuristic: prefer cpp17 for .cpp files
	if fileExt == ".cpp" {
		if x, ok := langs["cpp17"]; ok {
			return x
		}
		best := ""
		for name := range langs {
			if strings.HasPrefix(name, "cpp") && (best == "" || name < best) {
				best = name
			}
		}
		return langs[best]
	}
	bestLang := ""
	for k, v := range langs {
		for _, ext := range v.Extensions() {
			if ext == fileExt && (bestLang == "" || k < bestLang) {
				bestLang = k
			}
		}
	}
	return langs[bestLang]
}
