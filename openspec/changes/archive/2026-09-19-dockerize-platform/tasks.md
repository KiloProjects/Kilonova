# Tasks

## 1. Flag store in Postgres

- [x] 1.1 Add `db/psql_schema/015.flags.sql` creating the `flags` table (key/value/updated_at per design D1) and register id 015 in the `Migrations` list in `db/migrations.go`; verify by booting against a scratch database and seeing the table created exactly once.
- [x] 1.2 Add `db/flags.go` with `GetFlags(ctx) (map[string]json.RawMessage, error)` and `SetFlag(ctx, key string, value json.RawMessage) error` (upsert), following the existing raw-SQL style; verify with a unit test or a manual round trip through `psql`.
- [x] 1.3 In `domain/config/config_v2.go`, split the `KN_FLAG_OVERRIDES` loop out of `LoadConfigV2` into `ApplyFlagOverrides(ctx)`, add `LoadFlagsFromDB` applying stored rows through the existing `sneakUpdate` path (skipping `_`-prefixed keys, logging and skipping undecodable values and unknown keys without failing), and delete `SaveConfigV2`; verify `go test ./domain/config/...` passes with `config_v2_test.go` updated to the new surface.
- [x] 1.4 Change `SetOnFlagUpdate` to `func(name string)` and make `flag[T].Update` pass its own name; verify `go build ./...` is clean and the callback receives the edited flag's dotted name.
- [x] 1.5 Implement the one-time import in `domain/config`: when the table is empty, read `KN_FLAGS_PATH` via `LoadConfigV2`, persist every registered flag plus the `_flags_file_imported` sentinel, and log what was imported; verify a second boot with a modified stored value does not re-apply the file and leaves the file byte-identical (`md5` before/after).
- [x] 1.6 Rewire `cmd/kn/main.go` `Before` to call only `ApplyFlagOverrides`, and `cmd/kn/base.go` to run load → import → overrides → `SetDefaultLanguage` right after `RunMigrations`, registering the single-row upsert as the update callback; verify `kn main` boots, an admin flag edit survives a restart, and `kn grader-serve` still starts with no `KN_DB_DSN` set.

## 2. Process lifecycle and sandbox startup

- [x] 2.1 In `cmd/kn/kn.go`, replace `os.Kill` with `syscall.SIGTERM` in `signal.NotifyContext` and give `server.Shutdown` a `context.WithTimeout(context.WithoutCancel(ctx), 15s)`; verify `kill -TERM` on a running `kn main` exits 0 within the grace period while an in-flight request completes.
- [x] 2.2 Add `/healthz` to the platform router ahead of the session middleware: 200 when serving, non-2xx while draining or when a 2s pool ping fails; verify with `curl` against a running instance and again with the database stopped.
- [x] 2.3 In `cmd/kn/grader_serve.go`, move `ListenAndServeTLS` into a goroutine behind a signal context with the same bounded drain, and serve `/healthz` from an outer mux so it bypasses `registry.Auth` while every other path still requires a token; verify an unauthenticated `/healthz` returns 200 and an unauthenticated `/languages` still returns 401.
- [x] 2.4 Add the `KN_LOG_FILE` flag (bool, default true) in `cmd/kn/main.go` and gate the lumberjack handler and the `LogDir` `MkdirAll` on it in `cmd/kn/kn.go`; verify `KN_LOG_FILE=false` creates no `logs/` directory and still logs to stdout, and that the default still writes `run.log`.
- [x] 2.5 Document `KN_LOG_FILE` in `.env.example` and `docs/configuration.md`; verify `kn --help` lists it with its default.
- [x] 2.6 Implement the cgroup-v2 check and `cg_root` resolution in `eval/box/cgkeeper.go`: reject a host without `/sys/fs/cgroup/cgroup.subtree_control` or with `/sys/fs/cgroup/unified`, read the `cg_root` line from isolate's config file (default `auto:/run/isolate/cgroup`), and for `auto:` derive the path from the `0::` line of `/proc/self/cgroup`; verify with a unit test over fixture `/proc/self/cgroup` and config contents for the `auto:`, explicit-path and missing-config cases.
- [x] 2.7 Replace the `panic("TODO")` in `InitKeeper` with the setup itself (per design D5): publish the resolved path to the `auto:` file, `MkdirAll` the `daemon` leaf tolerating an existing one, write our pid to `<cg>/daemon/cgroup.procs`, enable `+cpuset +memory` in `<cg>/cgroup.subtree_control`, log the cgroup moved into, and return an error naming the failed step instead of continuing; verify on a cgroup-v2 Linux host that a second `kn grader-serve` start against the same cgroup succeeds.
- [x] 2.8 Verify the Go keeper against isolate's own checker: with `KN_SANDBOX_ENSURE_CG_KEEPER=true` and no keeper service running, `isolate-check-environment` reports no cgroup problems and an evaluation enforces both the time and the memory limit (a deliberate memory-limit-exceeding submission must report MLE, not a wrong answer).

