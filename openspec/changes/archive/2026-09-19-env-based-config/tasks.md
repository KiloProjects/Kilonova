## 1. Flags for admin-editable settings

- [x] 1.1 Add `DefaultLanguage` (`frontend.default_language`, "en"), `TestMaxMemKB` (`eval.test_max_mem_kb`, 655360) and `BannedHotProblems` (`frontend.banned_hot_problems`, `[]int{}`) to `sudoapi/flags`.
- [x] 1.2 Make `kilonova.SetDefaultLanguage` fall back to `en` with a warning instead of panicking on an unknown value.
- [x] 1.3 Add a `domain/config` test that round-trips a `[]int` flag through `SaveConfigV2`/`LoadConfigV2` and that `KN_FLAG_OVERRIDES` can set it.
- [x] 1.4 Switch consumers to the flags: `domain/archive/test/properties_file.go` (`TestMaxMemKB`), `sudoapi/admin.go` (`BannedHotProblems`), `web/web.go` template funcs (`testMaxMemMB`, `bannedHotProblems`; delete `globalMaxMem`, `numWorkers`).
- [x] 1.5 Rewrite `sudoapi.UpdateConfig` to update the three flags (dropping `num_workers`/`global_max_mem`, deleting the `config.Save` call and the `s.cmd` lookup); remove the two capacity inputs from `web/templ/admin/admin.html` and its submit script.

## 2. Environment contract

- [x] 2.1 Strip `domain/config/config.go` to the structs: delete `configStruct`, `spread`, `compactify`, `Save`, `Load`, the `toml` import and tags, `FrontendConf`, `CommonConf.{Debug,HostPrefix,HostURL,DefaultLang,TestMaxMemKB}`; replace `EmailConf.Enabled` with a `Host != ""` method and update its two callers.
- [x] 2.2 Strip `domain/config/grader.go` to a `GraderConf` struct with `Listen`, `CertFile`, `KeyFile`, `ScratchDir`, `ScratchTTLSec`, `Clients string`; delete `LoadGrader` and `GraderClientConf`.
- [x] 2.3 In `cmd/kn/main.go` declare the platform `KN_*` variables from design D2 as `cli` flags with `Sources: cli.EnvVars(...)` and `Destination` into `config.Common` / `config.Email` / `config.Eval` (plus local `debug` / `hostPrefix` vars); keep `--flags`; delete `--config`; the root `Before` now only loads flags, calls `kilonova.SetDebugMode/SetHostPrefix/SetDefaultLanguage`, installs the `SetOnFlagUpdate` hook (which also re-syncs `SetDefaultLanguage`), and saves flags once.
- [x] 2.4 Add `KN_EVAL_MODE` validation (`local|remote`) in `Before`; make `getRemoteRunner` return an error naming `KN_EVAL_REMOTE_ENDPOINT` / `KN_EVAL_REMOTE_TOKEN` when empty; change the `InitializeBaseAPI` data-dir error to name `KN_DATA_DIR`.
- [x] 2.5 In `cmd/kn/grader_serve.go` declare the `KN_GRADER_*` flags with `Destination` into a `config.GraderConf`, delete `--grader-config`, build the registry from `KN_GRADER_CLIENT_<NAME>` variables in `os.Environ()` (error naming the prefix when none are set), and fail fast when cert or key is empty.
- [x] 2.6 Confirm `postgres.NewDB` passes an empty DSN through to `pgxpool.ParseConfig` so libpq `PG*` variables work; add a comment pointing at that behaviour.
- [x] 2.7 `go build ./... && go vet ./... && golangci-lint run` clean; `go test ./...` green.

## 3. Migration tool

- [x] 3.1 Add `cmd/kn/config_migrate.go` with private TOML structs mirroring the legacy `config.toml` and `grader.toml`, flags `-c/--config` (default `./config.toml`), `--grader-config` (default `./grader.toml`), inheriting `-f/--flags`; register it in `main.go` `Commands`.
- [x] 3.2 Emit dotenv lines to stdout per design D2 (quote values with whitespace/`#`/`"`), map `[email] enabled=false` to no `KN_SMTP_*` lines, emit one `KN_GRADER_CLIENT_<NAME>` line per `[[grader.client]]`, skip missing files with a stderr notice, report unmapped keys on stderr.
- [x] 3.3 Write `default_language`, `test_max_mem_kb`, `banned_hot_problems` into the flags file via `flags.X.Update(...)` and report them on stderr.
- [x] 3.4 Add a test in `cmd/kn` that runs the conversion on fixture copies of `config.example.toml` and `grader.example.toml` and asserts the expected lines, the flags-file contents, and that the inputs are byte-identical afterwards.

## 4. Docs and cleanup

- [x] 4.1 Add `.env.example` documenting every variable with defaults and the `sudo`/`.env` note; delete `config.example.toml` and `grader.example.toml`; update `.gitignore` (drop the `!config.example.toml` / `!grader.example.toml` exceptions).
- [x] 4.2 Update `CLAUDE.md` Commands/Architecture text (config files → env variables, `kn config-migrate`, `.env`), and add a `docs/CHANGELOG.md` entry marked breaking with the migration steps from design.
- [x] 4.3 Remove `README`/docs mentions of `config.toml`, `grader.toml`, `KN_CONF_PATH`, `KN_GRADER_CONF_PATH`; verify `runkn.sh` needs no change.
- [x] 4.4 Smoke test: run `kn config-migrate` against the local `config.toml`/`grader.toml`, source the output into `.env`, start `./kn main` in local mode and `./kn grader-serve`, and confirm both boot without the TOML files present.
