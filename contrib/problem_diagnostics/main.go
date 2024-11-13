package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() {
	flag.Parse()

	config.SetConfigPath(*confPath)
	if err := config.Load(); err != nil {
		zap.S().Fatal(err)
	}

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}

func Kilonova() error {
	ctx := context.Background()

	// Print welcome message
	zap.S().Infof("Starting Kilonova Quick Problem Diagnostics Runner")

	base, err := sudoapi.InitializeBaseAPI(context.Background())
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

func initLogger(debug bool) {
	core := kilonova.GetZapCore(debug, os.Stdout)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)
}

func init() {
	initLogger(true)
}