- [x] 2.9 In `getRemoteRunner` (`eval/grader/grader.go`), build one `*http.Transport` carrying `tls.Config{InsecureSkipVerify: config.Eval.AllowInsecureSandbox}` and use it for both the control-plane client (today `http.DefaultClient`) and `scratchClient`, and log a startup warning naming the endpoint when verification is off; verify a self-signed grader is reachable with the flag set and rejected on both planes without it, and that an unauthenticated request still gets 401 either way.
- [x] 2.10 Update `.env.example` and `docs/configuration.md` for the mode-dependent meaning of `KN_SANDBOX_ALLOW_INSECURE` (stupidbox fallback in local mode, no grader certificate verification in remote mode; never in production either way); verify the `kn --help` usage string says both.

## 3. Images

- [x] 3.1 Rewrite `.dockerignore` to exclude `node_modules`, `.git`, `.idea`, the built `kn` binary, `.env`, generated assets and the local data directory, and delete the leftover `.docker/` directory; verify `docker build` context size drops to a few MB.
- [x] 3.2 Write the shared builder stage (Go + Node + pnpm) running generate → `pnpm install --frozen-lockfile` → `pnpm build` → `go build ./cmd/kn`, with dependency manifests copied first for layer caching; verify the stage builds from a clean checkout and `go test ./web/...` passes inside it (this is what catches a skipped asset build).
- [x] 3.3 Write `Dockerfile` for the platform: `bookworm-slim` runtime with `ca-certificates`, `diffutils`, `tzdata`, `curl`, `KN_LOG_FILE=false`, `EXPOSE 8070`, `HEALTHCHECK` on `/healthz`; verify the image runs against a Postgres container, serves the site, and grades nothing locally because it has no sandbox.
- [x] 3.4 Verify the platform image contains no `go`, `node` or `pnpm` and that a default-checker submission grades correctly through a remote grader (proves `diff` is present).
- [x] 3.5 Write `Dockerfile.grader`: an isolate build stage from source (without `isolate-cg-keeper`, which task 2.7 replaced), a runtime with every toolchain from design D7 (apt for most, pinned upstream archives for `kotlinc` and `uv`), the `isolate` user with its `/etc/subuid`/`/etc/subgid` ranges plus `box_root` and `/run/isolate/locks`, an entrypoint that creates `<KN_DATA_DIR>/uv` and generates a self-signed certificate for the service name when `KN_GRADER_TLS_CERT`/`KN_GRADER_TLS_KEY` point at files that do not exist, and `KN_SANDBOX_ENSURE_CG_KEEPER=true`; verify `kn grader-serve` starts in the image under `--privileged --cgroupns=host`.
- [x] 3.6 Verify the grader image reports every expected language with a probed version via `GET /languages`, and that a submission in each of C++, Python (both `python3` and `uv`), Java, Kotlin, Go, Rust, Haskell, Pascal, Node and PHP evaluates to the expected verdict.

## 4. Compose and integration

- [x] 4.1 Verify the in-process keeper end to end in the built image: with `--privileged --cgroupns=host`, run a real evaluation and confirm isolate enforces time and memory limits, then repeat with a private cgroup namespace to confirm the `/proc/self/cgroup` derivation holds in both modes; record the supported mode in the docs page.
- [x] 4.2 Write `compose.yaml` with Postgres (named volume, healthcheck), the platform (`KN_EVAL_MODE=remote`, named data volume, published port, `depends_on` healthy), and the grader (privileged, `cgroup: host`, no published port, own data volume, self-signed cert from its entrypoint with `KN_SANDBOX_ALLOW_INSECURE` set on the platform side), plus `.env.example` entries for the stack; verify `docker compose up` on a clean checkout brings the site up with migrations applied.
- [x] 4.3 Verify the stack end to end: submit a solution and get a verdict, restart the stack and confirm users, problems, uploaded tests and an edited runtime flag all survive, and confirm the grader port is unreachable from the host.
- [x] 4.4 Verify `docker compose stop` shuts the platform down gracefully — exit code 0, no 10-second SIGKILL wait in the logs.

## 5. Docs

- [x] 5.1 Write `docs/deployment.md` covering both images, the volumes, the environment for each service, the `--privileged --cgroupns=host` requirement and its security implications, the remote-grader topology and TLS/token setup, and the `docker logs` change; add it to the `mkdocs.yml` nav and verify `scripts/build_docs.sh` builds without warnings.
- [x] 5.2 Update `CLAUDE.md` and `docs/configuration.md` for the flag store (flags in Postgres, `flags.json` import-only) and the container workflow; verify no remaining doc claims the process writes `flags.json`.

## 6. Final verification

- [x] 6.1 Run the full local gate — `go generate ./...`, `pnpm -C web/assets build`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` — and confirm it is clean.
- [x] 6.2 Verify the pre-container upgrade path on a copy of a real deployment: start the new binary against an existing database with an existing `flags.json`, confirm the flags are imported, the admin UI shows them, an edit persists across a restart, and the file is unchanged on disk.
