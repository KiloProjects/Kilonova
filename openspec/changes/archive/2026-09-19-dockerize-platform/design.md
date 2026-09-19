# Design

## Context

See proposal.md — Why. The constraints that shape the approach:

- **Flags are loaded before the database exists.** `cmd/kn/main.go` `Before` calls `config.LoadConfigV2(ctx, flagsPath, false)` and then `kilonova.SetDefaultLanguage(...)`; the pgx pool is only built later, in `initBase`. A DB-backed store forces that ordering to change.
- **`kn grader-serve` shares `Before` with `kn main` but has no database** and, verified by grep, imports nothing from `sudoapi/flags`. It must keep starting with no DB at all.
- **`config.SetOnFlagUpdate` takes a no-arg callback** and the current implementation rewrites the whole `flags.json` on every admin edit. Persisting one row needs the flag's name at the callback.
- **The platform shells out to `diff`** (`eval/checkers/diff.go`, used by `getAppropriateChecker` in the grader handler, which runs inside `kn main`). The platform image therefore cannot be `scratch` or distroless-static.
- **Languages are discovered by probing.** `scheduler/langmgr.go` and `manager.go` call `exec.LookPath` on each language's version command; a toolchain absent from the grader image just disappears from the language list, silently.
- **`isolate` is found via `exec.LookPath`** over `/usr/local/bin/isolate`, `/usr/local/etc/isolate_bin`, `isolate`. `box.InitKeeper` runs once lazily from `box.New` (`keeperOnce`), and its `KN_SANDBOX_ENSURE_CG_KEEPER` branch is an unimplemented `panic("TODO")` with `verifyKeeper`/`startKeeper` stubbed out in comments — this change fills it in (D5).
- **Upstream's `isolate-cg-keeper` exists only because a systemd service needs a process to hold its delegated cgroup open.** Its whole job (read from the source): refuse a cgroup v1/hybrid host, resolve its own cgroup from the `0::` line of `/proc/self/cgroup`, write that path into the file named by `cg_root = auto:<file>` (default `/run/isolate/cgroup`), `mkdir <cg>/daemon`, move its own pid into `<cg>/daemon/cgroup.procs`, write `+cpuset +memory` into `<cg>/cgroup.subtree_control`, then `pause()` forever.
- **Build order is load-bearing**: `go generate` produces `_translations.json` (embedded) and `web/assets/chroma.css` (an input to the Vite CSS bundle); `web/assets_test.go` fails if the manifests point at files that were not embedded.

## Goals / Non-Goals

**Goals:**

- A platform container that is disposable: no mutable state outside its data volume and Postgres, graceful SIGTERM, probeable health.
- A grader container that an operator can scale out by running more copies against the same platform.
- Zero behaviour change for the existing bare-metal deployment, which keeps its log file and keeps working through the flag migration without operator action.

**Non-Goals:**

- Making the platform fully stateless. Datastore buckets (tests, subtests, attachments, avatars) stay on a local volume; moving them to object storage is a separate change.
- Unprivileged grading. The grader runs `--privileged --cgroupns=host`; hardening it is out of scope.
- Reproducible pinned toolchain versions in the grader image. Apt and upstream releases decide; language versions are reported by probing, not asserted.

## Decisions

### D1 — Flags live in a `flags` table, one row per flag

`db/psql_schema/015.flags.sql` (new id, appended to the `Migrations` list in `db/migrations.go`; 014 is the current maximum):

```sql
CREATE TABLE flags (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT NOW()
);
```

Reads are one `SELECT key, value FROM flags` at boot; writes are one `INSERT ... ON CONFLICT (key) DO UPDATE` per admin edit. The value column is the same JSON the file held, so `sneakUpdate(json.RawMessage)` — the existing decode path — is reused unchanged and no flag definition moves.

*Alternatives considered.* One row holding the whole JSON blob: simpler, but two admins editing different flags in the same second clobber each other, and it makes the table useless for inspection. A `LISTEN/NOTIFY` fan-out so replicas see each other's edits: real value later, but there is exactly one platform process today — deferred, and the per-row schema does not block it.

### D2 — Loading moves to the composition root, not the CLI `Before`

