package scheduler

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
	"golang.org/x/sync/semaphore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	cmdAuditLogger *slog.Logger
	loggerOnce     sync.Once
)

type BoxFunc func(ctx context.Context, id int, mem int64, logger *slog.Logger) (eval.Sandbox, error)

var _ eval.BoxScheduler = &BoxManager{}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	numConcurrent int64
	// concSem measures the number of running Box2 requests.
	// Since a request will be able to have multiple boxes (communication type submissions), it does not reflect the number of concurrent boxes running.
	concSem   *semaphore.Weighted
	memSem    *semaphore.Weighted
	maxMemory int64

	logger *slog.Logger

	availableIDs chan int

	parentMgr *BoxManager

	boxGenerator BoxFunc

	languageVersionsMu sync.RWMutex
	languageVersions   map[string]string
	supportedLanguages map[string]*eval.Language

	store *datastore.Manager
}

func (b *BoxManager) SubRunner(ctx context.Context, numConc int64) (eval.BoxScheduler, error) {
	if err := b.concSem.Acquire(ctx, numConc); err != nil {
		return nil, err
	}

	return &BoxManager{
		numConcurrent: numConc,
		concSem:       semaphore.NewWeighted(numConc),
		memSem:        b.memSem,

		logger: b.logger,

		availableIDs: b.availableIDs,

		parentMgr: b,

		boxGenerator: b.boxGenerator,

		languageVersions:   b.languageVersions,
		supportedLanguages: b.supportedLanguages,

		store: b.store,
	}, nil
}

func (b *BoxManager) NumConcurrent() int64 {
	return b.numConcurrent
}

func (b *BoxManager) getBox(ctx context.Context, memQuota int64) (eval.Sandbox, error) {
	if b.boxGenerator == nil {
		slog.WarnContext(ctx, "Empty box generator")
		return nil, errors.New("empty box generator")
	}
	if memQuota > 0 {
		if err := b.memSem.Acquire(ctx, memQuota); err != nil {
			return nil, err
		}
	}
	box, err := b.boxGenerator(ctx, <-b.availableIDs, memQuota, b.logger)
	if err != nil {
		return nil, err
	}
	// b.logger.Infof("Acquired box %d", box.GetID())
	return box, nil
}

func (b *BoxManager) releaseBox(ctx context.Context, sb eval.Sandbox) {
	q := sb.MemoryQuota()
	if err := sb.Close(); err != nil {
		slog.WarnContext(ctx, "Could not release sandbox", slog.Any("box_id", sb.GetID()), slog.Any("err", err))
	}
	// b.logger.Infof("Yielded back box %d", sb.GetID())
	b.availableIDs <- sb.GetID()
	b.memSem.Release(q)
}

// Close waits for all boxes to finish running
func (b *BoxManager) Close(ctx context.Context) error {
	b.concSem.Acquire(ctx, b.numConcurrent)
	if b.parentMgr != nil {
		b.parentMgr.concSem.Release(b.numConcurrent)
	} else {
		close(b.availableIDs)
	}
	return nil
}

// New creates a new box manager
func New(startingNumber int, count int, maxMemory int64, logger *slog.Logger, dataStore *datastore.Manager, boxGenerator BoxFunc) (*BoxManager, error) {

	if startingNumber < 0 {
		startingNumber = 0
	}

	availableIDs := make(chan int, 3*count)
	for i := 1; i <= 2*count; i++ {
		availableIDs <- i + startingNumber
	}

	bm := &BoxManager{
		concSem:       semaphore.NewWeighted(int64(count)),
		memSem:        semaphore.NewWeighted(maxMemory),
		maxMemory:     maxMemory,
		availableIDs:  availableIDs,
		numConcurrent: int64(count),

		logger: logger,

		parentMgr: nil,

		boxGenerator: boxGenerator,

		supportedLanguages: supportedLanguages(context.Background()),

		store: dataStore,
	}
	return bm, nil
}

