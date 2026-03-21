// knbox is the Kilonova sandbox service.
//
// It exposes an eval.BoxScheduler (backed by local isolate sandboxes) over HTTP
// so that the Kilonova monolith can delegate all sandbox execution to a
// dedicated Linux host while the rest of the platform runs anywhere.
//
// Both the monolith and knbox must mount the same data directory so that
// BucketFile references can be resolved on both sides without file streaming.
//
// Usage:
//
//	knbox --listen :8091 --data-dir /var/lib/kilonova/data --log-dir /var/log/kilonova \
//	      --num-concurrent 4 --global-max-mem-kb 4194304 --auth-token <secret>
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"

	"github.com/KiloProjects/kilonova/eval/box"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scheduler/server"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/KiloProjects/kilonova/domain/datastore"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	cmd := &cli.Command{
		Name:  "knbox",
		Usage: "Kilonova sandbox execution service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen",
				Aliases: []string{"l"},
				Usage:   "Address to listen on",
				Value:   ":8091",
				Sources: cli.EnvVars("KNBOX_LISTEN"),
			},
			&cli.StringFlag{
				Name:    "data-dir",
				Usage:   "Data directory (must match the monolith's common.data_dir)",
				Value:   "./data",
				Sources: cli.EnvVars("KNBOX_DATA_DIR"),
			},
			&cli.StringFlag{
				Name:    "log-dir",
				Usage:   "Log directory",
				Value:   "./logs",
				Sources: cli.EnvVars("KNBOX_LOG_DIR"),
			},
			&cli.IntFlag{
				Name:    "starting-box",
				Usage:   "First isolate box ID to use",
				Value:   0,
				Sources: cli.EnvVars("KNBOX_STARTING_BOX"),
			},
			&cli.IntFlag{
				Name:    "num-concurrent",
				Usage:   "Maximum number of concurrent sandbox executions",
				Value:   2,
				Sources: cli.EnvVars("KNBOX_NUM_CONCURRENT"),
			},
			&cli.Int64Flag{
				Name:    "global-max-mem-kb",
				Usage:   "Global memory limit across all sandboxes, in kilobytes",
				Value:   2 * 1024 * 1024, // 2 GB
				Sources: cli.EnvVars("KNBOX_GLOBAL_MAX_MEM_KB"),
			},
			&cli.StringFlag{
				Name:    "auth-token",
				Usage:   "Shared secret required in Authorization: Bearer header (leave empty to disable)",
				Sources: cli.EnvVars("KNBOX_AUTH_TOKEN"),
			},
		},
		Action: run,
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(ctx, "knbox exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	listenAddr := cmd.String("listen")
	dataDir := cmd.String("data-dir")
	logDir := cmd.String("log-dir")
	startingBox := cmd.Int("starting-box")
	numConcurrent := cmd.Int("num-concurrent")
	globalMaxMemKB := cmd.Int64("global-max-mem-kb")
	authToken := cmd.String("auth-token")

	// Create log and data directories.
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Set up structured logging to both stderr and a rotating log file.
	logLevel := slog.LevelInfo
	fileHandler := slog.NewJSONHandler(&lumberjack.Logger{
		Filename: path.Join(logDir, "knbox.log"),
		MaxSize:  80,
		Compress: true,
	}, &slog.HandlerOptions{Level: logLevel})
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(newMultiHandler(fileHandler, stderrHandler))
	slog.SetDefault(logger)

	// Initialise the shared datastore (same root as the monolith).
	dataFS := afero.NewBasePathFs(afero.NewOsFs(), dataDir)
	store, err := datastore.New(dataFS)
	if err != nil {
		return err
	}

	// Verify that isolate is available.
	if !scheduler.CheckCanRun(ctx, box.New) {
		return errors.New("isolate sandbox is not available on this host; knbox requires a Linux system with isolate installed")
	}

	boxLogger := slog.New(slog.NewJSONHandler(&lumberjack.Logger{
		Filename: path.Join(logDir, "knbox_grader.log"),
		MaxSize:  80,
		Compress: true,
	}, &slog.HandlerOptions{Level: logLevel}))

	bm, err := scheduler.New(startingBox, numConcurrent, globalMaxMemKB, boxLogger, store, box.New)
	if err != nil {
		return err
	}
	defer bm.Close(ctx)

	slog.InfoContext(ctx, "knbox starting",
		slog.String("listen", listenAddr),
		slog.String("data_dir", dataDir),
		slog.Int("num_concurrent", numConcurrent),
		slog.Int64("global_max_mem_kb", globalMaxMemKB),
		slog.String("isolate_version", box.IsolateVersion()),
	)

	svc := server.New(bm, authToken, logger)
	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: svc.Handler(),
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "Shutting down knbox")
		if err := httpServer.Shutdown(context.Background()); err != nil {
			slog.ErrorContext(ctx, "HTTP shutdown error", slog.Any("err", err))
		}
	}()

	slog.InfoContext(ctx, "knbox ready", slog.String("addr", listenAddr))
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// multiHandler fans out log records to multiple slog.Handler implementations.
type multiHandler struct{ handlers []slog.Handler }

func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var retErr error
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil && retErr == nil {
				retErr = err
			}
		}
	}
	return retErr
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
