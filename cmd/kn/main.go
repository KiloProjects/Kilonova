package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/infra/prometheus"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v3"
)

func main() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		slog.ErrorContext(ctx, "Error loading .env file", slog.Any("err", err))
	}

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
				Value:   "./config.toml",
				Sources: cli.EnvVars("KN_CONF_PATH"),
			},
			&cli.StringFlag{
				Name:    "flags",
				Aliases: []string{"f"},
				Usage:   "Path to flags file",
				Value:   "./flags.toml",
				Sources: cli.EnvVars("KN_FLAGS_PATH"),
			},
		},
		Before: func(ctx context.Context, command *cli.Command) (context.Context, error) {
			configPath := command.String("config")
			flagsPath := command.String("flags")

			if err := config.Load(ctx, configPath); err != nil {
				return nil, fmt.Errorf("error loading config: %w", err)
			}
			if err := config.LoadConfigV2(ctx, flagsPath, false); err != nil {
				return nil, fmt.Errorf("error loading flags: %w", err)
			}

			// save the config for formatting
			if err := config.Save(configPath); err != nil {
				return nil, fmt.Errorf("error saving config: %w", err)
			}

			// save the flags in case any new ones were added
			if err := config.SaveConfigV2(ctx, flagsPath); err != nil {
				return nil, fmt.Errorf("error saving flags: %w", err)
			}
			return ctx, nil
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			if err := os.MkdirAll(config.Common.LogDir, 0755); err != nil {
				return fmt.Errorf("error creating log directory: %w", err)
			}

			initLogger(kilonova.DebugMode(), true)

			prometheus.InitMetrics(ctx)

			if err := Kilonova(); err != nil {
				return fmt.Errorf("error running Kilonova: %w", err)
			}
			return nil
		},
		Commands: []*cli.Command{
			newOauth,
			printableContestants,
			problemDiagnostics,
			submissionSaver,
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.ErrorContext(context.Background(), "Error running CLI", slog.Any("err", err))
	}

	os.Exit(0)
}

func init() {
	initLogger(true, false)
}
