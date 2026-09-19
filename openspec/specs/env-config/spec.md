# env-config Specification

## Purpose

Defines the `KN_*` environment-variable contract that supplies every host-bound, startup-only setting to the platform and the grader, the validation and fallback rules around it, and the admin-editable settings that live in `flags.json` instead.

## Requirements

### Requirement: Host-bound configuration comes from `KN_*` environment variables
The platform and the grader SHALL read every startup-only, host-bound setting from environment variables prefixed `KN_`, declared as CLI flags so that `kn --help` and `kn grader-serve --help` list each variable with its default. Precedence SHALL be: explicit CLI flag, then process environment, then a `.env` file in the working directory, then the built-in default. The platform SHALL accept `KN_DATA_DIR`, `KN_DEBUG`, `KN_HOST_PREFIX`, `KN_DB_DSN`, `KN_SMTP_HOST`, `KN_SMTP_USERNAME`, `KN_SMTP_PASSWORD`, `KN_SMTP_FROM`, `KN_EVAL_MODE`, `KN_EVAL_REMOTE_ENDPOINT`, `KN_EVAL_REMOTE_TOKEN`, `KN_SANDBOX_NUM_CONCURRENT`, `KN_SANDBOX_GLOBAL_MAX_MEM_KB` and `KN_SANDBOX_STARTING_BOX`. The grader SHALL accept `KN_DATA_DIR`, `KN_SANDBOX_*`, `KN_GRADER_LISTEN`, `KN_GRADER_TLS_CERT`, `KN_GRADER_TLS_KEY`, `KN_GRADER_SCRATCH_TTL_SEC` and any number of `KN_GRADER_CLIENT_<NAME>` variables. A value that cannot be parsed into the variable's type SHALL abort startup with an error naming the variable.

#### Scenario: Platform boots from environment only
- **WHEN** `kn main` starts with `KN_DATA_DIR` and `KN_DB_DSN` set and no `config.toml` present
- **THEN** it initialises the datastore under `KN_DATA_DIR`, connects using `KN_DB_DSN`, and does not attempt to read or write `config.toml`

#### Scenario: `.env` file supplies variables for local development
- **WHEN** the working directory contains a `.env` file with `KN_DATA_DIR=/srv/kn/data` and the process environment does not set it
- **THEN** the platform uses `/srv/kn/data` as the data directory

#### Scenario: Process environment wins over `.env`
- **WHEN** `.env` sets `KN_DEBUG=true` and the process environment sets `KN_DEBUG=false`
- **THEN** debug mode is off

#### Scenario: Malformed numeric variable aborts startup
- **WHEN** `KN_SANDBOX_NUM_CONCURRENT=lots`
- **THEN** the command exits non-zero before any action runs, and the error names `KN_SANDBOX_NUM_CONCURRENT`

#### Scenario: Help lists the variables
- **WHEN** an operator runs `kn --help`
- **THEN** every platform variable above is shown with its environment name and default

### Requirement: Required variables are validated where they are consumed
Validation SHALL fail fast with an error naming the variable, and SHALL only run for the command that consumes the value, so subcommands that do not need the platform stack run without it. `KN_DATA_DIR` MUST be an absolute path for both `kn main` and `kn grader-serve`; log files SHALL be written under `<KN_DATA_DIR>/logs`, the grader's scratch root SHALL be `<KN_DATA_DIR>/scratch`, and there SHALL be no separate log- or scratch-directory variable. When `KN_EVAL_MODE=remote`, `KN_EVAL_REMOTE_ENDPOINT` and `KN_EVAL_REMOTE_TOKEN` MUST both be non-empty. `KN_EVAL_MODE` MUST be `local` or `remote`. `kn grader-serve` MUST have non-empty `KN_GRADER_TLS_CERT`, `KN_GRADER_TLS_KEY` and at least one `KN_GRADER_CLIENT_<NAME>` variable.

#### Scenario: Relative data dir is rejected
- **WHEN** `kn main` starts with `KN_DATA_DIR=./data`
- **THEN** it exits with an error stating that `KN_DATA_DIR` must be an absolute path

#### Scenario: Remote mode without endpoint is rejected
- **WHEN** `KN_EVAL_MODE=remote` and `KN_EVAL_REMOTE_ENDPOINT` is empty
- **THEN** the grader handler fails to start with an error naming `KN_EVAL_REMOTE_ENDPOINT`

