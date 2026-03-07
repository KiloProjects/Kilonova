package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/urfave/cli/v3"
)

var problemDiagnostics = &cli.Command{
	Name: "problem-diagnostics",
	Action: func(ctx context.Context, _ *cli.Command) error {
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
				fmt.Printf("- Error loading problem diagnostics for %d (URL: %s/problems/%d ): %v\n", pb.ID, kilonova.HostPrefix(), pb.ID, err)
				continue
			}
			if len(diags) > 0 {
				fmt.Printf("- Diagnostics for problem %d (URL: %s/problems/%d, published: %t):\n", pb.ID, kilonova.HostPrefix(), pb.ID, pb.Visible)
				for _, diag := range diags {
					fmt.Printf("\t- %s: %s\n", diag.Level.String(), diag.Message)
				}
			}
		}

		return nil
	},
}
