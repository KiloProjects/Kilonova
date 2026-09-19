## Context

Today three files feed the process:

- `config.toml` → `domain/config/config.go`: TOML decoded into `configStruct`, "spread" into package globals `config.Common/Eval/Email/Frontend` plus `kilonova.SetDebugMode/SetHostPrefix/SetDefaultLanguage`. `config.Save` "compactifies" the globals back and rewrites the file. `main.go`'s root `Before` loads it, then immediately rewrites it "for formatting". `sudoapi.UpdateConfig` (`POST /admin/updateConfig`) mutates the globals and rewrites the file again.
- `grader.toml` → `domain/config/grader.go`: `LoadGrader` for `kn grader-serve`. Because the root `Before` runs for subcommands, `grader-serve` cannot start without a `config.toml` it never reads.
- `flags.json` → `domain/config/config_v2.go`: typed flags declared with `config.GenFlag[T]`, persisted on every `Update`, overridable with `KN_FLAG_OVERRIDES`, editable from the admin flags page (bool/string/int only).

The values in `config.toml` fall into two kinds: host wiring that is fixed for a deployment (DB DSN, paths, SMTP credentials, sandbox capacity, remote grader endpoint/token, debug, host prefix) and instance settings an admin edits at runtime (`default_language`, `test_max_mem_kb`, `banned_hot_problems`). `num_concurrent` / `global_max_mem_kb` are exposed in the admin form but only take effect after a restart because `scheduler.New` reads them once.

Existing infrastructure we can lean on: `godotenv.Load()` already runs at startup; `urfave/cli/v3` flags already take `Sources: cli.EnvVars(...)` and support `Destination` pointers; `pgx` already resolves libpq `PG*` environment variables when the DSN is empty; the flag registry already persists and handles `KN_FLAG_OVERRIDES`.

## Goals / Non-Goals

**Goals:**
- One rule: host-bound and secret values are `KN_*` environment variables; admin-editable instance settings are flags in `flags.json`. Nothing else.
- The process never writes a config file other than `flags.json`.
- `kn --help` / `kn grader-serve --help` are the authoritative list of env variables.
- `kn grader-serve` runs with only its own variables set.
- A one-shot converter so existing deployments migrate mechanically.

**Non-Goals:**
- Multitenancy. This change only makes the instance/host split clean enough that it can be built later.
- Plain-HTTP grader listener for in-cluster deployments (TLS stays mandatory per `grader-auth-config`).
- Docker images / compose files. They become straightforward after this change but are a separate deliverable.
- Making the flags admin page edit list-typed flags. `banned_hot_problems` stays editable through the existing admin config form.
- Hot-reload of env variables. They are startup-only by definition.

## Decisions

### D1. Env variables are declared as `urfave/cli` flags with `Destination` into the existing `config.*` structs

Each variable is a `cli.StringFlag` / `cli.IntFlag` / `cli.Int64Flag` / `cli.BoolFlag` with `Sources: cli.EnvVars("KN_...")` and `Destination: &config.Common.LogDir` (etc.). Platform variables live on the root command in `cmd/kn/main.go`; grader variables on the `grader-serve` subcommand in `cmd/kn/grader_serve.go`. `domain/config` keeps only the structs and the flag registry; it loses `toml`, `Load`, `Save`, `spread`, `compactify`.

Why: the ladder stops at "already-installed dependency". We get `--help` documentation, defaults, type parsing, CLI override (`--log-dir`) and `.env` support (godotenv is already loaded before `cmd.Run`) with zero glue code. Consumers such as `config.Common.LogDir` in six packages do not change.

Alternatives: a hand-rolled `os.LookupEnv` + `strconv` loader in `domain/config` (~40 lines, no `--help`, duplicates what cli does); `caarlos0/env` or `kelseyhightower/envconfig` (new dependency for what an installed one does); Viper (far too much).

### D2. Variable naming: `KN_<AREA>_<NAME>`, grouped by which process consumes it

