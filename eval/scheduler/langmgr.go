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
	"sync/atomic"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/tasks"
)

type LanguageManager struct {
	scheduler eval.BoxScheduler
	logger    *slog.Logger

	// supported is swapped atomically so lookups stay lock-free across resync.
	supported atomic.Pointer[map[string]language.GraderLang]

	versionsMu       sync.RWMutex
	languageVersions map[string]string

	// refetch is set in remote mode: it pulls the grader's supported set +
	// versions over RPC. nil in local mode (versions come from the scheduler).
	refetch func(ctx context.Context) (map[string]language.GraderLang, map[string]string, error)
}

func (mgr *LanguageManager) langs() map[string]language.GraderLang {
	if p := mgr.supported.Load(); p != nil {
		return *p
	}
	return nil
}

func (mgr *LanguageManager) getLangVersions(ctx context.Context) map[string]string {
	mgr.versionsMu.Lock()
	defer mgr.versionsMu.Unlock()
	mgr.languageVersions = make(map[string]string)
	for name, lang := range mgr.langs() {
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
	return maps.Clone(mgr.languageVersions)
}

func (mgr *LanguageManager) Language(name string) language.GraderLang {
	lang, ok := mgr.langs()[name]
	if !ok {
		return nil
	}
	return lang
}

func (mgr *LanguageManager) Languages() map[string]language.GraderLang {
	return mgr.langs()
}

func (mgr *LanguageManager) LanguageVersions(ctx context.Context) map[string]string {
	mgr.versionsMu.RLock()
	cached := mgr.languageVersions
	mgr.versionsMu.RUnlock()
	if cached == nil {
		return mgr.getLangVersions(ctx)
	}
	return maps.Clone(cached)
}

// Resync refreshes the language inventory. In remote mode it re-pulls the
// grader's supported set + versions over RPC and replaces the cache; in local
// mode it recomputes versions via the scheduler.
func (mgr *LanguageManager) Resync(ctx context.Context) error {
	if mgr.refetch == nil {
		mgr.getLangVersions(ctx)
		return nil
	}
	langs, versions, err := mgr.refetch(ctx)
	if err != nil {
		return err
	}
	mgr.supported.Store(&langs)
	mgr.versionsMu.Lock()
	mgr.languageVersions = versions
	mgr.versionsMu.Unlock()
	return nil
}

// NewRemoteLanguageManager builds a platform-side LanguageManager whose inventory
// is pulled from a remote grader. It rebuilds full language behavior from the
// binary's compiled-in language.Langs, filtered to the grader's supported names.
func NewRemoteLanguageManager(ctx context.Context, client *GraderClient, logger *slog.Logger) (eval.LanguageManager, error) {
	mgr := &LanguageManager{
		logger: logger,
		refetch: func(ctx context.Context) (map[string]language.GraderLang, map[string]string, error) {
			versions, err := client.languageVersions(ctx)
			if err != nil {
				return nil, nil, err
			}
			langs := make(map[string]language.GraderLang, len(versions))
			for name := range versions {
				if l, ok := language.Langs[name]; ok {
					langs[name] = l.GraderLang()
				} else {
					logger.WarnContext(ctx, "Grader reports a language this platform build does not know", slog.String("lang", name))
				}
			}
			return langs, versions, nil
		},
	}
	if err := mgr.Resync(ctx); err != nil {
		return nil, err
	}
	return mgr, nil
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
		for _, lang := range mgr.langs() {
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
	mgr := &LanguageManager{
		scheduler: scheduler,
		logger:    logger,
	}
	supported := supportedLanguages(ctx)
	mgr.supported.Store(&supported)
	return mgr
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
