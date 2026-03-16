package sudoapi

import (
	"cmp"
	"context"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova/eval/language"
)

var _ language.Lang = (*sudoLanguage)(nil)

func (s *BaseAPI) evalLangToInternal(lang *language.Language) language.Lang {
	if lang == nil {
		return nil
	}
	return &sudoLanguage{
		lang: lang,
	}
}

// Language returns nil if the language was not found
func (s *BaseAPI) Language(name string) language.Lang {
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

func (s *BaseAPI) LanguageFromMOSS(ctx context.Context, mossLang string) language.Lang {
	var lang *language.Language
	for _, elang := range s.grader.Languages() {
		if elang.MOSSName == mossLang && (lang == nil || lang.InternalName < elang.InternalName) {
			lang = elang
		}
	}
	if lang == nil {
		slog.WarnContext(ctx, "Could not find language for MOSS language", slog.String("moss_lang", mossLang))
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
				if cmds[i] == language.MagicReplace {
					cmds[i] = lang.SourceName("")
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

type sudoLanguage struct {
	lang *language.Language
}

func (l sudoLanguage) InternalName() string {
	if l.lang == nil || l.lang.InternalName == "" {
		slog.WarnContext(context.Background(), "Language has no internal name", slog.Any("lang", l.lang))
		return "err"
	}
	return l.lang.InternalName
}

func (l sudoLanguage) PrintableName() string {
	if l.lang == nil || l.lang.PrintableName == "" {
		slog.WarnContext(context.Background(), "Language has no internal name", slog.Any("lang", l.lang))
		return "N/A (error)"
	}
	return l.lang.PrintableName
}

func (l sudoLanguage) DefaultFilename() string {
	if l.lang == nil {
		slog.WarnContext(context.Background(), "Language not initialized", slog.Any("lang", l.lang))
		return "err.txt"
	}
	return l.lang.DefaultFilename()
}

func (l sudoLanguage) Extensions() []string {
	if l.lang == nil {
		return nil
	}
	return l.lang.Extensions
}

func (l sudoLanguage) MOSSName() string {
	if l.lang == nil {
		return "ascii"
	}
	return l.lang.MOSSName
}