| Variable | Default | Consumer | Replaces |
|---|---|---|---|
| `KN_DATA_DIR` | (required, absolute) | both | `common.data_dir` / parent of `grader.scratch_dir`; the process root: datastore buckets + `logs/` for the platform, `scratch/` + `logs/` for the grader (replaces `common.log_dir` and `grader.scratch_dir`) |

| `KN_DEBUG` | `false` | platform | `common.debug` |
| `KN_HOST_PREFIX` | `http://localhost:8070` | platform | `common.host_prefix` |
| `KN_DB_DSN` | empty → libpq `PG*` vars | platform | `common.db_dsn` |
| `KN_SMTP_HOST` (`host:port`) | empty → mail disabled | platform | `email.enabled` + `email.host` |
| `KN_SMTP_USERNAME` / `KN_SMTP_PASSWORD` / `KN_SMTP_FROM` | empty | platform | `email.*` |
| `KN_EVAL_MODE` | `local` | platform | `eval.mode` |
| `KN_EVAL_REMOTE_ENDPOINT` / `KN_EVAL_REMOTE_TOKEN` | empty (required when remote) | platform | `eval.remote.*` |
| `KN_SANDBOX_NUM_CONCURRENT` | `3` | platform (local mode) and grader | `eval.num_concurrent` / `grader.num_concurrent` |
| `KN_SANDBOX_GLOBAL_MAX_MEM_KB` | `2097152` | same | `*.global_max_mem_kb` |
| `KN_SANDBOX_STARTING_BOX` | `1` | same | `*.starting_box` |
| `KN_GRADER_LISTEN` | `:9000` | grader | `grader.listen` |
| `KN_GRADER_TLS_CERT` / `KN_GRADER_TLS_KEY` | (required) | grader | `grader.cert_file` / `key_file` |

| `KN_GRADER_SCRATCH_TTL_SEC` | `3600` | grader | `grader.scratch_ttl_sec` |
| `KN_GRADER_CLIENT_<NAME>` | (at least one) | grader | `[[grader.client]]` — one variable per client, `NAME` lowercased is the client name |

`KN_SANDBOX_*` are declared on the root command so one definition serves both the platform's local grader and `grader-serve` (root flags are parsed for subcommands). `KN_EVAL_*` is "how the platform evaluates"; `KN_GRADER_*` is "how the grader process serves". `KN_FLAG_OVERRIDES`, `KN_FLAGS_PATH` keep their current meaning; `KN_CONF_PATH` and `KN_GRADER_CONF_PATH` disappear.

`EmailConf.Enabled` becomes a method (`Host != ""`) rather than a separate variable; a mailer with no host is meaningless.

The client registry is one variable per client rather than a list: `KN_GRADER_CLIENT_KILONOVA=<token>` registers client `kilonova`. It is read by scanning `os.Environ()` for the prefix (not a cli flag), so each token can be its own orchestrator secret and adding a client never means editing an existing value. The reserved `priority` field is dropped: nothing consumes it and it can return as `KN_GRADER_CLIENT_<NAME>_PRIORITY` when scheduling uses it. `config-migrate` uppercases the legacy name and maps any character outside `[A-Za-z0-9]` to `_`.

Alternative considered: a single `KN_GRADER_CLIENTS=name=token,...` list in the `KN_FLAG_OVERRIDES` style. Rejected — a list with secrets in it cannot be split across secrets.

### D3. Admin-editable values become flags; `/admin/updateConfig` writes flags

New flags in `sudoapi/flags`:

- `frontend.default_language` (string, `en`)
- `eval.test_max_mem_kb` (int, `655360`)
- `frontend.banned_hot_problems` (`[]int`, `[]`)

`sudoapi.UpdateConfig` keeps its endpoint and JSON shape minus `num_workers` / `global_max_mem`, and calls `flags.X.Update(...)`, which already persists to `flags.json`. The admin form drops the two capacity inputs. `web/web.go` template funcs read the flags; `domain/archive/test` and `sudoapi/admin.go` read the flags.

