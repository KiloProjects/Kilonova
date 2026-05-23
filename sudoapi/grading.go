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

// Language returns nil if the language was not found
func (s *BaseAPI) Language(name string) language.Lang {
	if name == "ai" {
		return language.AI
	}
	return s.grader.Language(name)
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
	var lang language.Lang
	for _, elang := range s.grader.Languages() {
		if elang.MOSSName() == mossLang && (lang == nil || lang.InternalName() < elang.InternalName()) {
			lang = elang
		}
	}
	if lang == nil {
		slog.WarnContext(ctx, "Could not find language for MOSS language", slog.String("moss_lang", mossLang))
		return nil
	}
	return lang
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
			name = lang.PrintableName()
			cmds := lang.CompileCommand([]string{lang.DefaultFilename()})
			if !lang.Compiled() {
				cmds = lang.RunCommand([]string{lang.DefaultFilename()}, 512*1024)
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
		langs[lang.InternalName()] = lang.PrintableName()
	}
	return langs
}