`Before` keeps only what needs no database: `LoadConfigV2` is no longer called there, `applyFlagOverrides()` (the `KN_FLAG_OVERRIDES` loop, split out of `LoadConfigV2`) is. `initBase` gains, right after `postgres.RunMigrations`:

1. `config.LoadFlagsFromDB(ctx, pool)` — apply stored rows over the compiled-in defaults.
2. the one-time import (D3).
3. `applyFlagOverrides()` again, so a per-process override still beats a stored value.
4. `kilonova.SetDefaultLanguage(flags.DefaultLanguage.Value())`, moved out of `Before`.

Every DB-backed entry point (`kn main` and the `aitools` / `contest-utils` / `submission-saver` / `problem-diagnostics` subcommands) already funnels through `initBase`, so one call site covers them all. `kn grader-serve` never reaches `initBase` and therefore never touches a flag store — which is correct, since it reads no flags.

`SetOnFlagUpdate` changes signature to `func(name string)` and the callback becomes a single-row upsert with a short context timeout; failures are logged, never fatal, and never revert the in-memory value. `SaveConfigV2` is deleted.

### D3 — The legacy file is imported once, marked by a sentinel row

After loading, if the table holds no rows the platform reads `KN_FLAGS_PATH` with the existing `LoadConfigV2` reader, then persists every registered flag plus a sentinel row `_flags_file_imported`. Keys beginning with `_` are skipped by the loader, so the sentinel never collides with a real flag and never produces an "unknown key" warning. Presence of *any* row — sentinel included — means the import already happened, which also makes a fresh instance that never had a file import-clean after its first flag write.

*Alternative considered.* A `kn flags-import` subcommand an operator runs deliberately. Explicit, but it turns a container upgrade into a manual step for every existing deployment; the sentinel makes the upgrade a no-op.

### D4 — Lifecycle fixes are three small edits, not a framework

- `cmd/kn/kn.go`: `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` — `os.Kill` is dropped because SIGKILL cannot be caught, and its presence is what hid the missing SIGTERM. Shutdown uses `context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)`; today `server.Shutdown(ctx)` is handed the already-cancelled signal context, so it returns instantly and drains nothing.
- `cmd/kn/grader_serve.go`: `ListenAndServeTLS` moves into a goroutine, the command waits on a signal context and calls `Shutdown` with the same bounded drain.
- `/healthz` on both. The platform registers it on the chi router ahead of the session middleware; it reports unhealthy once a `draining` atomic is set at the top of shutdown, otherwise pings the pool with a 2s timeout. The grader wraps its mux so `/healthz` is registered on an outer mux and everything else goes through `registry.Auth`, and answers from the `scheduler.CheckCanRun` result captured at startup rather than re-probing isolate per request.

### D5 — `kn grader-serve` is its own cg-keeper

`box.InitKeeper` implements the keeper's logic directly instead of the container depending on systemd. The daemon half of upstream's program is unnecessary here: the keeper is a separate process only because a systemd unit needs *someone* to sit in the delegated cgroup, and `kn grader-serve` is already that long-lived process. So `InitKeeper` does the setup and returns; the grader itself keeps the cgroup alive for as long as it runs.

When `KN_SANDBOX_ENSURE_CG_KEEPER` is set, after resolving the isolate binary:

1. Refuse to continue unless `/sys/fs/cgroup/cgroup.subtree_control` exists and `/sys/fs/cgroup/unified` does not — cgroup v1 and hybrid layouts are out.
2. Read `cg_root` from isolate's config file (the single `cg_root = …` line; default `auto:/run/isolate/cgroup` when the file or key is absent). An explicit path is used as-is; an `auto:<file>` value means resolve and publish.
3. Resolve own cgroup: the `0::<path>` line of `/proc/self/cgroup`, joined onto `/sys/fs/cgroup`. This is correct in both cgroup namespace modes — with `--cgroupns=host` the line is the full host path and `/sys/fs/cgroup` is the host root; with a private namespace the line is `/` and `/sys/fs/cgroup` is already the container's own cgroup.
4. Write that path to `<file>` (creating its directory), so the `isolate` child process reads the same root.
5. `MkdirAll(<cg>/daemon)` — tolerating "already exists", unlike upstream, because `kn` restarting inside a still-live container cgroup is normal — write our own pid into `<cg>/daemon/cgroup.procs`, and write `+cpuset +memory` into `<cg>/cgroup.subtree_control`.

