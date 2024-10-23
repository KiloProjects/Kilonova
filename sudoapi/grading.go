package sudoapi

import (
	"cmp"
	"context"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
)

type Language struct {
	InternalName  string `json:"internal_name"`
	PrintableName string `json:"printable_name"`

	lang *eval.Language
}

func (s *BaseAPI) evalLangToInternal(lang *eval.Language) *Language {
	if lang == nil {
		return nil
	}
	return &Language{
		InternalName:  lang.InternalName,
		PrintableName: lang.PrintableName,

		lang: lang,
	}
}

// Language returns nil if the language was not found
func (s *BaseAPI) Language(name string) *Language {
	return s.evalLangToInternal(s.grader.Language(name))
}

// TODO: Just use the one from grader.LanguageFromFilename maybe? Reduce code duplication
func (s *BaseAPI) LanguageFromFilename(filename string) string {
	fileExt := path.Ext(filename)
	if fileExt == "" {
		return ""
	}
	// bestLang heuristic to match .cpp to cpp17
	if fileExt == ".cpp" {
		return "cpp17"
	}
	bestLang := ""
	for k := range s.EnabledLanguages() {
		for _, ext := range s.Language(k).Extensions() {
			if ext == fileExt && (bestLang == "" || k < bestLang) {
				bestLang = k
			}
		}
	}
	return bestLang
}

func (s *BaseAPI) LanguageFromMOSS(mossLang string) *Language {
	var lang *eval.Language
	for _, elang := range s.grader.Languages() {
		if elang.MOSSName == mossLang && (lang == nil || lang.InternalName < elang.InternalName) {
			lang = elang
		}
	}
	if lang == nil {
		slog.Warn("Could not find language for MOSS language", slog.String("moss_lang", mossLang))
		return nil
	}
	return s.evalLangToInternal(lang)
}

type GraderLanguage struct {
	Name    string
	Version string
	Command string
}

// TODO: Refactor
func (s *BaseAPI) GraderLanguages(ctx context.Context) []*GraderLanguage {
	versions := s.grader.LanguageVersions(ctx)
	langs := make([]*GraderLanguage, 0, len(versions))
	for name := range s.EnabledLanguages() {
		if _, ok := versions[name]; !ok {
			versions[name] = "?"
		}
	}
	for langName, version := range versions {
		name, cmd := langName, "-"

		if lang := s.grader.Language(langName); lang != nil {
			name = lang.PrintableName
			cmds := slices.Clone(lang.CompileCommand)
			if !lang.Compiled {
				cmds = slices.Clone(lang.RunCommand)
			}
			for i := range cmds {
				if cmds[i] == eval.MagicReplace {
					cmds[i] = lang.SourceName
				}
				cmds[i] = strings.TrimPrefix(cmds[i], "/box/")
			}
			if len(cmds) > 0 {
				cmds[0] = path.Base(cmds[0])
				cmd = strings.Join(cmds, " ")
			}
		}
		langs = append(langs, &GraderLanguage{
			Name:    name,
			Version: version,
			Command: cmd,
		})
	}
	slices.SortFunc(langs, func(a, b *GraderLanguage) int { return cmp.Compare(a.Name, b.Name) })

	return langs
}

func (s *BaseAPI) EnabledLanguages() map[string]string {
	langs := make(map[string]string)
	for _, lang := range s.grader.Languages() {
		langs[lang.InternalName] = lang.PrintableName
	}
	return langs
}

// The extension when used for saving (unique per language)
func (l *Language) Extension() string {
	if l == nil {
		return ".txt"
	}
	if l.lang == nil {
		slog.Warn("Language created outside sudoapi tried to get extension")
		return ".err"
	}
	return l.lang.Extensions[len(l.lang.Extensions)-1]
}

// List of all valid extensions
func (l *Language) Extensions() []string {
	if l == nil {
		return nil
	}
	if l.lang == nil {
		slog.Warn("Language created outside sudoapi tried to get extensions")
		return nil
	}
	return l.lang.Extensions
}

func (l *Language) MOSSName() string {
	if l == nil {
		return "ascii"
	}
	if l.lang == nil {
		slog.Warn("Language created outside sudoapi tried to get MOSS name")
		return "ascii"
	}
	return l.lang.MOSSName
}

func (l *Language) SimilarLangs() []string {
	if l == nil {
		return nil
	}
	if l.lang == nil {
		slog.Warn("Language created outside sudoapi tried to get similar languages")
		return nil
	}
	return l.lang.SimilarLangs
}
