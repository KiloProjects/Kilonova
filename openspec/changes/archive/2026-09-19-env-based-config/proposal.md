## Why

Startup configuration is split across `config.toml` (platform), `grader.toml` (remote grader) and `flags.json` (runtime flags), with no principled boundary: `config.toml` mixes deployment facts (DB DSN, SMTP credentials, data/log dirs) with instance settings that the admin UI edits at runtime and writes back to the same TOML file. That makes containerised deployment awkward (secrets in a mounted TOML, a file the process rewrites on boot, `grader-serve` needing a `config.toml` it never reads) and blocks the longer-term multitenancy direction, where per-instance settings must be separable from per-host wiring.

## What Changes

- **BREAKING** `config.toml` and `grader.toml` are removed. Every startup-only, host-bound value (DB DSN, data/log dirs, SMTP, debug, host prefix, sandbox capacity, grader listen/TLS/scratch/clients, remote-grader endpoint/token) becomes a `KN_*` environment variable, declared as a CLI flag with an env source so `kn --help` documents them and a `.env` file (already loaded by godotenv) keeps local dev ergonomic.
- The values the admin UI edits at runtime (`default_language`, `test_max_mem_kb`, `banned_hot_problems`) move to `flags.json` as ordinary flags. The `/admin/updateConfig` endpoint keeps working against those flags; the `num_workers` / `global_max_mem` fields are dropped from it and from the admin form, since sandbox capacity is a host property and never took effect without a restart anyway.
- The process never writes `config.toml` again: `config.Save`, the compactify/spread round-trip and the boot-time rewrite are deleted.
- `kn grader-serve` no longer requires a platform `config.toml` to exist.
- New `kn config-migrate` subcommand reads an existing `config.toml` and/or `grader.toml` and emits the equivalent `KN_*=...` lines (dotenv format) on stdout, and writes the flag-bound values into the target `flags.json`. This is the only remaining TOML reader for these files.
- `config.example.toml` / `grader.example.toml` are replaced by `.env.example`; CLAUDE.md and the docs describe the env-based layout.

## Capabilities

### New Capabilities
- `env-config`: the `KN_*` environment contract for the platform and the grader: which variables exist, their defaults, how they are validated at boot, how `.env` and CLI flags interact, and what stays in `flags.json`.
- `config-migration`: the `kn config-migrate` tool that converts legacy `config.toml` / `grader.toml` into env lines plus `flags.json` entries.

### Modified Capabilities
- `grader-auth-config`: the "Role-split configuration" requirement currently mandates a grader config *file* and states that local mode keeps reading `config.toml`. It changes to: the grader and the platform read their role-specific settings from environment variables; the client registry is supplied as one `KN_GRADER_CLIENT_<NAME>` variable per client; local mode reads sandbox capacity from the same `KN_SANDBOX_*` variables the grader uses. Token/TLS/registry semantics are unchanged.

## Impact

- `domain/config`: `config.go` loses TOML decoding, `Save`/`Load`, `spread`/`compactify`, `FrontendConf`, `TestMaxMemKB`; `grader.go` loses `LoadGrader` and the TOML tags. Structs `Common`, `Eval`, `Email`, `Grader` stay as the in-process view, populated from env.
- `cmd/kn`: `main.go` declares the platform env flags and drops config loading/saving from `Before`; `grader_serve.go` declares grader env flags; new `config_migrate.go`.
- `sudoapi/config.go` + `api/api.go` `/admin/updateConfig`: rewritten to update flags; `web/templ/admin/admin.html` and `web/web.go` template funcs lose the two dropped fields.
- `sudoapi/flags`: three new flags (`frontend.default_language`, `eval.test_max_mem_kb`, `frontend.banned_hot_problems`).
- `kilonova.SetDefaultLanguage` stops panicking on an invalid value (it is now admin-editable at runtime) and falls back to `en` with a warning.
- Consumers of `config.Common.TestMaxMemKB` (`domain/archive/test`, `web/web.go`) and `config.Frontend.BannedHotProblems` (`sudoapi/admin.go`) switch to the flags.
- `github.com/BurntSushi/toml` remains a dependency only for `cmd/kn/config_migrate.go` and `scripts/toml_gen`.
- Operators: one-time migration per deployment via `kn config-migrate`; `runkn.sh` keeps working because `.env` is read by the process itself after `sudo`.
