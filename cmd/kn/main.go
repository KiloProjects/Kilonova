package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/infra/prometheus"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

// Startup-only values that live outside the config structs.
var (
	debugMode  bool
	hostPrefix string
	flagsPath  string
)

// envFlags declares the platform's KN_* environment contract. Each entry writes
// straight into the config structs; `kn --help` is the reference for operators.
// Validation of required values happens where they are consumed (design D4).
func envFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "flags", Aliases: []string{"f"}, Usage: "Path to the runtime flags file", Value: "./flags.json", Sources: cli.EnvVars("KN_FLAGS_PATH"), Destination: &flagsPath},

		&cli.StringFlag{Name: "data-dir", Usage: "Absolute root directory: datastore buckets and logs/ for the platform, scratch/ and logs/ for grader-serve",
			Sources: cli.EnvVars("KN_DATA_DIR"), Destination: &config.Common.DataDir},
		&cli.BoolFlag{Name: "debug", Usage: "Debug mode (verbose logs, slower)", Sources: cli.EnvVars("KN_DEBUG"), Destination: &debugMode},
		&cli.StringFlag{Name: "host-prefix", Usage: "Public URL of this instance", Value: "http://localhost:8070", Sources: cli.EnvVars("KN_HOST_PREFIX"), Destination: &hostPrefix},
		&cli.StringFlag{Name: "db-dsn", Usage: "PostgreSQL DSN; empty falls back to the libpq PG* variables", Sources: cli.EnvVars("KN_DB_DSN"), Destination: &config.Common.DBDSN},

		&cli.StringFlag{Name: "smtp-host", Usage: "SMTP host:port; empty disables mail", Sources: cli.EnvVars("KN_SMTP_HOST"), Destination: &config.Email.Host},
		&cli.StringFlag{Name: "smtp-username", Sources: cli.EnvVars("KN_SMTP_USERNAME"), Destination: &config.Email.Username},
		&cli.StringFlag{Name: "smtp-password", Sources: cli.EnvVars("KN_SMTP_PASSWORD"), Destination: &config.Email.Password},
		&cli.StringFlag{Name: "smtp-from", Usage: "From address; defaults to the username", Sources: cli.EnvVars("KN_SMTP_FROM"), Destination: &config.Email.SendAs},

		&cli.StringFlag{Name: "eval-mode", Usage: "Grader mode: local (in-process sandbox) or remote (kn grader-serve)", Value: "local", Sources: cli.EnvVars("KN_EVAL_MODE"), Destination: &config.Eval.Mode},
		&cli.StringFlag{Name: "eval-remote-endpoint", Usage: "Remote grader base URL, e.g. https://grader:9000", Sources: cli.EnvVars("KN_EVAL_REMOTE_ENDPOINT"), Destination: &config.Eval.Remote.Endpoint},
		&cli.StringFlag{Name: "eval-remote-token", Usage: "Bearer token registered on the remote grader", Sources: cli.EnvVars("KN_EVAL_REMOTE_TOKEN"), Destination: &config.Eval.Remote.Token},

		// Shared by the local grader and by grader-serve.
		&cli.IntFlag{Name: "sandbox-num-concurrent", Usage: "Sandboxes run in parallel (local grader and grader-serve)", Value: 3, Sources: cli.EnvVars("KN_SANDBOX_NUM_CONCURRENT"), Destination: &config.Eval.NumConcurrent},
		&cli.Int64Flag{Name: "sandbox-global-max-mem-kb", Usage: "Memory budget in KB across all sandboxes (local grader and grader-serve)", Value: 2097152, Sources: cli.EnvVars("KN_SANDBOX_GLOBAL_MAX_MEM_KB"), Destination: &config.Eval.GlobalMaxMem},
		&cli.IntFlag{Name: "sandbox-starting-box", Usage: "First isolate box ID (local grader and grader-serve)", Value: 1, Sources: cli.EnvVars("KN_SANDBOX_STARTING_BOX"), Destination: &config.Eval.StartingBox},
	}
}

func main() {
	ctx := context.Background()
	// .env is read by the process itself, so it survives `sudo`. Existing
	// environment always wins over the file.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		slog.ErrorContext(ctx, "Error loading .env file", slog.Any("err", err))
	}

	cmd := &cli.Command{
		Flags: envFlags(),
		Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
			if m := config.Eval.Mode; m != "local" && m != "remote" {
				return nil, fmt.Errorf("KN_EVAL_MODE must be local or remote, got %q", m)
			}
			kilonova.SetDebugMode(debugMode)
			kilonova.SetHostPrefix(hostPrefix)

			if err := config.LoadConfigV2(ctx, flagsPath, false); err != nil {
				return nil, fmt.Errorf("error loading flags: %w", err)
			}
			kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())
			config.SetOnFlagUpdate(func() {
				kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())
				if err := config.SaveConfigV2(context.Background(), flagsPath); err != nil {
					slog.WarnContext(context.Background(), "Couldn't save flag", slog.Any("err", err))
				}
			})

			// save the flags in case any new ones were added
			if err := config.SaveConfigV2(ctx, flagsPath); err != nil {
				return nil, fmt.Errorf("error saving flags: %w", err)
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			if err := config.RequireDataDir(); err != nil {
				return err
			}
			if err := os.MkdirAll(config.Common.LogDir(), 0755); err != nil {

				return fmt.Errorf("error creating log directory: %w", err)
			}

			initLogger(kilonova.DebugMode(), true)

			prometheus.InitMetrics(ctx)

			if err := Kilonova(ctx, command); err != nil {
				return fmt.Errorf("error running Kilonova: %w", err)
			}
			return nil
		},
		Commands: []*cli.Command{
			newOauth,
			printableContestants,
			problemDiagnostics,
			submissionSaver,
			aiTools,
			contestUtils,
			graderServe,
			configMigrate,
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(context.Background(), "Error running CLI", slog.Any("err", err))
		os.Exit(1)
	}

	os.Exit(0)
}

func init() {
	initLogger(true, false)
}
