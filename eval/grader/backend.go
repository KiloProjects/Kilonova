package grader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/box"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scratch"
	"github.com/spf13/afero"
)

// Backend is the sandbox side of grading: the box scheduler (local isolate or a
// remote grader), its scratch store and the language inventory. Build one per
// process and share it between every tenant's Handler, so box IDs and the
// memory budget are handed out once.
type Backend struct {
	sched   eval.Box3Scheduler
	scratch eval.Scratch
	langs   eval.LanguageManager
}

// NewBackend connects to the sandbox described by conf. scratchDir is only used
// in local mode.
func NewBackend(ctx context.Context, conf config.EvalConf, scratchDir string) (*Backend, error) {
	initLogger()
	if conf.IsRemote() {
		return newRemoteBackend(ctx, conf)
	}
	return newLocalBackend(ctx, conf, scratchDir)
}

// runner wraps the shared scheduler with one tenant's datastore.
func (b *Backend) runner(store eval.Store) eval.BoxScheduler {
	return scheduler.NewBox2Wrapper(b.scratch, store, b.sched)
}

func (b *Backend) Close(ctx context.Context) error {
	return b.sched.Close(ctx)
}

func newLocalBackend(ctx context.Context, conf config.EvalConf, scratchDir string) (*Backend, error) {
	var boxFunc scheduler.BoxFunc
	var boxVersion = "NONE"
	if scheduler.CheckCanRun(ctx, box.New) {
		boxFunc = box.New
		boxVersion = box.IsolateVersion()
	} else if scheduler.CheckCanRun(ctx, box.NewStupid) && conf.AllowInsecureSandbox {
		slog.WarnContext(ctx, "Secure sandbox not found. Using stupid sandbox")
		boxFunc = box.NewStupid
		boxVersion = "stupid"
	}
	if boxFunc == nil {
		return nil, errors.New("no local sandbox available and eval mode is not remote")
	}

	if err := os.MkdirAll(scratchDir, 0777); err != nil {
		return nil, fmt.Errorf("couldn't create scratch dir: %w", err)
	}
	scratchFS := afero.NewBasePathFs(afero.NewOsFs(), scratchDir)
	sc := scratch.New(scratchFS)
	// Fixed dir + orphan janitor, so files left by a crash are reaped on the next run.
	// NOTE: TTL hardcoded to grader-serve's default
	go scratch.PeriodicSweep(ctx, scratchFS, 15*time.Minute, time.Hour, graderLogger)

	slog.InfoContext(ctx, "Trying to spin up local grader")
	bm, err := scheduler.New(conf.StartingBox, conf.NumConcurrent, conf.GlobalMaxMem, graderLogger, sc, boxFunc)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "Running local grader", slog.String("version", boxVersion))

	// Version probes only move byte files, so they need no tenant datastore.
	langs := scheduler.NewLanguageManager(ctx, scheduler.NewBox2Wrapper(sc, nil, bm), graderLogger)
	return &Backend{sched: bm, scratch: sc, langs: langs}, nil
}

// newRemoteBackend wires the platform against a remote grader: JSON control plane
// (GraderClient) + HTTP data plane (scratch), both on the grader's one TLS+token
// endpoint. No local sandbox is created.
func newRemoteBackend(ctx context.Context, conf config.EvalConf) (*Backend, error) {
	rc := conf.Remote
	if rc.Endpoint == "" || rc.Token == "" {
		return nil, fmt.Errorf("KN_EVAL_MODE=remote requires KN_EVAL_REMOTE_ENDPOINT and KN_EVAL_REMOTE_TOKEN")
	}
	// One transport for both planes: the control plane and the /scratch data
	// plane must never disagree about whether the grader is verified.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if conf.AllowInsecureSandbox {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in, local/dev only
		slog.WarnContext(ctx, "Grader certificate verification is DISABLED (KN_SANDBOX_ALLOW_INSECURE). Never do this in production",
			slog.String("endpoint", rc.Endpoint))
	}

	// Bounded per-transfer so a dropped connection fails the eval fast instead of
	// hanging (design D2). ponytail: 60s covers the largest test file; lift to a
	// config field if real transfers approach it.
	scratchClient := &http.Client{Timeout: 60 * time.Second, Transport: transport}
	remoteScratch, err := scratch.NewHTTP(scratchClient, rc.Endpoint+"/scratch", rc.Token)
	if err != nil {
		return nil, fmt.Errorf("couldn't set up remote scratch: %w", err)
	}

	client := scheduler.NewGraderClient(&http.Client{Transport: transport}, rc.Endpoint, rc.Token)
	langs, err := scheduler.NewRemoteLanguageManager(ctx, client, graderLogger)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch remote language inventory: %w", err)
	}

	slog.InfoContext(ctx, "Running remote grader", slog.String("endpoint", rc.Endpoint))
	return &Backend{sched: client, scratch: remoteScratch, langs: langs}, nil
}