Step 5 is the part that looks odd and is not optional: cgroup v2 forbids a cgroup from both holding processes and enabling controllers for its children, so the grader has to park itself in a leaf before isolate can get `cpuset` and `memory` on its per-box groups. Writing the pid moves the whole thread group, which is what we want. `isolate` children inherit `daemon/` and then move each sandboxed process into its own box cgroup, so nothing else needs to know.

The grader image sets `KN_SANDBOX_ENSURE_CG_KEEPER=true`; the default stays false, so host deployments running the real `isolate.service` are untouched.

*Alternatives considered.* Running upstream's `isolate-cg-keeper` from the container entrypoint with a supervisor or a shell `&`: needs a second process and PID-1 reaping in every image, for a program whose entire purpose is to be a process we already have. Shipping systemd in the container: no.

### D6 — One builder stage, two runtime images

`Dockerfile` (platform) and `Dockerfile.grader` share a builder: `golang:1.27-bookworm` plus Node and pnpm, running the documented order — `go generate ./...`, `pnpm -C web/assets install --frozen-lockfile`, `pnpm -C web/assets build`, `go build ./cmd/kn`. `go.mod`/`go.sum` and `pnpm-lock.yaml` are copied before the rest so dependency layers cache. The templ generator comes from the Go tool deps already in `go.mod`, so nothing extra is installed for it.

Platform runtime: `debian:bookworm-slim` + `ca-certificates`, `diffutils`, `tzdata`, `curl` (for `HEALTHCHECK`). It runs as root like the grader: in an unprivileged container a dropped uid buys only the default capability set (raw sockets, `dac_override`, and friends) and no protection for the database, datastore or grader token the process legitimately holds, so it was not worth the data-directory ownership friction it imposes on anyone reusing an existing `KN_DATA_DIR`. Grader runtime: also `bookworm-slim`, running as root because isolate needs it, with `isolate` built from source in its own stage (`libcap-dev`, `libsystemd-dev`, `pkg-config`) and toolchains installed on top. Upstream's default config expects `subid_user = isolate` with a range in `/etc/subuid`/`/etc/subgid`, plus `box_root` and `lock_root` (`/run/isolate/locks`) present — the image creates the user and the ranges, or pins `first_uid`/`first_gid`/`num_boxes` instead. `isolate-cg-keeper` itself is not installed: D5 replaced it.

*Alternative considered.* Building the two images from separate contexts, or basing the grader on the platform image. Separate contexts duplicate the Go build; a shared base would drag `diff`-only baggage into the grader and the grader's toolchains are what dominate the size anyway.

### D7 — Kotlin and uv are downloaded; everything else is apt

`build-essential`, `mold`, `fp-compiler`, `golang`, `ghc`, `default-jdk`, `python3`, `nodejs`, `php-cli` and `rustc` come from Debian. `kotlinc` and `uv` have no usable Debian package, so they are fetched from their upstream release archives at pinned versions in the builder and copied in. `<KN_DATA_DIR>/uv` must be writable: the uv language mounts it into the sandbox as `/mnt/uv` for `XDG_*`, so the entrypoint creates it.

### D8 — `.dockerignore` is rewritten, `.docker/` is deleted

The current file only excludes generated frontend output, which means the build context today would carry `node_modules`, the 100 MB `./kn` binary, `.env`, `.git` and the local data directory. The rewrite excludes those as well; `.docker/data/.gitkeep` is a leftover of images deleted years ago and goes with them.

## Risks / Trade-offs

