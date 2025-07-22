package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	flagPath = flag.String("flags", "./flags.json", "Flag configuration path")

	name                       = flag.String("name", "", "Client name")
	appType                    = flag.String("app-type", "web", "Application type")
	authorID                   = flag.Int("author-id", 0, "Author ID")
	devMode                    = flag.Bool("dev-mode", false, "Developer mode")
	allowedRedirects           = flag.String("allowed-redirects", "", "Allowed redirects")
	allowedPostLogoutRedirects = flag.String("allowed-post-logout-redirects", "", "Allowed post logout redirects")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	config.SetConfigPath(*confPath)
	config.SetConfigV2Path(*flagPath)
	if err := config.Load(ctx); err != nil {
		slog.ErrorContext(ctx, "Couldn't load config", slog.Any("err", err))
		os.Exit(1)
	}
	if err := config.LoadConfigV2(ctx, false); err != nil {
		slog.ErrorContext(ctx, "Could not load flags", slog.Any("err", err))
		os.Exit(1)
	}

	if err := Kilonova(ctx); err != nil {
		slog.ErrorContext(ctx, "Script run failed", slog.Any("err", err))
		os.Exit(1)
	}
}

func Kilonova(ctx context.Context) error {
	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova OAuth Client Creator")

	base, err := sudoapi.InitializeBaseAPI(ctx)
	if err != nil {
		return err
	}
	defer base.Close()

	var authAppType auth.ApplicationType
	switch *appType {
	case "web":
		authAppType = auth.ApplicationTypeWeb
	case "native":
		authAppType = auth.ApplicationTypeNative
	case "userAgent":
		authAppType = auth.ApplicationTypeUserAgent
	default:
		return fmt.Errorf("invalid app type: %s", *appType)
	}
	authorID := *authorID
	devMode := *devMode
	allowedRedirects := strings.Split(*allowedRedirects, ",")
	allowedPostLogoutRedirects := strings.Split(*allowedPostLogoutRedirects, ",")

	id, secret, err := base.CreateClient(ctx, *name, authAppType, authorID, devMode, allowedRedirects, allowedPostLogoutRedirects)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "Client created", slog.String("id", id.String()), slog.String("secret", secret))
	return nil
}