func CheckCanRun(ctx context.Context, boxFunc BoxFunc) bool {
	box, err := boxFunc(ctx, 0, 0, slog.Default())
	if err != nil {
		slog.WarnContext(ctx, "Error creating sandbox", slog.Any("err", err))
		return false
	}
	if err := box.Close(); err != nil {
		slog.WarnContext(ctx, "Error closing sandbox", slog.Any("err", err))
		return false
	}
	return true
}

func (mgr *BoxManager) getLangVersions(ctx context.Context) map[string]string {
	mgr.languageVersionsMu.Lock()
	defer mgr.languageVersionsMu.Unlock()
	mgr.languageVersions = make(map[string]string)
	for name, lang := range mgr.supportedLanguages {
		if lang.Disabled {
			continue
		}
		ver, err := tasks.VersionTask(ctx, mgr, lang)
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

func (mgr *BoxManager) Language(name string) *eval.Language {
	lang, ok := mgr.supportedLanguages[name]
	if !ok {
		return nil
	}
	return lang
}

func (mgr *BoxManager) Languages() map[string]*eval.Language {
	// TODO: maybe a maps.Clone()?
	return mgr.supportedLanguages
}

func (mgr *BoxManager) LanguageVersions(ctx context.Context) map[string]string {
	if mgr.languageVersions == nil {
		return mgr.getLangVersions(ctx)
	}
	mgr.languageVersionsMu.RLock()
	defer mgr.languageVersionsMu.RUnlock()
	return maps.Clone(mgr.languageVersions)
}

// TODO: Improve
func (mgr *BoxManager) LanguageFromFilename(filename string) *eval.Language {
	fileExt := path.Ext(filename)
	if fileExt == "" {
		return nil
	}
	// bestLang heuristic to match .cpp to cpp17
	if fileExt == ".cpp" {
		x := mgr.Language("cpp17")
		if x != nil {
			return x
		}
		// Otherwise fall back to earliest cpp version
		best := ""
		for _, lang := range mgr.supportedLanguages {
			if strings.HasPrefix(lang.InternalName, ".cpp") && (best == "" || lang.InternalName < best) {
				best = lang.InternalName
			}
		}
		return mgr.Language(best)
	}
	bestLang := ""
	for k, v := range mgr.Languages() {
		for _, ext := range v.Extensions {
			if ext == fileExt && (bestLang == "" || k < bestLang) {
				bestLang = k
			}
		}
	}
	return mgr.Language(bestLang)
}

func (mgr *BoxManager) RunBox2(ctx context.Context, req *eval.Box2Request, memQuota int64) (*eval.Box2Response, error) {
	loggerOnce.Do(func() {
		cmdAuditLogger = slog.New(slog.NewJSONHandler(&lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "sandbox_runs.log"),
			MaxSize:  200, // MB
			Compress: true,
		}, &slog.HandlerOptions{
			AddSource: false,
		}))
	})

	goodCmd, err := makeGoodCommand(req.Command)
	if err != nil {
		slog.ErrorContext(ctx, "Error running MakeGoodCommand", slog.Any("err", err))
		return nil, err
	}

	if err := mgr.concSem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer mgr.concSem.Release(1)
	box, err := mgr.getBox(ctx, memQuota)
	if err != nil {
		slog.WarnContext(ctx, "Could not get box", slog.Any("err", err))
		return nil, err
	}
	defer mgr.releaseBox(ctx, box)

	for fpath, val := range req.InputByteFiles {
		if val.Mode == 0 {
			val.Mode = 0666
		}
		if err := box.WriteFile(fpath, bytes.NewReader(val.Data), val.Mode); err != nil {
			return nil, err
		}
	}

	for fpath, val := range req.InputBucketFiles {
		bucket, err := mgr.store.Get(val.Bucket)
		if err != nil {
			slog.ErrorContext(ctx, "Error getting bucket", slog.Any("err", err))
			continue
		}

		// Do not reset val.Mode here, since CopyInBox stats and sets the proper mode
		if err := copyInBox(box, bucket, val.Filename, fpath, val.Mode); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				slog.WarnContext(ctx, "Bucket file doesn't exist when copying in sandbox",
					slog.Any("bucket", val.Bucket), slog.String("filename", val.Filename),
					slog.String("target_path", fpath), slog.Int("box_id", box.GetID()),
				)
			}
			return nil, err
		}
	}

	stats, err := box.RunCommand(ctx, goodCmd, req.RunConfig)
	if err != nil {
		return nil, err
	}
	cmdAuditLogger.InfoContext(ctx, "Ran command",
		slog.Any("command", goodCmd),
		slog.Any("stats", stats),
		slog.Any("output_byte_files", req.OutputByteFiles),
		slog.Int64("mem_quota", memQuota),
	)

	resp := &eval.Box2Response{
		Stats:       stats,
		ByteFiles:   make(map[string][]byte),
		BucketFiles: make(map[string]*eval.BucketFile),
	}

	var b bytes.Buffer
	for _, path := range req.OutputByteFiles {
		b.Reset()
		if !box.FileExists(path) {
			continue
		}
		if err := box.ReadFile(path, &b); err != nil {
			return resp, err
		}
		resp.ByteFiles[path] = bytes.Clone(b.Bytes())
	}

	for path, file := range req.OutputBucketFiles {
		if !box.FileExists(path) {
			continue
		}
		if file.Mode == 0 {
			file.Mode = 0666
		}

		bucket, err := mgr.store.Get(file.Bucket)
		if err != nil {
			slog.ErrorContext(ctx, "Error getting bucket", slog.Any("err", err))
			continue
		}

		if err := box.SaveFile(path, bucket, file.Filename, file.Mode); err != nil {
			slog.WarnContext(ctx, "Error saving box file", slog.Any("err", err), slog.String("path", path), slog.Any("bucket", file.Bucket))
			return resp, err
		}
		resp.BucketFiles[path] = &eval.BucketFile{
			Bucket:   file.Bucket,
			Filename: file.Filename,
			Mode:     file.Mode,
		}
	}

	return resp, nil
}