`kilonova.DefaultLanguage()` stays a global getter because 26 call sites use it and the `kilonova` package cannot import `sudoapi/flags`. `main.go` calls `kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())` after loading flags and again inside the `SetOnFlagUpdate` hook, so edits from either the config form or the flags page take effect. `SetDefaultLanguage` stops panicking on an unknown value and falls back to `en` with a warning, because the value is now user input at runtime, not a boot-time file.

Why not delete `UpdateConfig` and use the flags page for everything: the flags page only renders bool/string/int flags; `banned_hot_problems` is a list and the config form already parses it. Extending the flags page is more code than keeping the three-field form.

### D4. Validation happens where the value is consumed, not at parse time

`InitializeBaseAPI` already rejects a non-absolute data dir; it keeps doing so with a message naming `KN_DATA_DIR`. `getRemoteRunner` fails when endpoint or token is empty. `grader-serve` fails when cert, key or clients are empty. Type errors (non-integer in `KN_SANDBOX_NUM_CONCURRENT`) are rejected by cli before any action runs. Subcommands that never touch the DB (`config-migrate`, `grader-serve`) therefore run with no platform variables set.

### D5. `kn config-migrate` is the only remaining TOML reader

`cmd/kn/config_migrate.go` owns private copies of the legacy TOML structs (moved from `domain/config`). It reads `--config` (default `./config.toml`) and `--grader-config` (default `./grader.toml`), skips whichever is absent, prints `KEY=value` lines in dotenv format to stdout (values quoted as needed), and writes `default_language`, `test_max_mem_kb`, `banned_hot_problems` into the flags file via the existing registry (`flags.X.Update`, which triggers the save hook). It never modifies or deletes the TOML files. Running it twice produces the same output and flags.

The `[email] enabled = false` with a non-empty host case is mapped to an empty `KN_SMTP_HOST` so behaviour is preserved.

### D6. `.env.example` replaces both example TOML files

One documented file listing every variable from D2 with comments. `.env` is already gitignored and already loaded by godotenv, so local development keeps a "config file" without the process owning it. `runkn.sh` needs no change: the process loads `.env` after `sudo`.

## Risks / Trade-offs

- [Operators upgrade the binary without migrating] → boot fails fast with "KN_DATA_DIR must be set to an absolute path" before touching the DB; the changelog and the error message point at `kn config-migrate`.
- [`sudo` drops exported environment] → `.env` in the working directory is read by the process itself, and `runkn.sh` already forwards `KN_FLAG_OVERRIDES` explicitly. Documented in `.env.example`.
- [Secrets in environment are visible via `/proc/<pid>/environ`] → same threat model as the mounted TOML today (readable by the same uid); acceptable, and standard for containers. A `_FILE` suffix convention for secrets can be added later without breaking anything.
- [Root-level `KN_SANDBOX_*` flags show under `kn --help` rather than `kn grader-serve --help`] → the help text says "used by the local grader and by grader-serve".
- [Admin form loses the capacity fields] → they never applied without a restart; the values are now visible in `kn --help` and the compose file instead.
- [`[]int` flag type is new to the registry] → `sneakUpdate` and `SaveConfigV2` are generic over JSON, but the admin flags page's `GetFlags[T]` calls will simply not list it; covered by a unit test that round-trips the flag through `SaveConfigV2`/`LoadConfigV2`.

## Migration Plan

1. Deploy the new binary alongside the old files. Run `kn config-migrate -c config.toml --grader-config grader.toml -f flags.json > .env` on the platform host (and with only `--grader-config` on the grader host). Review the output, move secrets into the orchestrator's secret store if desired.
2. Start `kn main` / `kn grader-serve` with the environment set. Confirm boot logs, then delete the TOML files.
3. Rollback: the TOML files are untouched by the tool, so the previous binary starts as before. `flags.json` gains three keys the old binary logs as unknown and ignores.

## Open Questions

- None blocking. A `_FILE` suffix convention for secrets is deferred to the Docker packaging work.

