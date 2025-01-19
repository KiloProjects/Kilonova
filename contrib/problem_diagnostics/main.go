package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	config.SetConfigPath(*confPath)
	if err := config.Load(ctx); err != nil {
		slog.ErrorContext(ctx, "Couldn't load config", slog.Any("err", err))
		os.Exit(1)
	}

	if err := Kilonova(ctx); err != nil {
		slog.ErrorContext(ctx, "Script run failed", slog.Any("err", err))
		os.Exit(1)
	}

	os.Exit(0)
}

func Kilonova(ctx context.Context) error {
	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova Quick Problem Diagnostics Runner")

	base, err := sudoapi.InitializeBaseAPI(ctx)
	if err != nil {
		return err
	}
	defer base.Close()

	pbs, err := base.Problems(ctx, kilonova.ProblemFilter{})
	if err != nil {
		return err
	}

	for _, pb := range pbs {
		diags, err := base.ProblemDiagnostics(ctx, pb)
		if err != nil {
			fmt.Printf("- Error loading problem diagnostics for %d (URL: %s/problems/%d ): %v\n", pb.ID, config.Common.HostPrefix, pb.ID, err)
			continue
		}
		if len(diags) > 0 {
			fmt.Printf("- Diagnostics for problem %d (URL: %s/problems/%d, published: %t):\n", pb.ID, config.Common.HostPrefix, pb.ID, pb.Visible)
			for _, diag := range diags {
				fmt.Printf("\t- %s: %s\n", diag.Level.String(), diag.Message)
			}
		}
	}

	return nil
}
