# Spec Delta

## ADDED Requirements

### Requirement: Admin-editable instance settings are runtime flags
The default UI language, the per-test maximum memory cap and the banned hot-problem list SHALL be flags `frontend.default_language` (string, default `en`), `eval.test_max_mem_kb` (int, default `655360`) and `frontend.banned_hot_problems` (list of ints, default empty), persisted in the database-backed flag store defined by the `flag-store` capability and overridable per process through `KN_FLAG_OVERRIDES`. `POST /admin/updateConfig` SHALL update these flags and SHALL NOT accept `num_workers` or `global_max_mem`. The default language SHALL be validated at read time: a value other than `en` or `ro` SHALL behave as `en` with a logged warning, never a panic.

#### Scenario: Admin changes the default language
- **WHEN** an admin submits `default_lang=ro` to `/admin/updateConfig`
- **THEN** the flag store holds `frontend.default_language: "ro"`, and new visitors are served Romanian without a restart

#### Scenario: Admin bans hot problems
- **WHEN** an admin submits `banned_hot_pbs=[1,2]`
- **THEN** the flag store holds `frontend.banned_hot_problems: [1,2]`, the hot-problem view is refreshed, and the list survives a container replacement

#### Scenario: Invalid default language in the flag store
- **WHEN** the flag store holds `frontend.default_language: "xx"`
- **THEN** the platform starts, logs a warning, and serves English

#### Scenario: Sandbox capacity is not admin-editable
- **WHEN** an admin submits `num_workers=8` to `/admin/updateConfig`
- **THEN** the field is ignored and sandbox capacity remains what `KN_SANDBOX_NUM_CONCURRENT` set at boot

## MODIFIED Requirements

### Requirement: The process never reads or writes a platform config file
The platform SHALL NOT read `config.toml`, the grader SHALL NOT read `grader.toml`, and no command SHALL write either file. The platform SHALL write no configuration file at all: runtime flags live in the database-backed flag store, and the file named by `KN_FLAGS_PATH` SHALL be read at most once per database, for the one-time import defined by the `flag-store` capability, and never written. The `KN_CONF_PATH` and `KN_GRADER_CONF_PATH` variables and the `-c/--config` and `--grader-config` options of `kn main` / `kn grader-serve` SHALL be removed.

#### Scenario: Stale config.toml is ignored
- **WHEN** a `config.toml` with a different `db_dsn` exists in the working directory and `KN_DB_DSN` is set
- **THEN** the platform connects using `KN_DB_DSN` and the file is left untouched

#### Scenario: Only flags.json is written at boot
- **WHEN** `kn main` starts in an empty directory with the required variables set
- **THEN** no configuration file is written at all: the directory contains neither `config.toml` nor `flags.json` after startup

#### Scenario: An existing flags file is never rewritten
- **WHEN** a `flags.json` is present and an admin then changes a flag
- **THEN** the change is persisted to the flag store and the file on disk stays byte-identical

### Requirement: Startup-only settings are never flags
Settings that are read once at process start or that describe the host SHALL be `KN_*` environment variables, not flag-store entries: the web listen address (`KN_LISTEN`), the reverse-proxy client IP header (`KN_TRUE_IP_HEADER`), the Prometheus exporter address (`KN_PROMETHEUS_LISTEN`), migration and SQL debugging switches (`KN_DB_RUN_MIGRATIONS`, `KN_DB_LOG_SQL`, `KN_DB_COUNT_QUERIES`), the rotating-file log switch (`KN_LOG_FILE`, default enabled), the MaxMind database path (`KN_MAXMIND_DB`), the OpenTelemetry toggle (`KN_OTEL_ENABLED`), sandbox policy (`KN_SANDBOX_ENSURE_CG_KEEPER`, `KN_SANDBOX_ALLOW_INSECURE`), and the Discord and OpenAI credentials (`KN_DISCORD_TOKEN`, `KN_DISCORD_CLIENT_ID`, `KN_DISCORD_CLIENT_SECRET`, `KN_OPENAI_TOKEN`, `KN_OPENAI_MODEL`, `KN_OPENAI_VISION_MODEL`). An integration SHALL be enabled by the presence of its token. Secrets held in these variables SHALL NOT be readable from templates. `kn config-migrate` SHALL convert the corresponding retired keys found in an existing `flags.json`, and retired keys SHALL be ignored when that file is imported into the flag store.

#### Scenario: Retired flag keys are converted
- **WHEN** `kn config-migrate -f flags.json` runs against a file containing `server.listen.port: 8080` and `integrations.openai.token: "sk-x"`
- **THEN** stdout contains `KN_LISTEN=localhost:8080` and `KN_OPENAI_TOKEN=sk-x`

#### Scenario: Discord enabled by token
- **WHEN** `KN_DISCORD_TOKEN` is set
- **THEN** the Discord gateway session is opened at start, and when it is empty no Discord features are offered

#### Scenario: OpenAI key stays server-side
- **WHEN** the problem edit page is rendered with `KN_OPENAI_TOKEN` set
- **THEN** the LLM tools section is shown without the token value being available to the template

#### Scenario: Log file switch is an environment variable
- **WHEN** an operator sets `KN_LOG_FILE=false`
- **THEN** the process logs only to stdout, and the setting is not editable from the admin UI

## REMOVED Requirements

### Requirement: Admin-editable instance settings are flags in `flags.json`
**Reason**: The flag store moved from a JSON file the process rewrites to a PostgreSQL table, so that platform containers hold no mutable configuration state. Replaced by "Admin-editable instance settings are runtime flags" above, which keeps the same three flags and the same `/admin/updateConfig` behaviour.

**Migration**: None for operators — the first boot after the upgrade imports the existing `flags.json` into the database automatically, after which the file is inert and can be deleted.
