package sudoapi

import (
	"cmp"
	"context"
	"log/slog"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova/eval"
)

type Language struct {
	InternalName  string
	PrintableName string

	lang *eval.Language
}

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

func (s *BaseAPI) evalLangToInternal(lang eval.Language) *Language {
	return &Language{
		InternalName:  lang.InternalName,
		PrintableName: lang.PrintableName,

		lang: &lang,
	}
}

// Language returns nil if the language was not found
func (s *BaseAPI) Language(name string) *Language {
	lang, ok := eval.Langs[name]
	if !ok {
		return nil
	}
	return s.evalLangToInternal(lang)
}

func (s *BaseAPI) LanguageFromMOSS(mossLang string) *Language {
	var lang eval.Language
	for _, elang := range eval.Langs {
		if elang.MOSSName == mossLang && (lang.InternalName == "" || lang.InternalName < elang.InternalName) {
			lang = elang
		}
	}
	if lang.InternalName == "" {
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
	for _, lang := range eval.Langs {
		if lang.Disabled {
			continue
		}
		if _, ok := versions[lang.InternalName]; !ok {
			versions[lang.InternalName] = "?"
		}
	}
	for langName, version := range versions {
		name, cmd := langName, "-"

		if lang, ok := eval.Langs[langName]; ok {
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

var enabledLanguages = sync.OnceValue(func() map[string]string {
	langs := make(map[string]string)
	for _, lang := range eval.Langs {
		if !lang.Disabled {
			langs[lang.InternalName] = lang.PrintableName
		}
	}
	return langs
})

func (s *BaseAPI) EnabledLanguages() map[string]string {
	return enabledLanguages()
}