// Copies in box an object from a bucket
func copyInBox(b eval.Sandbox, bucket eval.Bucket, filename string, p2 string, mode fs.FileMode) error {
	file, err := bucket.Reader(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if mode == 0000 {
		stat, err := bucket.Stat(filename)
		if err != nil {
			return err
		}
		mode = stat.Mode()
	}

	return b.WriteFile(p2, file, mode)
}

// makeGoodCommand makes sure it's a full path (with no symlinks) for the command.
// Some languages (like java) are hidden pretty deep in symlinks, and we don't want a hardcoded path that could be different on other platforms.
func makeGoodCommand(command []string) ([]string, error) {
	tmp := slices.Clone(command)

	if strings.HasPrefix(tmp[0], "/box") {
		return tmp, nil
	}

	cmd, err := exec.LookPath(tmp[0])
	if err != nil {
		return nil, err
	}

	cmd2, err := filepath.EvalSymlinks(cmd)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(cmd2, "ccache") {
		cmd = cmd2
	}

	tmp[0] = cmd
	return tmp, nil
}

// supportedLanguages disables all languages that are *not* detected by the system in the current configuration
// It should be run at the start of the execution (and implemented more nicely tbh)
func supportedLanguages(ctx context.Context) map[string]*eval.Language {
	langs := make(map[string]*eval.Language)
	for k, v := range eval.Langs {
		if v.Disabled { // Skip search if already disabled
			continue
		}
		var toSearch []string
		if v.Compiled {
			toSearch = v.CompileCommand
		} else {
			toSearch = v.RunCommand
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

		langs[k] = &v
	}
	return langs
}
