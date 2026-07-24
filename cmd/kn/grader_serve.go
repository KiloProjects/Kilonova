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

// graderServe runs the sandbox executor as a standalone remote grader.
var graderServe = &cli.Command{
	Name:  "grader-serve",
	Usage: "Run a standalone remote grader",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "grader-config",
			Aliases: []string{"gc"},
			Usage:   "Path to grader.toml",
			Value:   "./grader.toml",
			Sources: cli.EnvVars("KN_GRADER_CONF_PATH"),
		},
	},
	Action: func(ctx context.Context, command *cli.Command) error {
		gc, err := config.LoadGrader(ctx, command.String("grader-config"))
		if err != nil {
			return fmt.Errorf("load grader config: %w", err)
		}
		g := gc.Grader

		boxFunc := box.New
		if !scheduler.CheckCanRun(ctx, boxFunc) {
			return fmt.Errorf("secure sandbox (isolate) is unavailable; refusing to start remote grader")
		}

		if err := os.MkdirAll(g.ScratchDir, 0o777); err != nil {
			return fmt.Errorf("create scratch dir: %w", err)
		}
		scratchFS := afero.NewBasePathFs(afero.NewOsFs(), g.ScratchDir)
		sc := scratch.New(scratchFS)

		bm, err := scheduler.New(g.StartingBox, g.NumConcurrent, g.GlobalMaxMem, slog.Default(), sc, boxFunc)
		if err != nil {
			return fmt.Errorf("create box manager: %w", err)
		}
		defer bm.Close(ctx)

		// LanguageManager probes versions through the Box2 path (byte files only,
		// no datastore), so a nil-store wrapper over the box manager suffices.
		langMgr := scheduler.NewLanguageManager(ctx, scheduler.NewBox2Wrapper(sc, nil, bm), slog.Default())

		registry := scheduler.NewClientRegistry()
		for _, cl := range g.Clients {
			if err := registry.Add(cl.Token, cl.Name, cl.Priority); err != nil {
				return err
			}
		}

		// Orphan janitor: TTL >> max eval so it can only ever reap crash-orphans.
		ttl := time.Duration(g.ScratchTTLSec) * time.Second
		go scratch.PeriodicSweep(ctx, scratchFS, ttl/4, ttl, slog.Default())

		srv := scheduler.NewGraderServer(bm, langMgr)
		path, handler := srv.Handler(registry)
		mux := http.NewServeMux()
		mux.Handle(path, handler)

		httpSrv := &http.Server{Addr: g.Listen, Handler: mux}
		slog.InfoContext(ctx, "Remote grader listening", slog.String("addr", g.Listen), slog.Int("clients", len(g.Clients)))
		if err := httpSrv.ListenAndServeTLS(g.CertFile, g.KeyFile); err != nil {
			return fmt.Errorf("grader server: %w", err)
		}
		return nil
	},
}
