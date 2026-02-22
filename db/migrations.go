package db

import (
	"context"
	"embed"
	"path"

	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/jackc/pgx/v5"
)

//go:embed psql_schema
var migrationFiles embed.FS

var Migrations = postgres.MigrationConfig{
	SchemaTable: "kn_schema_version",
	// Order is important. Add new migrations at the end of the list.
	Migrations: []postgres.Migration{
		{
			ID:      1,
			Name:    "Base schema",
			Handler: runFile("001.base.sql"),
		},
		{
			ID:      2,
			Name:    "Discord Integration",
			Handler: runFile("002.discord_oauth_secret.sql"),
		},
		{
			ID:      3,
			Name:    "Discord Avatar as main",
			Handler: runFile("003.use_discord_avatar.sql"),
		},
		{
			ID:      4,
			Name:    "Add Stripe support to donations workflow",
			Handler: runFile("004.stripe.sql"),
		},
		{
			ID:      5,
			Name:    "External resources support",
			Handler: runFile("005.external_resources.sql"),
		},
		{
			ID:      6,
			Name:    "External resources language support",
			Handler: runFile("006.external_resource_lang.sql"),
		},
		{
			ID:      7,
			Name:    "OAuth/OIDC support",
			Handler: runFile("007.oauth2_oidc.sql"),
		},
		{
			ID:      8,
			Name:    "Communication tasks support",
			Handler: runFile("008.problem_task_type.sql"),
		},
		{
			ID:      9,
			Name:    "Add support for setting custom filenames to submission code",
			Handler: runFile("009.source_file_name.sql"),
		},
		{
			ID:      10,
			Name:    "Add support for submissions having multiple files",
			Handler: runFile("010.split_submission_code.sql"),
		},
		{
			ID:      11,
			Name:    "Add support for whitelisting IPs in contests",
			Handler: runFile("011.contest_whitelist.sql"),
		},
		{
			ID:      12,
			Name:    "Add IPs to submissions",
			Handler: runFile("012.submission_ip.sql"),
		},
	},
	// Run every time a migrate up happens
	SpecialMigrations: []postgres.Migration{
		{
			ID:      999,
			Name:    "Views",
			Handler: runFile("999.views.sql"),
		},
	},
}

func runFile(name string) func(ctx context.Context, tx pgx.Tx) error {
	return func(ctx context.Context, tx pgx.Tx) error {
		f, err := migrationFiles.ReadFile(path.Join("psql_schema", name))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, string(f))
		return err
	}
}
