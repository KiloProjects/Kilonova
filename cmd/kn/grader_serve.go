package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/eval/box"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scratch"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

var graderConf config.GraderConf

// graderServe runs the sandbox executor as a standalone remote grader.
// Its root is KN_DATA_DIR (scratch/ and logs/ under it); sandbox capacity
// comes from the root KN_SANDBOX_* variables and the client registry from
// KN_GRADER_CLIENT_<NAME>=<token> variables.

var graderServe = &cli.Command{
	Name:  "grader-serve",
	Usage: "Run a standalone remote grader",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "listen", Usage: "host:port for the grader HTTPS server", Value: ":9000", Sources: cli.EnvVars("KN_GRADER_LISTEN"), Destination: &graderConf.Listen},
		&cli.StringFlag{Name: "tls-cert", Usage: "TLS server certificate file", Sources: cli.EnvVars("KN_GRADER_TLS_CERT"), Destination: &graderConf.CertFile},
		&cli.StringFlag{Name: "tls-key", Usage: "TLS server key file", Sources: cli.EnvVars("KN_GRADER_TLS_KEY"), Destination: &graderConf.KeyFile},
		&cli.IntFlag{Name: "scratch-ttl-sec", Usage: "Orphan scratch GC TTL in seconds; must exceed the longest eval", Value: 3600, Sources: cli.EnvVars("KN_GRADER_SCRATCH_TTL_SEC"), Destination: &graderConf.ScratchTTLSec},
	},
	Action: func(ctx context.Context, command *cli.Command) error {
		g := graderConf
		if err := config.RequireDataDir(); err != nil { // scratch/ and logs/ live under KN_DATA_DIR
			return err
		}
		if g.CertFile == "" || g.KeyFile == "" {

			return fmt.Errorf("KN_GRADER_TLS_CERT and KN_GRADER_TLS_KEY are required")
		}
		// One variable per client: KN_GRADER_CLIENT_<NAME>=<token>.
		clients, err := config.GraderClientsFromEnv(os.Environ())
		if err != nil {
			return err
		}

		boxFunc := box.New
		if !scheduler.CheckCanRun(ctx, boxFunc) {
			return fmt.Errorf("secure sandbox (isolate) is unavailable; refusing to start remote grader")
		}

		scratchDir := config.Common.ScratchDir()
		if err := os.MkdirAll(scratchDir, 0o777); err != nil {
			return fmt.Errorf("create scratch dir: %w", err)
		}
		scratchFS := afero.NewBasePathFs(afero.NewOsFs(), scratchDir)
		sc := scratch.New(scratchFS)

		bm, err := scheduler.New(config.Eval.StartingBox, config.Eval.NumConcurrent, config.Eval.GlobalMaxMem, slog.Default(), sc, boxFunc)
		if err != nil {
			return fmt.Errorf("create box manager: %w", err)
		}
		defer bm.Close(ctx)

		// LanguageManager probes versions through the Box2 path (byte files only,
		// no datastore), so a nil-store wrapper over the box manager suffices.
		langMgr := scheduler.NewLanguageManager(ctx, scheduler.NewBox2Wrapper(sc, nil, bm), slog.Default())

		registry := scheduler.NewClientRegistry()
		for _, cl := range clients {
			if err := registry.Add(cl.Token, cl.Name, ""); err != nil {
				return err
			}
		}

		// Orphan janitor: TTL >> max eval so it can only ever reap crash-orphans.
		ttl := time.Duration(g.ScratchTTLSec) * time.Second
		go scratch.PeriodicSweep(ctx, scratchFS, ttl/4, ttl, slog.Default())

		// Control plane (JSON) and data plane (/scratch) on one listener, behind
		// one bearer-token check.
		mux := http.NewServeMux()
		mux.Handle("/", scheduler.NewGraderServer(bm, langMgr).Handler())
		mux.Handle(scheduler.ScratchHandler(scratchFS))

		// /healthz is the one unauthenticated path: an orchestrator has no token.
		// It hangs off an outer mux so everything else still goes through Auth.
		// The sandbox was proven usable by CheckCanRun above, so this is a static
		// answer rather than a per-probe isolate run.
		var draining atomic.Bool
		outer := http.NewServeMux()
		outer.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			if draining.Load() {
				http.Error(w, "draining", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, "ok")
		})
		outer.Handle("/", registry.Auth(mux))

		ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		httpSrv := &http.Server{Addr: g.Listen, Handler: outer}
		errCh := make(chan error, 1)
		go func() {
			slog.InfoContext(ctx, "Remote grader listening", slog.String("addr", g.Listen), slog.Int("clients", len(clients)))
			if err := httpSrv.ListenAndServeTLS(g.CertFile, g.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("grader server: %w", err)
				return
			}
			errCh <- nil
		}()

		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
		}

		slog.InfoContext(ctx, "Shutting down remote grader")
		draining.Store(true)
		shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("grader shutdown: %w", err)
		}
		return nil
	},
}
