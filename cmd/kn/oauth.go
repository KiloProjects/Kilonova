package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/urfave/cli/v3"
)

var newOauth = &cli.Command{
	Name: "new-oauth",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "Client name",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "app-type",
			Aliases:  []string{"t"},
			Usage:    "Application type",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "author-id",
			Aliases:  []string{"a"},
			Usage:    "User ID of app author",
			Required: true,
		},
		&cli.BoolFlag{
			Name:    "dev-mode",
			Aliases: []string{"d"},
			Usage:   "Developer mode",
		},
		&cli.StringSliceFlag{
			Name:     "allowed-redirects",
			Aliases:  []string{"r"},
			Usage:    "Allowed redirects",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:    "allowed-post-logout-redirects",
			Aliases: []string{"p"},
			Usage:   "Allowed post logout redirects",
		},
	},
	Action: func(ctx context.Context, command *cli.Command) error {
		// Print welcome message
		slog.InfoContext(ctx, "Starting Kilonova OAuth Client Creator")

		base, err := sudoapi.InitializeBaseAPI(ctx)
		if err != nil {
			return err
		}
		defer base.Close()

		var authAppType auth.ApplicationType
		switch command.String("app-type") {
		case "web":
			authAppType = auth.ApplicationTypeWeb
		case "native":
			authAppType = auth.ApplicationTypeNative
		case "userAgent":
			authAppType = auth.ApplicationTypeUserAgent
		default:
			return fmt.Errorf("invalid app type: %s", command.String("app-type"))
		}
		authorID := command.Int("author-id")
		devMode := command.Bool("dev-mode")
		allowedRedirects := command.StringSlice("allowed-redirects")
		allowedPostLogoutRedirects := command.StringSlice("allowed-post-logout-redirects")

		id, secret, err := base.CreateClient(ctx, command.String("name"), authAppType, authorID, devMode, allowedRedirects, allowedPostLogoutRedirects)
		if err != nil {
			return err
		}

		slog.InfoContext(ctx, "Client created", slog.String("id", id.String()), slog.String("secret", secret))
		return nil
	},
}