- **The Go keeper has to track a C program we do not control.** Upstream could change how `cg_root` is resolved or which controllers isolate needs. → The implementation stays a literal translation of `setup_cg()` with the source referenced in a comment, and the grader fails loudly at startup rather than degrading if any step fails. Isolate's own `isolate-check-environment` is the cross-check in task 2.8.
- **Parking the grader process in `<cg>/daemon` moves the whole process.** If the container runtime or a supervisor later expects `kn` in the cgroup it started it in, accounting attributed to the container changes shape. → Only the leaf changes, not the container's cgroup subtree, so runtime-level limits still apply; the grader logs the cgroup path it moved into.
- **A host running the real `isolate.service` must not run the Go keeper too** — both would try to own the same subtree. → `KN_SANDBOX_ENSURE_CG_KEEPER` defaults to false and only the grader image sets it; the docs state the two are mutually exclusive.
- **`--privileged --cgroupns=host` is close to host root.** → Documented as such: the grader is arbitrary code execution as a service, must sit on an isolated network with no host port published, and must hold no platform credentials — the property `grader-auth-config` already requires.
- **The flag migration is one-way.** A rollback to a pre-migration binary reads `flags.json`, which is frozen at its pre-upgrade content, so edits made after the upgrade are lost on rollback. → The import leaves the file untouched, so the rollback is to a working older state rather than to defaults; the window is short and the setting count is small.
- **Multi-GB grader image.** → Accepted per the toolchain decision; a layer per toolchain group keeps rebuilds incremental.
- **Language versions drift with the base image.** → Versions are probed and displayed, never asserted; a Debian bump can change a reported compiler version without any code change. Contest operators should pin an image tag.
- **The default `HEALTHCHECK` adds `curl` to the platform image.** → A few MB against an orchestrator being able to probe without extra config; acceptable.
- **`docker logs` replaces `<data>/logs/run.log` in containers.** → Called out in the docs page; bare-metal deployments are unaffected because `KN_LOG_FILE` defaults to true.

## Migration Plan

1. Ship the binary changes (D1–D4) first — they are independently deployable on bare metal. The first restart imports `flags.json` into Postgres; nothing else changes because `KN_LOG_FILE` defaults on and the host keeps its log file.
2. Verify on the existing deployment that admin flag edits persist across a restart, then delete `flags.json` at leisure.
3. Build and exercise the images (D5–D7) against a throwaway database.
4. **Rollback:** the previous binary still reads `flags.json`, which the import left byte-identical. Migration `015` is additive and can stay in place across a rollback.

### D9 — `KN_SANDBOX_ALLOW_INSECURE` also turns off grader TLS verification in remote mode

`kn grader-serve` requires `KN_GRADER_TLS_CERT`/`KN_GRADER_TLS_KEY`, so the compose stack has to present *some* certificate, and in a compose network that is a self-signed one the platform will reject. Rather than a CA-distribution dance for a local stack, the existing insecure switch gains the second meaning: in remote mode it sets `InsecureSkipVerify` on the transport the platform uses to reach the grader.

The two meanings never apply at once, which is what makes the reuse safe rather than a pun: `KN_EVAL_MODE=local` gives it the stupidbox meaning and the variable is dead in remote mode today; `KN_EVAL_MODE=remote` gives it the TLS meaning and the sandbox meaning is unreachable. It also keeps the production audit to one question — "is `KN_SANDBOX_ALLOW_INSECURE` false?" — instead of two.

Both planes must share the transport. `getRemoteRunner` builds two clients (`scratchClient` with its 60s timeout, and `http.DefaultClient` for the control plane); a single `*http.Transport` carrying the `tls.Config` goes into both, or the data plane silently keeps verifying while the control plane does not. TLS itself is still required — this disables verification, never the encryption, and never the bearer token.

Two guardrails: the platform logs a warning at startup naming the endpoint whenever it is on, and the grader image's entrypoint generates a self-signed certificate for its service name when none is mounted, so the compose stack needs no manual step. One asymmetry to keep in mind while implementing: in local mode the flag is permissive (stupidbox is used only if isolate is genuinely absent), while in remote mode it is unconditional.

*Alternatives considered.* A separate `KN_EVAL_REMOTE_INSECURE_TLS`: more precise, but a second production-audit item for a variable that would only ever be set together with contexts where the first already means "this eval path is not secure". Distributing the grader's self-signed cert to the platform as a CA: `SSL_CERT_FILE` replaces Go's system pool rather than adding to it (breaking outbound TLS to SMTP, Discord and OpenAI), and `update-ca-certificates` in the entrypoint needs root, which the platform image deliberately does not have.
