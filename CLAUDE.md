# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Kilonova is a competitive programming platform: a Go monolith (HTTP frontend + JSON API + submission grader) on PostgreSQL, with a Vite/TypeScript asset pipeline embedded into the binary.

## Commands

Build order matters — generated files are `go:embed`ed, so generation and asset builds must run **before** `go build`/`go test`:

```sh
go generate ./...            # translations.toml -> _translations.json (x2), chroma.css, templ generate
pnpm -C web/assets install
pnpm -C web/assets build     # -> web/static/misc + manifest.{bundled,vendored,css}.json
go build ./cmd/kn            # produces ./kn
go vet ./... && go test ./...
go test ./eval/scheduler -run TestScratchRoundTripOverWire   # single test
golangci-lint run            # config in .golangci.yaml
```

- `./runkn.sh` does generate → asset build → build → run in a restart loop (needs `sudo` for isolate sandboxing).
- Containers: one `Dockerfile`, two targets — `--target platform` (`kn main`, no toolchains) and `--target grader` (`kn grader-serve`, isolate + every language). `compose.yaml` runs Postgres + platform + grader; the platform image cannot sandbox, so containers always run `KN_EVAL_MODE=remote`. See `docs/deployment.md`.
- `pnpm -C web/assets watch` rebuilds assets on change (three parallel Vite builds, one per bundle).
- `./kn main` runs the platform; `./kn grader-serve` runs a standalone remote grader. Startup config is `KN_*` environment variables (`.env` in the cwd is loaded by the process; `.env.example` and `kn --help` list them); runtime-editable settings live in the `flags` table in Postgres, seeded once from `flags.json` (`-f`) if that file exists. `./kn config-migrate` converts a legacy `config.toml`/`grader.toml`.
- CLI tools are Go tool deps (`go tool templ`).
- Docs site: `mkdocs` (see mkdocs.yml, `scripts/build_docs.sh`).

## Architecture

Layering is strict, top to bottom: **transport (`web`, `api`) → `sudoapi` → `db`/`domain` → PostgreSQL**. Handlers never touch `db` directly.

- **`sudoapi`** — `BaseAPI` is the god object holding the pgx pool, `db.DB`, datastore buckets, mailer, markdown renderer, session cache, Discord session, OIDC provider, and the grader hook. Everything is wired in `InitializeBaseAPI`/`GetBaseAPI` and passed down to `web`, `api`, and `eval/grader`. Methods here do permission checks and return errors created with `kilonova.Statusf(code, ...)`.
- **Errors** — `kilonova.Statusf` attaches an HTTP status to an error; status 500 falls back to a plain `fmt.Errorf`. `kilonova.ErrorCode`/`MaybeErrorCode` recover it at the transport layer. Package-level sentinels (`ErrNotFound`, …) live in `status.go`.
- **`web`** — server-rendered pages on chi. Two template systems coexist: legacy `html/template` files under `web/templ` (embedded) and newer [templ](https://templ.guide) components under `web/views`, `web/components`, `web/tutils` (`*.templ` + generated `*_templ.go`, both committed). HTMX + Alpine drive interactivity. `web/handlers.go` and `web/web.go` are large; routing lives in `web.go`.
- **`api`** — `/api` is the legacy hand-rolled JSON API (`api.go`, `StatusData` envelopes); `/api/v2` is [huma](https://huma.rocks) with OpenAPI, session/scope middleware in `middlewarev2.go`.
- **`db`** — thin pgx layer, one file per entity, raw SQL with `pgx.RowToStructByNameLax`. Schema changes are **append-only migration files** in `db/psql_schema/NNN.name.sql` registered in the `Migrations` list in `db/migrations.go` (IDs are the ordering key and must not be reused); `999.views.sql` is a special migration re-run on every startup. Migrations run at boot when the `behavior.db.run_migrations` flag is on.
- **`domain/`** — newer, more isolated pieces: `config` (env-populated startup structs + flag registry), `datastore` (afero-backed buckets for tests/subtests/attachments/avatars), `user`, `archive` (problem import/export).
- **Configuration & flags** — startup-only, host-bound settings (DB DSN, dirs, SMTP, sandbox capacity, grader wiring) are `KN_*` environment variables declared as urfave/cli flags in `cmd/kn/main.go` and `cmd/kn/grader_serve.go` with `Destination` pointers into `config.Common`/`config.Eval`/`config.Email`/`config.GraderConf`; required values are validated where they are consumed. Everything runtime-tunable is a flag declared with `config.GenFlag[T]("dotted.key", default, description)` in `sudoapi/flags/{backend,frontend}_flags.go`, persisted one row per flag in the `flags` table, and editable from the admin UI. Flags load in `initBase` (after the pool and migrations), not in the CLI `Before`, because `kn grader-serve` shares that hook and has no database. Add new toggles as flags, not config fields; gate routes with `rt.checkFlag(...)`.
- **Grading** — `eval/grader` polls the DB for waiting submissions (`feeder.go`) and evaluates them (`grader.go`) through an `eval.BoxScheduler`. Two modes, chosen by `KN_EVAL_MODE`:
 *local* runs `eval/scheduler` in-process over `eval/box` (isolate sandboxes; `stupidbox` is the insecure fallback, allowed only with `KN_SANDBOX_ALLOW_INSECURE`), *remote* talks JSON over HTTP to a separate `kn grader-serve` process (`eval/scheduler/rpc_{client,server}.go`; `POST /run/box3`, `POST /run/multibox3`, `GET /languages`; file transfer via `eval/scratch` on `/scratch/{id}`; platform and grader must be the same build). Language definitions and version probing live in `eval/language` + `scheduler/langmgr.go`.
- **Assets** — `web/assets/vite.config.ts` runs one build per `BUNDLE` (`bundled` = app.ts, `vendored` = vendored.ts, `css`), because templates call IIFE globals from inline scripts. `web/assets.go` merges the per-bundle manifests at init and `assets.Asset(ctx, "app.ts")` resolves the hashed `/static/misc/...` URL. `web/assets_test.go` verifies the manifests point at files that actually got embedded — it fails if the asset build was skipped.
- **i18n** — `translations.toml` is the source of truth; `go generate` renders `_translations.json` for both Go (`kilonova.GetText(lang, key, args...)`) and the frontend. Only `en` and `ro` are valid languages.

## Conventions

- Logging is `log/slog` with context, enforced by sloglint: `slog.InfoContext(ctx, "msg", slog.Any("err", err))` — attributes only, no mixed args, always pass a context.
- `errcheck` is disabled but `gocritic`, `unparam`, `sqlclosecheck` and staticcheck-all are on.
- Spec-driven work under `openspec/` (see `.claude/skills/openspec-*` and the `opsx:*` commands) — proposals in `openspec/changes/`, accepted specs in `openspec/specs/`.
