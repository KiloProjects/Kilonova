# config-migration Specification

## Purpose

Defines the `kn config-migrate` subcommand that converts legacy `config.toml` / `grader.toml` files into the environment contract and the flags file, so operators can move to env-based configuration without hand-translating keys.

## Requirements

### Requirement: `kn config-migrate` converts legacy TOML files to the environment contract
The `kn config-migrate` subcommand SHALL read a legacy `config.toml` (`-c/--config`, default `./config.toml`) and/or `grader.toml` (`--grader-config`, default `./grader.toml`), skipping any file that does not exist with a notice on stderr, and SHALL print to stdout one `KEY=value` line per non-default value in dotenv format (values quoted when they contain whitespace or shell-significant characters), using the variable names defined by the `env-config` capability. Every legacy key that maps to a variable SHALL be emitted; legacy keys with no mapping SHALL be reported on stderr and never silently dropped. The command SHALL NOT modify or delete the TOML files, and running it twice on the same input SHALL produce identical output.

#### Scenario: Platform config is converted
- **WHEN** `kn config-migrate -c config.toml` is run against a file containing `[common] db_dsn = "host=db user=kn"` and `[eval] num_concurrent = 4`
- **THEN** stdout contains `KN_DB_DSN="host=db user=kn"` and `KN_SANDBOX_NUM_CONCURRENT=4`

#### Scenario: Grader client registry is converted
- **WHEN** `grader.toml` contains two `[[grader.client]]` entries named `a` and `b` with tokens `t1` and `t2`
- **THEN** stdout contains `KN_GRADER_CLIENT_A=t1` and `KN_GRADER_CLIENT_B=t2`


#### Scenario: Disabled email with a host set
- **WHEN** `config.toml` has `[email] enabled = false` and `host = "smtp:587"`
- **THEN** no `KN_SMTP_*` line is emitted, so mail stays disabled

#### Scenario: Missing file is skipped
- **WHEN** only `grader.toml` exists and the command is run with defaults
- **THEN** stderr notes that `./config.toml` was not found, stdout contains only `KN_GRADER_*` and `KN_SANDBOX_*` lines, and the exit status is zero

#### Scenario: Source files are untouched
- **WHEN** the command completes
- **THEN** `config.toml` and `grader.toml` have identical contents and modification times to before the run

### Requirement: Admin-editable values are written to the flags file
For the legacy keys `common.default_language`, `common.test_max_mem_kb` and `frontend.banned_hot_problems`, the command SHALL write the corresponding flags into the flags file given by `-f/--flags` (default `./flags.json`, created if absent) instead of emitting environment lines, and SHALL report on stderr which flags were written.

#### Scenario: Flags are persisted
- **WHEN** `config.toml` has `default_language = "ro"` and `banned_hot_problems = [5]`
- **THEN** after the run `flags.json` contains `frontend.default_language: "ro"` and `frontend.banned_hot_problems: [5]`, and stdout contains no line for either key
