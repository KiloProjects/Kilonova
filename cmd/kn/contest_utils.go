package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/KiloProjects/kilonova/internal/contestgen"
	"github.com/urfave/cli/v3"
)

var contestUtils = &cli.Command{
	Name: "contest-utils",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "profiles",
			Value: "./profiles.json",
			Usage: "Profiles file",
		},
		&cli.BoolFlag{
			Name:  "dryRun",
			Usage: "Dry run to check data",
			Value: false,
		},
	},
	Action: func(ctx context.Context, command *cli.Command) error {
		ctx, base, err := initBase(ctx)
		if err != nil {
			return err
		}
		defer base.Close()

		return runContestUtils(ctx, command, contestgen.LocalBackend(base))
	},
}

func runContestUtils(ctx context.Context, command *cli.Command, backend contestgen.Backend) error {
	auditLogFile, err := os.Create("./generator-" + time.Now().Format(time.RFC3339) + ".log")
	if err != nil {
		return err
	}
	defer func() {
		if err := auditLogFile.Close(); err != nil {
			slog.WarnContext(ctx, "Error closing audit log", slog.Any("err", err))
		}
	}()
	auditLog := slog.New(slog.NewJSONHandler(auditLogFile, &slog.HandlerOptions{AddSource: true}))

	slog.InfoContext(ctx, "Starting Kilonova Contest Registration Manager")

	raw, err := os.ReadFile(command.String("profiles"))
	if err != nil {
		return err
	}
	cfg, err := contestgen.Parse(raw)
	if err != nil {
		return err
	}

	res, err := contestgen.Run(ctx, cfg, backend, auditLog, command.Bool("dryRun"))
	if err != nil {
		return err
	}
	files, err := res.Files()
	if err != nil {
		return err
	}
	if out, ok := files["requestsOut.json"]; ok {
		fmt.Println(string(out))
	}
	for name, data := range files {
		if err := os.WriteFile("./"+name, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
