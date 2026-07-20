package scheduler

import (
	"context"
	"log/slog"
	"maps"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/tasks"
)

type LanguageManager struct {
	scheduler eval.BoxScheduler
	logger    *slog.Logger

	languageVersionsMu sync.RWMutex
	languageVersions   map[string]string
	supportedLanguages map[string]language.GraderLang
}

func (mgr *LanguageManager) getLangVersions(ctx context.Context) map[string]string {
	mgr.languageVersionsMu.Lock()
	defer mgr.languageVersionsMu.Unlock()
	mgr.languageVersions = make(map[string]string)
	for name, lang := range mgr.supportedLanguages {
		// disabled languages are not added to supportedLanguages
		// if lang.Disabled {
		//	continue
		// }

		ver, err := tasks.VersionTask(ctx, mgr.scheduler, lang)
		if err != nil {
			slog.WarnContext(ctx, "Could not get version for language", slog.String("lang", name))
			ver = "ERR"
		} else {
			ver = strings.TrimSpace(ver)
			mgr.logger.InfoContext(ctx, "Got version for language", slog.String("lang", name), slog.String("version", ver))
		}
		mgr.languageVersions[name] = ver
	}
	return mgr.languageVersions
}

func (mgr *LanguageManager) Language(name string) language.GraderLang {
	lang, ok := mgr.supportedLanguages[name]
	if !ok {
		return nil
	}
	return lang
}

func (mgr *LanguageManager) Languages() map[string]language.GraderLang {
	// TODO: maybe a maps.Clone()?
	return mgr.supportedLanguages
}

func (mgr *LanguageManager) LanguageVersions(ctx context.Context) map[string]string {
	if mgr.languageVersions == nil {
		return mgr.getLangVersions(ctx)
	}
	mgr.languageVersionsMu.RLock()
	defer mgr.languageVersionsMu.RUnlock()
	return maps.Clone(mgr.languageVersions)
}

// TODO: Improve
func (mgr *LanguageManager) LanguageFromFilename(filename string) language.GraderLang {
	fileExt := path.Ext(filename)
	if fileExt == "" {
		return nil
	}
	// bestLang heuristic to match .cpp to cpp17
	if fileExt == ".cpp" {
		if x := mgr.Language("cpp17"); x != nil {
			return x
		}
		// Otherwise fall back to earliest cpp version
		best := ""
		for _, lang := range mgr.supportedLanguages {
			if strings.HasPrefix(lang.InternalName(), ".cpp") && (best == "" || lang.InternalName() < best) {
				best = lang.InternalName()
			}
		}
		return mgr.Language(best)
	}
	bestLang := ""
	for k, v := range mgr.Languages() {
		for _, ext := range v.Extensions() {
			if ext == fileExt && (bestLang == "" || k < bestLang) {
				bestLang = k
			}
		}
	}
	return mgr.Language(bestLang)
}

func NewLanguageManager(ctx context.Context, scheduler eval.BoxScheduler, logger *slog.Logger) eval.LanguageManager {
	return &LanguageManager{
		scheduler: scheduler,
		logger:    logger,

		languageVersions:   make(map[string]string),
		supportedLanguages: supportedLanguages(ctx),
	}
}

// supportedLanguages disables all languages that are *not* detected by the system in the current configuration
// It should be run at the start of the execution (and implemented more nicely tbh)
func supportedLanguages(ctx context.Context) map[string]language.GraderLang {
	langs := make(map[string]language.GraderLang)
	for k, v := range language.Langs {
		if v.Disabled() { // Skip search if already disabled
			continue
		}
		var toSearch []string
		if v.GraderLang().Compiled() {
			toSearch = v.GraderLang().CompileCommand([]string{""})
		} else {
			toSearch = v.GraderLang().RunCommand([]string{""}, 0)
		}
		if v.Lang().InternalName() == "java" {
			toSearch = []string{"javac"}
		}
		if len(toSearch) == 0 {
			slog.InfoContext(ctx, "Disabled language - empty line", slog.String("lang", k))
			continue
		}
		cmd, err := exec.LookPath(toSearch[0])
		if err != nil {
			slog.InfoContext(ctx, "Disabled language - compiler/interpreter was not found in $PATH", slog.String("lang", k))
			continue
		}
		if _, err = filepath.EvalSymlinks(cmd); err != nil {
			slog.InfoContext(ctx, "Disabled language - compiler/interpreter had a bad symlink", slog.String("lang", k))
			continue
		}

		langs[k] = v.GraderLang()
	}
	return langs
}
