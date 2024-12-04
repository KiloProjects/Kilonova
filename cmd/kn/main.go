package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	flagPath = flag.String("flags", "./flags.json", "Flag configuration path")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	config.SetConfigPath(*confPath)
	config.SetConfigV2Path(*flagPath)
	if err := config.Load(); err != nil {
		slog.ErrorContext(ctx, "Could not load config", slog.Any("err", err))
		os.Exit(1)
	}
	if err := config.LoadConfigV2(ctx, false); err != nil {
		slog.ErrorContext(ctx, "Could not load flags", slog.Any("err", err))
		os.Exit(1)
	}

	initLogger(config.Common.Debug)

	if err := os.MkdirAll(config.Common.LogDir, 0755); err != nil {
		slog.ErrorContext(ctx, "Error initializing log directory", slog.Any("err", err))
		os.Exit(1)
	}

	// save the config for formatting
	if err := config.Save(); err != nil {
		slog.ErrorContext(ctx, "Error saving config", slog.Any("err", err))
		os.Exit(1)
	}

	// save the flags in case any new ones were added
	if err := config.SaveConfigV2(ctx); err != nil {
		slog.ErrorContext(ctx, "Error saving flags", slog.Any("err", err))
		os.Exit(1)
	}

	if err := Kilonova(); err != nil {
		slog.ErrorContext(ctx, "Error running Kilonova", slog.Any("err", err))
		os.Exit(1)
	}

	os.Exit(0)
}

func init() {
	initLogger(true)
}
