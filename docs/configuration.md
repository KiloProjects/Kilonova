---
title: Configuration and the v26.09 migration
---

# Overview

Starting with v26.09, Kilonova no longer reads `config.toml` or `grader.toml`. Configuration is split in two:

| Kind | Where it lives | Examples |
|---|---|---|
| Host wiring, set once per deployment | `KN_*` environment variables | database DSN, data and log directories, SMTP credentials, sandbox capacity, remote grader endpoint and TLS files |
| Instance settings, editable at runtime | `flags.json` (admin UI → *Flags*, or the config form on the admin page) | default language, per-test memory cap, banned hot problems, every feature toggle |

The process never writes a config file other than `flags.json`. Environment variables are documented by the binary itself: `kn --help` lists the platform variables, `kn grader-serve --help` the grader ones, and `.env.example` in the repository has all of them with comments.

!!! note "Precedence"
    An explicit CLI flag (`--data-dir /srv/kn`) wins over the process environment, which wins over a `.env` file in the working directory, which wins over the built-in default. `.env` is read by the process itself, so it also works under `sudo ./kn main`.

# Migrating an existing deployment

The new binary refuses to start without `KN_DATA_DIR` (and, in remote mode, the grader endpoint and token). Convert your files first:

```sh
# platform host
./kn config-migrate -c config.toml --grader-config grader.toml -f flags.json > .env

# grader host (no config.toml there)
./kn config-migrate --grader-config grader.toml > .env
```

The command:

- prints one `KEY=value` line per non-empty legacy value to **stdout**, quoted where needed, so redirecting to `.env` is enough;
- writes `default_language`, `test_max_mem_kb` and `banned_hot_problems` into the flags file given by `-f` instead of emitting them;
- reads the same flags file for keys that stopped being flags (listen address, migrations, DB debugging, MaxMind path, OpenTelemetry, Prometheus, sandbox policy, Discord and OpenAI credentials) and emits their `KN_*` lines; the platform drops those keys from `flags.json` on its next start;
- reports on **stderr** which flags it wrote, any file it could not find, and any legacy key that has no equivalent (the never-used `priority` field on grader clients is dropped);
- never modifies or deletes the TOML files, and gives identical output when run twice.

Then start `kn main` / `kn grader-serve` as before, check the boot log, and delete the TOML files. If both files are present on one host and both set sandbox capacity, the grader's value wins and a warning is printed.

!!! tip "Docker and orchestrators"
    You don't need `.env` at all. Pass the variables through `environment:` / `env_file:` in Compose, or as secrets in Kubernetes, and mount only the data directory and `flags.json`. Leaving `KN_DB_DSN` empty makes the driver use the standard libpq variables (`PGHOST`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`, `PGSSLMODE`, ...), which most Postgres images and operators already provide.

## Rollback

The migration tool leaves `config.toml` and `grader.toml` untouched, so the previous binary starts exactly as before. `flags.json` gains three keys the old binary logs as unknown and ignores.

# Variable reference

## Platform (`kn main`)

| Variable | Default | Replaces | Notes |
|---|---|---|---|
| `KN_FLAGS_PATH` | `./flags.json` | `-f` | unchanged |
| `KN_DATA_DIR` | *(required)* | `common.data_dir` | absolute root directory: datastore buckets plus `logs/` |
| `KN_DEBUG` | `false` | `common.debug` | |
| `KN_HOST_PREFIX` | `http://localhost:8070` | `common.host_prefix` | public URL; used for CORS and as the OIDC issuer |
| `KN_DB_DSN` | *(empty)* | `common.db_dsn` | empty falls back to `PG*` variables |
| `KN_DB_RUN_MIGRATIONS` | `true` | flag `behavior.db.run_migrations` | |
| `KN_DB_LOG_SQL` | `false` | flag `behavior.db.log_sql` | debugging |
| `KN_DB_COUNT_QUERIES` | `false` | flag `behavior.db.count_queries` | debugging |
| `KN_LISTEN` | `localhost:8070` | flags `server.listen.host` / `port` | web server bind address |
| `KN_TRUE_IP_HEADER` | *(empty)* | flag `server.listen.true_ip_header` | e.g. `X-Forwarded-For` behind a reverse proxy |
| `KN_PROMETHEUS_LISTEN` | *(empty)* | flags `integrations.prometheus.enabled` / `port` | e.g. `:8071`; empty disables `/metrics` |
| `KN_MAXMIND_DB` | `/usr/share/GeoIP/GeoLite2-City.mmdb` | flag `integrations.maxmind.db_path` | |
| `KN_OTEL_ENABLED` | `false` | flag `integrations.otel.enabled` | endpoint from the standard `OTEL_*` variables |
| `KN_DISCORD_TOKEN` | *(empty)* | flags `integrations.discord.enabled` / `token` | a set token enables the integration |
| `KN_DISCORD_CLIENT_ID` / `KN_DISCORD_CLIENT_SECRET` | *(empty)* | flags `integrations.discord.client_*` | |
| `KN_OPENAI_TOKEN` | *(empty)* | flag `integrations.openai.token` | enables statement translation/transcription; no longer visible to templates |
| `KN_OPENAI_MODEL` / `KN_OPENAI_VISION_MODEL` | `gpt-5.6-sol` | flags `integrations.openai.default_model` / `vision_model` | the per-request model field is gone |