#### Scenario: Grader without clients is rejected
- **WHEN** `kn grader-serve` starts with no `KN_GRADER_CLIENT_<NAME>` variable set
- **THEN** it exits with an error naming `KN_GRADER_CLIENT_<NAME>` before opening a listener


#### Scenario: Grader runs without platform variables
- **WHEN** `kn grader-serve` starts with only `KN_DATA_DIR`, `KN_GRADER_*` and `KN_SANDBOX_*` set and no `KN_DB_DSN`
- **THEN** it starts normally, serves scratch from `<KN_DATA_DIR>/scratch` and writes its logs under `<KN_DATA_DIR>/logs`


### Requirement: Database connection falls back to libpq environment
When `KN_DB_DSN` is empty the platform SHALL connect using the standard libpq variables (`PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`, …) as resolved by the pgx driver.

#### Scenario: Connect with PG* variables only
- **WHEN** `KN_DB_DSN` is unset and `PGHOST`, `PGUSER`, `PGPASSWORD`, `PGDATABASE` are set
- **THEN** the platform connects to that database

### Requirement: Mail is enabled by the presence of an SMTP host
The mailer SHALL be enabled if and only if `KN_SMTP_HOST` is non-empty. `KN_SMTP_HOST` MUST be `host:port`. `KN_SMTP_FROM` SHALL default to `KN_SMTP_USERNAME` when empty.

#### Scenario: No SMTP host disables mail
- **WHEN** `KN_SMTP_HOST` is empty
- **THEN** email verification and other mail features report mail as unavailable and no mailer is constructed

#### Scenario: Malformed SMTP host
- **WHEN** `KN_SMTP_HOST=smtp.example.com` (no port)
- **THEN** startup logs a warning that the mailer could not be initialised and continues with mail disabled

### Requirement: Admin-editable instance settings are flags in `flags.json`
The default UI language, the per-test maximum memory cap and the banned hot-problem list SHALL be flags `frontend.default_language` (string, default `en`), `eval.test_max_mem_kb` (int, default `655360`) and `frontend.banned_hot_problems` (list of ints, default empty), persisted in `flags.json` like every other flag and overridable through `KN_FLAG_OVERRIDES`. `POST /admin/updateConfig` SHALL update these flags and SHALL no longer accept `num_workers` or `global_max_mem`. The default language SHALL be validated at read time: a value other than `en` or `ro` SHALL behave as `en` with a logged warning, never a panic.

#### Scenario: Admin changes default language
- **WHEN** an admin submits `default_lang=ro` to `/admin/updateConfig`
- **THEN** `flags.json` contains `frontend.default_language: "ro"`, and new visitors are served Romanian without a restart

#### Scenario: Admin bans hot problems
- **WHEN** an admin submits `banned_hot_pbs=[1,2]`
- **THEN** `flags.json` stores `frontend.banned_hot_problems: [1,2]`, the hot-problem view is refreshed, and the list survives a restart

#### Scenario: Invalid default language in flags file
- **WHEN** `flags.json` contains `frontend.default_language: "xx"`
- **THEN** the platform starts, logs a warning, and serves English

#### Scenario: Capacity fields are gone
- **WHEN** an admin submits `num_workers=8` to `/admin/updateConfig`
- **THEN** the field is ignored and sandbox capacity remains what `KN_SANDBOX_NUM_CONCURRENT` set at boot

### Requirement: The process never reads or writes a platform config file
The platform SHALL NOT read `config.toml`, the grader SHALL NOT read `grader.toml`, and no command SHALL write either file. `flags.json` SHALL remain the only file the process writes. The `KN_CONF_PATH` and `KN_GRADER_CONF_PATH` variables and the `-c/--config` and `--grader-config` options of `kn main` / `kn grader-serve` SHALL be removed.

#### Scenario: Stale config.toml is ignored
- **WHEN** a `config.toml` with a different `db_dsn` exists in the working directory and `KN_DB_DSN` is set
- **THEN** the platform connects using `KN_DB_DSN` and the file is left untouched

#### Scenario: Only flags.json is written at boot
- **WHEN** `kn main` starts in an empty directory with the required variables set
- **THEN** after startup the directory contains `flags.json`, and logs are under `<KN_DATA_DIR>/logs`; no `config.toml` is created
