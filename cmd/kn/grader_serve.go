package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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

		httpSrv := &http.Server{Addr: g.Listen, Handler: registry.Auth(mux)}
		slog.InfoContext(ctx, "Remote grader listening", slog.String("addr", g.Listen), slog.Int("clients", len(clients)))
		if err := httpSrv.ListenAndServeTLS(g.CertFile, g.KeyFile); err != nil {
			return fmt.Errorf("grader server: %w", err)
		}
		return nil
	},
}