| `KN_SMTP_HOST` | *(empty)* | `email.enabled` + `email.host` | `host:port`; empty disables mail |
| `KN_SMTP_USERNAME` | | `email.username` | |
| `KN_SMTP_PASSWORD` | | `email.password` | |
| `KN_SMTP_FROM` | username | `email.sendAs` | |
| `KN_EVAL_MODE` | `local` | `eval.mode` | `local` or `remote` |
| `KN_EVAL_REMOTE_ENDPOINT` | | `eval.remote.endpoint` | required in remote mode, e.g. `https://grader:9000` |
| `KN_EVAL_REMOTE_TOKEN` | | `eval.remote.token` | required in remote mode |

## Sandbox capacity (`kn main` in local mode, and `kn grader-serve`)

| Variable | Default | Replaces |
|---|---|---|
| `KN_SANDBOX_NUM_CONCURRENT` | `3` | `eval.num_concurrent` / `grader.num_concurrent` |
| `KN_SANDBOX_GLOBAL_MAX_MEM_KB` | `2097152` | `eval.global_max_mem_kb` / `grader.global_max_mem_kb` |
| `KN_SANDBOX_STARTING_BOX` | `1` | `eval.starting_box` / `grader.starting_box` |
| `KN_SANDBOX_ENSURE_CG_KEEPER` | `false` | flag `feature.grader.ensure_keeper` |
| `KN_SANDBOX_ALLOW_INSECURE` | `false` | flag `feature.grader.force_secure_sandbox` (inverted) |

These were editable from the admin page before v26.09 but only took effect after a restart, so they are now environment-only.

## Remote grader (`kn grader-serve`)

| Variable | Default | Replaces | Notes |
|---|---|---|---|
| `KN_DATA_DIR` | *(required)* | parent of `grader.scratch_dir` | absolute root directory: `scratch/` (served on `/scratch`) and `logs/` |
| `KN_GRADER_LISTEN` | `:9000` | `grader.listen` | |
| `KN_GRADER_TLS_CERT` | *(required)* | `grader.cert_file` | |
| `KN_GRADER_TLS_KEY` | *(required)* | `grader.key_file` | |
| `KN_GRADER_SCRATCH_TTL_SEC` | `3600` | `grader.scratch_ttl_sec` | orphan cleanup; keep it far above the longest eval |
| `KN_GRADER_CLIENT_<NAME>` | *(at least one)* | `[[grader.client]]` | one variable per client; `KN_GRADER_CLIENT_KILONOVA=<token>` registers client `kilonova` |


`grader-serve` ignores the platform variables other than `KN_DATA_DIR` and `KN_SANDBOX_*`. The migration tool derives its `KN_DATA_DIR` from the parent of the legacy `scratch_dir`, so `/var/lib/kilonova/scratch` keeps working unchanged.

## Moved to `flags.json`

| Flag | Default | Replaces |
|---|---|---|
| `frontend.default_language` | `en` | `common.default_language` |
| `eval.test_max_mem_kb` | `655360` | `common.test_max_mem_kb` |
| `frontend.banned_hot_problems` | `[]` | `frontend.banned_hot_problems` |

All three remain editable from the config form on the admin page. An invalid default language now falls back to English with a warning instead of refusing to start.

## Removed

The flags listed in the tables above with a `flag ...` origin no longer exist in `flags.json`; they were read once at startup, so editing them from the admin page never took effect anyway. `feature.grader.isolate_config_path` was never read and is dropped without replacement.

`common.log_dir`
 and `grader.scratch_dir` have no replacement: both processes use `$KN_DATA_DIR/logs`, and the grader uses `$KN_DATA_DIR/scratch`. Move or symlink your old log directory if you want the history in one place.

`KN_CONF_PATH`,
 `KN_GRADER_CONF_PATH`, `-c/--config` on `kn main` and `--grader-config` on `kn grader-serve`. `KN_FLAG_OVERRIDES` (`key=value,key=value` for one-off flag overrides) keeps working.
