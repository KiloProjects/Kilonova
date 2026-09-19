# Proposal

## Why

The `KN_*` environment contract landed, but nothing else about the deployment is container-shaped: there is no Dockerfile (the last ones were deleted years ago, leaving a stale `.dockerignore` and an empty `.docker/data`), the process rewrites `flags.json` on every admin edit, `docker stop` hard-kills the platform because SIGTERM is never handled, there is no health endpoint for an orchestrator to probe, and logs are duplicated into a rotating file under the data volume. Running Kilonova today means a tmux session, `sudo ./runkn.sh`, and host-installed toolchains — an operator who wants a second grader has to hand-build a machine.

## What Changes

- Two images, built from one repo: `kilonova` (platform: `kn main`, no compilers) and `kilonova-grader` (`kn grader-serve`: isolate plus every language toolchain the grader probes for). The grader image is deliberately fat — languages are auto-detected by probing binaries, so a missing compiler silently removes a language from the platform.
- One compose stack: Postgres + platform in `KN_EVAL_MODE=remote` + privileged grader. Remote eval is the only sound container topology, because a platform image without toolchains cannot sandbox in-process.
- **BREAKING** Runtime flags move from `flags.json` to a Postgres table. The platform loads them after the DB pool exists, persists a single row per flag update, and imports an existing `flags.json` once on first boot if the table is empty. `KN_FLAGS_PATH` becomes an import-only path; the process no longer writes any file outside the datastore.
- `box.InitKeeper`'s `panic("TODO")` becomes the real thing: the grader does isolate's cgroup setup itself (verify cgroup v2, resolve and publish `cg_root`, park itself in a leaf subgroup, enable `cpuset`/`memory`) instead of depending on the `isolate-cg-keeper` systemd service. Upstream needs a separate daemon only because a systemd unit needs some process to hold its delegated cgroup open — `kn grader-serve` is already that process. This is what makes the grader container need no init system.
- Container-fitness fixes in the binary: SIGTERM is handled (today the list is `os.Interrupt, os.Kill` — SIGKILL is uncatchable and SIGTERM is ignored), HTTP shutdown gets an uncancelled drain context instead of the already-cancelled one, `kn grader-serve` learns the same signal handling, and both processes expose an unauthenticated `/healthz`.
- `KN_SANDBOX_ALLOW_INSECURE`, which is dead in remote mode today, gains a second meaning there: skip verification of the grader's certificate, so a compose stack works with a self-signed cert. The two meanings are mutually exclusive by eval mode, TLS and the token stay mandatory, and the platform warns loudly at startup.
- Logging becomes stdout-first: the rotating-file sink is gated behind `KN_LOG_FILE` (default true, so bare-metal behaviour is unchanged) and the images set it to false.
- `docs/deployment.md` documents both images, the volumes, the privileged/`--cgroupns=host` requirement for isolate, the remote-grader topology and what to use instead of the self-signed-certificate shortcut in production; `.dockerignore` is rewritten and the dead `.docker/` directory is removed.

Explicit non-goals: object storage for the datastore (buckets stay on a volume), publishing images to a registry from CI, Kubernetes manifests, and running the grader unprivileged.

## Capabilities

### New Capabilities
- `container-images`: what the two images contain, how they are built (generate → assets → `go build` order), what they expect at runtime (volumes, privileges, cgroups), and the compose topologies they compose into.
- `flag-store`: where runtime flags live, how they are loaded and persisted, the one-time `flags.json` import, and what happens to flag reads in processes without a database.
- `process-lifecycle`: signal handling, graceful shutdown, health endpoints and log destinations for `kn main` and `kn grader-serve`.

### Modified Capabilities
- `grader-auth-config`: "Authenticated, single-direction transport" gains an opt-out from grader certificate verification, gated on `KN_SANDBOX_ALLOW_INSECURE` and applying to both planes at once. TLS and the bearer token stay mandatory.
- `env-config`: the requirement "Admin-editable instance settings are flags in `flags.json`" and the claim in "The process never reads or writes a platform config file" that "`flags.json` SHALL remain the only file the process writes" both change — flags are stored in Postgres, `flags.json` is read once for import and never written. Adds `KN_LOG_FILE` to the contract.

## Impact

- `domain/config`: `config_v2.go` gains a DB-backed load/save path; `LoadConfigV2` stays as the import reader, `SaveConfigV2` is deleted.
- `db`: new append-only migration `015.flags.sql` (id must not be reused) plus a small `db/flags.go` accessor.
- `eval/box/cgkeeper.go`: `InitKeeper` gains the cgroup setup that replaces `isolate-cg-keeper`.
- `cmd/kn`: `main.go` loses flag loading from `Before` (moves after the pool is built in `base.go`), `kn.go` gains SIGTERM/drain/health wiring and the `KN_LOG_FILE` gate, `grader_serve.go` gains signals and `/healthz` outside the auth middleware.
- New files: `Dockerfile`, `Dockerfile.grader`, `compose.yaml`, `docs/deployment.md`; rewritten `.dockerignore`; deleted `.docker/`.
- Operators: one boot per instance auto-imports `flags.json`, after which the file is inert; anyone tailing `<data>/logs/run.log` in a container must switch to `docker logs`.
- `sudoapi/flags` and every flag consumer are untouched — the registry API does not change.
