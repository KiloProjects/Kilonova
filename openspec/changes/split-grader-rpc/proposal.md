## Why

The grader runs arbitrary, untrusted user code inside sandboxes (`Box3` / isolate). A sandbox-escape vulnerability today lands the attacker in the same process and on the same host as the platform — with reach to the Postgres DSN, the datastore, and every secret in `config.toml`. We want the grader to run as a separate process (initially on one remote server) so that an escape is contained to a machine that holds **no** platform credentials and can only ever answer requests, never initiate them.

The recent refactors already did the hard part: `Box3Request`/`Box3Response` carry only scratch identifiers (never bytes), and everything below the `Box3Scheduler` interface (`BoxManager`, `box/`, `makeGoodCommand`) is already free of `datastore`/`db`/`sudoapi`. The seam exists; this change turns it into a network boundary.

## What Changes

- Introduce a **remote grader mode**: the box execution layer (`Box3Scheduler`) and the language inventory (`LanguageManager`) are exposed from the grader over **ConnectRPC** (unary only), and the platform talks to them through client stubs implementing the existing `eval.Box3Scheduler` / `eval.LanguageManager` interfaces.
- Introduce a **remote scratch** (`eval.Scratch`) implemented over **SFTP** (`github.com/pkg/sftp`) against the grader's `sshd`. The data plane (file bytes) rides SFTP; the control plane (RPC) stays byte-free. No FUSE / sshfs mount.
- Split configuration by role: the grader gets its **own config file** in remote mode holding box-execution settings, its scratch/listen settings, and a **per-client registry** (name + token) for the platform instances allowed to connect. The platform config gains a `mode = local|remote` switch plus the grader endpoint, token, and SFTP settings.
- Add a **`mode` switch** at the existing wiring point (`eval/grader/grader.go`): `local` keeps today's in-process `BoxManager` + local scratch (dev-friendly, backward-compatible, no new deps at runtime); `remote` selects the client stubs.
- Authenticate every RPC with a **grader-minted bearer token over TLS**, presented via a ConnectRPC interceptor; SFTP authenticates via SSH key. All connections are platform→grader.
- Change **`Close()` semantics**: over RPC, `Close` is a client-side no-op — one platform instance disconnecting MUST NOT tear down a grader shared by others.
- Add a **grader-side scratch GC sweep** (orphan janitor) reusing the eviction policy from `domain/datastore`, extracted into a `datastore`-free helper so the grader stays store-free.
- Rename `Box3Adapter` → a name reflecting its lasting role as the platform-side convenience wrapper (Box2 request ergonomics over the Box3 scheduler).

## Capabilities

### New Capabilities
- `grader-transport`: The RPC service boundary between platform and grader — the `Box3Scheduler` and `LanguageManager` surfaces exposed as ConnectRPC, the `local`/`remote` mode switch, `Close` semantics, and language-metadata pull/cache/resync.
- `grader-scratch-transport`: The remote `eval.Scratch` data plane over SFTP, its ordering guarantees, connection pooling/redial, and the scratch GC janitor (with the content-addressing deferral rationale recorded).
- `grader-auth-config`: The role-split configuration, the per-client token registry on the grader, TLS + bearer-token authentication, and the network-trust-direction requirements.

### Modified Capabilities
<!-- None: no existing specs in openspec/specs/. -->

## Impact

- **New dependency**: `github.com/pkg/sftp` (platform side). `afero` and `golang.org/x/crypto/ssh` already present. ConnectRPC codegen via the existing `buf` setup.
- **Code**: new `.proto` + generated service; grader-side RPC server wrapping `BoxManager` + scratch + langmgr; platform-side client stubs (`eval.Box3Scheduler`, `eval.LanguageManager`, `eval.Scratch` over SFTP); a `mode` branch at `eval/grader/grader.go` (~L603–632); extraction of eviction policy out of `domain/datastore/local_bucket.go` into a shared, `datastore`-free helper used by `eval/scratch`.
- **Config**: `[eval]` execution fields (`num_concurrent`, `global_max_mem_kb`, `starting_box`) move to the grader config in remote mode; platform config gains `[eval] mode` + remote endpoint/token/SFTP block; new `grader.toml` schema with `[[grader.client]]` blocks.
- **Unchanged**: `eval/scheduler/manager.go` (`BoxManager`), `eval/box/`, `eval/tasks/`, and the adapter's conversion logic — they already speak the interfaces being made remote. Local mode is byte-for-byte the current behavior.
- **Deferred (explicitly out of scope)**: scratch-over-RPC streaming, content-addressed caching, grader pools with per-grader language sets, priority queuing (config shape is seeded but the queue is not built).
