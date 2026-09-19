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
		&cli.StringFlag{Name: "listen", Usage: "host:port for the web server", Value: "localhost:8070", Sources: cli.EnvVars("KN_LISTEN"), Destination: &config.Server.Listen},
		&cli.StringFlag{Name: "true-ip-header", Usage: "Header carrying the client IP when behind a reverse proxy (e.g. X-Forwarded-For); empty otherwise", Sources: cli.EnvVars("KN_TRUE_IP_HEADER"), Destination: &config.Server.TrueIPHeader},
		&cli.StringFlag{Name: "prometheus-listen", Usage: "host:port for the Prometheus /metrics exporter; empty disables it", Sources: cli.EnvVars("KN_PROMETHEUS_LISTEN"), Destination: &config.Server.PrometheusListen},

		&cli.StringFlag{Name: "db-dsn", Usage: "PostgreSQL DSN; empty falls back to the libpq PG* variables", Sources: cli.EnvVars("KN_DB_DSN"), Destination: &config.DB.DSN},
		&cli.BoolFlag{Name: "db-run-migrations", Usage: "Run schema migrations on startup", Value: true, Sources: cli.EnvVars("KN_DB_RUN_MIGRATIONS"), Destination: &config.DB.RunMigrations},
		&cli.BoolFlag{Name: "db-log-sql", Usage: "Log every SQL query (debugging)", Sources: cli.EnvVars("KN_DB_LOG_SQL"), Destination: &config.DB.LogQueries},
		&cli.BoolFlag{Name: "db-count-queries", Usage: "Count SQL queries per request (debugging)", Sources: cli.EnvVars("KN_DB_COUNT_QUERIES"), Destination: &config.DB.CountQueries},

		&cli.StringFlag{Name: "maxmind-db", Usage: "Path to the MaxMind GeoLite2-City database", Value: "/usr/share/GeoIP/GeoLite2-City.mmdb", Sources: cli.EnvVars("KN_MAXMIND_DB"), Destination: &config.Integrations.MaxMindDB},
		&cli.BoolFlag{Name: "otel-enabled", Usage: "Export OpenTelemetry traces and logs (endpoint from the OTEL_* variables)", Sources: cli.EnvVars("KN_OTEL_ENABLED"), Destination: &config.Integrations.OtelEnabled},
		&cli.StringFlag{Name: "discord-token", Usage: "Discord bot token; empty disables the integration", Sources: cli.EnvVars("KN_DISCORD_TOKEN"), Destination: &config.Integrations.DiscordToken},
		&cli.StringFlag{Name: "discord-client-id", Sources: cli.EnvVars("KN_DISCORD_CLIENT_ID"), Destination: &config.Integrations.DiscordClientID},
		&cli.StringFlag{Name: "discord-client-secret", Sources: cli.EnvVars("KN_DISCORD_CLIENT_SECRET"), Destination: &config.Integrations.DiscordClientSecret},
		&cli.StringFlag{Name: "openai-token", Usage: "OpenAI API key; empty disables statement translation/transcription", Sources: cli.EnvVars("KN_OPENAI_TOKEN"), Destination: &config.Integrations.OpenAIToken},
		&cli.StringFlag{Name: "openai-model", Usage: "Model for statement translation", Value: "gpt-5.6-sol", Sources: cli.EnvVars("KN_OPENAI_MODEL"), Destination: &config.Integrations.OpenAIModel},
		&cli.StringFlag{Name: "openai-vision-model", Usage: "Model for PDF statement transcription", Value: "gpt-5.6-sol", Sources: cli.EnvVars("KN_OPENAI_VISION_MODEL"), Destination: &config.Integrations.OpenAIVisionModel},

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
		&cli.BoolFlag{Name: "sandbox-ensure-cg-keeper", Usage: "Ensure isolate-cg-keeper is running (local grader and grader-serve)", Sources: cli.EnvVars("KN_SANDBOX_ENSURE_CG_KEEPER"), Destination: &config.Eval.EnsureCGKeeper},
		&cli.BoolFlag{Name: "sandbox-allow-insecure", Usage: "Allow the insecure stupidbox fallback when isolate is missing; never in production", Sources: cli.EnvVars("KN_SANDBOX_ALLOW_INSECURE"), Destination: &config.Eval.AllowInsecureSandbox},
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

			prometheus.InitMetrics(ctx, config.Server.PrometheusListen)

			if err := Kilonova(ctx); err != nil {
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
