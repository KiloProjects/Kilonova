package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/KiloProjects/kilonova/net/discord"
	"github.com/KiloProjects/kilonova/net/email"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/spf13/afero"
)

// initBase is the composition root for the platform stack: everything that
// comes from the environment contract is turned into a dependency here and
// handed to sudoapi, which never reads config itself.
func initBase(ctx context.Context) (*sudoapi.BaseAPI, error) {
	if err := config.RequireDataDir(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.Common.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("couldn't create data dir: %w", err)
	}
	mgr, err := datastore.New(afero.NewBasePathFs(afero.NewOsFs(), config.Common.DataDir))
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize data store: %w", err)
	}

	// A nil mailer means mail is not configured.
	var mailer kilonova.Mailer
	if config.Email.Enabled() {
		mailer, err = email.NewMailer(config.Email.Host, config.Email.Username, config.Email.Password, config.Email.SendAs)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't initialize mailer. Make sure you entered the correct information", slog.Any("err", err))
		}
	}

	pgxDB, err := postgres.NewDB(ctx, postgres.Config{
		DSN:          config.DB.DSN,
		CountQueries: config.DB.CountQueries,
		LogQueries:   config.DB.LogQueries,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to DB: %w", err)
	}
	slog.InfoContext(ctx, "Connected to DB")
	if config.DB.RunMigrations {
		if err := postgres.RunMigrations(ctx, pgxDB, db.Migrations); err != nil {
			return nil, fmt.Errorf("couldn't run migrations: %w", err)
		}
	}

	if err := loadFlags(ctx, db.NewPSQL(pgxDB)); err != nil {
		return nil, err
	}

	dc, err := discord.New(discord.Config{
		Token:        config.Integrations.DiscordToken,
		ClientID:     config.Integrations.DiscordClientID,
		ClientSecret: config.Integrations.DiscordClientSecret,
		RedirectURL:  kilonova.HostURL().JoinPath("api/webhook/discord_callback").String(),
	})
	if err != nil {
		return nil, fmt.Errorf("discord: %w", err)
	}

	return sudoapi.GetBaseAPI(ctx, pgxDB, mgr, mailer, dc)
}

// loadFlags brings the runtime flag registry up now that a database exists:
// stored values first, then the one-time import of a legacy flags.json for
// deployments upgrading from the file, then this process's overrides. It also
// registers the persistence callback, so an admin edit writes exactly one row.
//
// A failure to read the store is fatal on purpose: continuing on compiled-in
// defaults would silently undo every setting an admin ever changed.
func loadFlags(ctx context.Context, store config.FlagStore) error {
	empty, err := config.LoadFlagsFromDB(ctx, store)
	if err != nil {
		return fmt.Errorf("couldn't load flags: %w", err)
	}
	if empty {
		if err := config.ImportFlagsFile(ctx, store, flagsPath); err != nil {
			return fmt.Errorf("couldn't import flags file: %w", err)
		}
	}
	config.ApplyFlagOverrides(ctx)
	kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())

	config.SetOnFlagUpdate(func(name string) {
		kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := config.PersistFlag(ctx, store, name); err != nil {
			slog.WarnContext(ctx, "Couldn't persist flag", slog.String("flag", name), slog.Any("err", err))
		}
	})
	return nil
}
