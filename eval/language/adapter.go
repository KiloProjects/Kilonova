package language

import (
	"context"
	"log/slog"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
)

var _ Lang = (*legacyLanguageAdapter)(nil)
var _ GraderLang = (*legacyLanguageAdapter)(nil)

type legacyLanguageAdapter struct {
	lang *legacyLanguage
}

func (l legacyLanguageAdapter) InternalName() string {
	if l.lang == nil || l.lang.InternalName == "" {
		slog.WarnContext(context.Background(), "Language has no internal name", slog.Any("lang", l.lang))
		return "err"
	}
	return l.lang.InternalName
}

func (l legacyLanguageAdapter) PrintableName() string {
	if l.lang == nil || l.lang.PrintableName == "" {
		slog.WarnContext(context.Background(), "Language has no internal name", slog.Any("lang", l.lang))
		return "N/A (error)"
	}
	return l.lang.PrintableName
}

func (l legacyLanguageAdapter) DefaultFilename() string {
	if l.lang == nil {
		slog.WarnContext(context.Background(), "Language not initialized", slog.Any("lang", l.lang))
		return "err.txt"
	}
	return path.Base(l.lang.sourceName)
}

func (l legacyLanguageAdapter) Extensions() []string {
	if l.lang == nil {
		return nil
	}
	return slices.Clone(l.lang.Extensions)
}

func (l legacyLanguageAdapter) MOSSName() string {
	if l.lang == nil {
		return "ascii"
	}
	return l.lang.MOSSName
}

func (l legacyLanguageAdapter) Compiled() bool {
	if l.lang == nil {
		panic("Uninitialized language in grader")
	}
	return l.lang.Compiled
}

func (l legacyLanguageAdapter) CompileCommand(files []string) []string {
	return makeGoodSandboxCommand(l.lang.CompileCommand, files)
}

func (l legacyLanguageAdapter) RunCommand(files []string, memoryLimit int) []string {
	cmd := makeGoodSandboxCommand(l.lang.RunCommand, files)
	for i := range cmd {
		if strings.Contains(cmd[i], memoryReplace) {
			cmd[i] = strings.ReplaceAll(cmd[i], memoryReplace, strconv.Itoa(memoryLimit))
		}
	}
	return cmd
}

func (l legacyLanguageAdapter) SourceName(userFilename string) string {
	return l.lang.SourceName(userFilename)
}

func (l legacyLanguageAdapter) CompiledName(userFilename string) string {
	return l.lang.CompiledName(userFilename)
}

func (l legacyLanguageAdapter) ExecuteName(userFilename string) string {
	return l.lang.ExecuteName(userFilename)
}

func (l legacyLanguageAdapter) VersionCommand() []string {
	return slices.Clone(l.lang.VersionCommand)
}

func (l legacyLanguageAdapter) ParseVersion(version []byte) string {
	if l.lang.VersionParser == nil {
		return string(version)
	}
	return l.lang.VersionParser(string(version))
}

func (l legacyLanguageAdapter) BuildEnv() map[string]string {
	return maps.Clone(l.lang.BuildEnv)
}

func (l legacyLanguageAdapter) RunEnv() map[string]string {
	return maps.Clone(l.lang.RunEnv)
}

func (l legacyLanguageAdapter) Mounts() []Directory {
	return slices.Clone(l.lang.Mounts)
}

func (l legacyLanguageAdapter) TimeLimitMultiplier() float64 {
	return l.lang.TimeLimitMultiplier
}

func (l legacyLanguageAdapter) MemoryLimitMultiplier() float64 {
	return l.lang.MemoryLimitMultiplier
}

func (l legacyLanguageAdapter) SimilarLanguages() []string {
	return slices.Clone(l.lang.SimilarLangs)
}

func makeGoodSandboxCommand(command []string, files []string) []string {
	for i := range command {
		if command[i] == magicReplace {
			return slices.Concat(command[:i], files, command[i+1:])
		}
	}

	slog.WarnContext(context.Background(), "Did not replace any fields in command", slog.Any("command", command))
	return slices.Clone(command)
}
