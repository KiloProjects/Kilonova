## Context

Kilonova grades submissions by running untrusted user code in sandboxes (`isolate`, behind the `eval.Sandbox` interface). Today the grader is in-process with the platform, so a sandbox escape immediately reaches the DB DSN, the datastore, and all config secrets.

Prior refactors left a clean seam:

```
   ┌─ PLATFORM SIDE (store-aware) ──────────────────────────────────┐
   │  grader.go · feeder.go · tasks/{compile,execute}               │
   │      build Box2Request{ BucketFile, ByteFile }  ← knows datastore│
   │            │ RunBox2()                                          │
   │            ▼                                                    │
   │  Box3Adapter          holds *datastore.Manager                 │
   │      BucketFile ──▶ scratch.SaveFile ──▶ identifier            │
   │            │ RunBox3(Box3Request{ identifiers, cmd, cfg })      │
   └────────────┼───────────────────────────────────────────────────┘
            ════╪════  ← THE WIRE GOES HERE (Box3Scheduler iface) ═══
   ┌────────────▼─────── GRADER SIDE (store-FREE) ──────────────────┐
   │  BoxManager   depends only on eval.Scratch                     │
   │      copyScratchFile ──▶ isolate box ──▶ run ──▶ SaveFile      │
   │  makeGoodCommand (LookPath/EvalSymlinks on grader's disk)      │
   │  box/isolatebox · cgkeeper                                     │
   └────────────────────────────────────────────────────────────────┘
```

Verified constraints from the code:
- `Box3Request.InputFiles` are `[]ScratchFile{Identifier, BoxPath, Mode}`; `Box3Response.Files` is `map[path]identifier`. **No bytes cross the `Box3Scheduler` interface.**
- Everything below `Box3Scheduler` imports no `datastore`/`db`/`sudoapi` (grep-confirmed).
- `eval.Scratch` is three methods (`SaveFile`/`ReadFile`/`DeleteFile`) already backed by an `afero.Fs` (`grader.go:623`).
- `LanguageManager` runs on the grader (`feeder.go:167`) because it inspects installed compilers; the platform needs the list to build requests and render UI.
- `BoxManager.Close()` drains all boxes and closes `availableIDs` — a grader-lifecycle op.
- `domain/datastore/local_bucket.go:229` `RunEvictionPolicy` is already LRU-by-modtime + TTL.

## Goals / Non-Goals

**Goals:**
- Run the grader as a separate process (one remote server) holding no platform credentials.
- Keep all connections platform→grader; the grader only ever answers.
- Make `adapter`, `tasks/`, `BoxManager`, `box/` change by ~zero lines — swap concrete impls for client stubs behind existing interfaces.
- Keep the RPC surface **unary** (identifiers only); move bytes over a separate HTTP `/scratch` endpoint so no streaming RPC code is written.
- Preserve today's single-machine dev experience under `mode = local`.
- Split config by role and seed (not build) future priority queuing.

**Non-Goals:**
- Grader pools / horizontal fan-out, and the per-grader language-set divergence that implies.
- Scratch-over-RPC streaming transport.
- Content-addressed / deduplicating scratch and its LRU-as-lifetime GC.
- Building the priority queue (only the config identity that would feed it).

## Decisions

### D1. The wire boundary is the `Box3Scheduler` + `LanguageManager` interfaces, exposed as unary ConnectRPC
The platform holds client stubs implementing `eval.Box3Scheduler` and `eval.LanguageManager`; the grader hosts a ConnectRPC service wrapping the real `BoxManager` and language manager. RPC messages carry `Box3Request`/`Box3Response`/`RunConfig`/`RunStats`/`ScratchFile` and the language map — all small, all unary.

**Why not stream files through the RPC?** ConnectRPC streaming forces hand-rolled chunking (the thing we're avoiding). A separate plain-HTTP endpoint (D2) avoids both that *and* SFTP, and buys nothing to defer.

### D2. Data plane is a remote `eval.Scratch` over a plain HTTP `/scratch` endpoint

> **Superseded SFTP.** The original plan moved bytes over the grader's `sshd` via `github.com/pkg/sftp` (chrooted sftp subsystem, SSH-key auth, a redialing SSH connection pool). That was rejected once we noticed the "no streaming code" argument it rested on applies equally to a plain `net/http` handler — `io.Copy(w, file)` / `io.Copy(file, r.Body)` streams with zero chunking, and the http stack pools connections for free. SFTP's advanced features (seek, resume, dir ops) are all unused: `eval.Scratch` is just whole-file put/get/delete. So SFTP bought a second listening service, a second auth system (SSH keys + host-key management), a new dependency, and ~250 lines of pool code — for nothing the HTTP endpoint doesn't do in ~60. The comparison against sshfs/FUSE that justified SFTP never considered "an endpoint on the server we already run."

`eval.Scratch` is three methods (`SaveFile`/`ReadFile`/`DeleteFile`). The remote impl is an `http.Client` speaking `PUT`/`GET`/`DELETE /scratch/{id}` against the grader's existing TLS server; the handler `io.Copy`s to/from a file in the grader's scratch dir. The grader owns the scratch directory locally and serves it on the **same** `listen` endpoint as the RPC — one port, one cert, one bearer token.

**Path traversal** is the one thing the transport swap moves onto us (SFTP had chroot). Ids are platform-minted UUIDv7; the grader `uuid.Parse`s the `{id}` and rejects anything else, so a client can only ever name a flat file in the scratch dir. No slashes, no `..`. One line.

**Ordering guarantee:** the `PUT` handler `Close()`s the file before writing its `204`, so the platform's `SaveFile` returns only after the bytes are on the grader's disk — same durability barrier the SFTP `Close()` gave, before any `RunBox3` references the id.

**Concurrency:** `http.Client`'s default transport pools and reuses keep-alive connections across parallel evals — no hand-rolled pool.

### D3. Role-split configuration; execution config moves to the grader
`[eval]`'s `num_concurrent` / `global_max_mem_kb` / `starting_box` are box-runner settings and move to the grader's config in remote mode. The platform keeps only "how to reach the grader."

```
   config.toml (LOCAL, unchanged)           grader.toml (REMOTE, grader's own file)
   [common][email][frontend]                [grader]  listen, cert/key, scratch_dir,
   [eval] num_concurrent,                             num_concurrent, global_max_mem, box#
          global_max_mem_kb, starting_box    [[grader.client]] name, token
                                                       # priority = "…"  (future field)
   config.toml (PLATFORM, REMOTE)            [[grader.client]] name, token
   [eval] mode = "remote"
          endpoint, token   # RPC + /scratch both live on endpoint
```

The commented-out `// Address string` already in `EvalConf` (config.go:43) anticipated this. Per-client blocks (name+token) are needed for auth and observability **now**; `priority` is a documented-but-unconsumed field so the future queue is a one-field add, not a schema refactor.

### D4. Authentication: one grader-minted bearer token over TLS
TLS authenticates the grader to the platform (server cert). A per-platform-instance bearer token (minted on the grader, stored on the platform) authenticates the platform to the grader — carried on the ConnectRPC interceptor **and** the `/scratch` requests, since both hit the same endpoint. One secret, one auth system. (The SFTP data plane would have needed a second one — SSH keys + host-key pinning; folding data onto HTTP deleted it.) All platform→grader.

**Why token, not mTLS (day one):** token + TLS is the least machinery that fits "multiple platform instances, one grader, grader authoritative." mTLS (per-platform cryptographic identity, cert-based revocation) is the upgrade path when audit/revocation needs it, at the cost of running a CA.

**Load-bearing caveat:** `RunBox3` is literally "run arbitrary code as a service." A leaked token = RCE on the grader. The token is necessary but not sufficient — the grader MUST also be network-segmented / IP-allowlisted to the platform. Token + TLS + segmentation, all three.

### D5. `Close()` becomes a client-side no-op over RPC
With multiple platform instances sharing one grader, a platform shutdown must not call `BoxManager.Close()` (which drains boxes and closes `availableIDs`) over the wire. The grader's lifecycle is independent of any platform's; the remote `Close` is a no-op (or at most a per-session drain).

### D6. Scratch GC is a janitor, never a participant in live-object lifetime
Extract `RunEvictionPolicy`'s LRU+TTL logic (local_bucket.go) into a `datastore`-free `evict(fs, maxSize, maxTTL)` helper. Both `localBucket` and `eval/scratch` use it. The grader runs the sweep on its own disk.

The MVP is **race-free by construction**: identifiers are UUIDv7, ownership is explicit (`adapter` `defer clearInput` / `readDeleteScratch`), and a live blob's lifetime is one eval (seconds–minutes) while the orphan TTL is hours. The sweep can therefore only ever catch crash-orphans, never a live blob.

**Deferral rationale — recorded verbatim so future-me doesn't reintroduce it:**

Content-addressed scratch (sha256 identifiers, dedup, skip-upload-if-present) is deferred not because hashing is hard (TeeReader + rename-at-close is trivial and the LRU already exists as `RunEvictionPolicy`), but because it **drags GC into live-object lifetime** and introduces a TOCTOU:

```
   platform:  stat(h) → present! ─────────skip upload──────────▶ RunBox3(h)
                        │                                            │
                        │   ┌── GC sweep: h looks cold by modtime ──┐│
                        └───┤   unlink(h)                           ├┘
                            └────────────────────────────────────────┘
                                 ↑ blob gone before the box reads it
```

POSIX `unlink` protects an already-open read (inode survives the fd), so this passes naive tests and fails intermittently under load on hot problems — the "frustrating-to-find" profile. Content-addressed modtime-LRU makes it *likely*: a blob written once but reused constantly looks coldest exactly when it's hottest. A correct fix needs a **lease/pin protocol** (pin on the skip-upload decision, honored by the sweep, released by `RunBox3`) — distributed refcounting across the boundary. That's the real cost.

**Rule that explains the whole split:** *keep GC out of live-object lifetime.* UUIDv7 + explicit delete does; content-addressing turns the janitor into a correctness-critical participant that needs leases.

## Risks / Trade-offs

- **[Leaked token → grader RCE]** → TLS + network segmentation / IP allowlist are mandatory alongside the token, not optional. Tokens are per-instance and revocable via the grader's client registry.
- **[HTTP scratch drop mid-eval]** → the platform's scratch `http.Client` carries a 60s `Timeout`, so a dropped transfer becomes a Go error and the eval fails fast and retries. No hung syscalls (the FUSE failure mode we rejected). The RPC client keeps no global timeout — `RunBox3` is legitimately long — so hang-protection there stays server-side (mem/time quotas).
- **[Path traversal via a crafted scratch id]** → the grader `uuid.Parse`s `{id}` and rejects non-UUIDs, so a client can only name a flat file in the scratch dir. Covered by `TestHTTPScratchRejectsPathTraversal`.
- **[Bandwidth: every submission re-uploads the same test inputs]** → accepted for the MVP; content-addressed caching (deferred, see D6) is the eventual fix.
- **[Config drift between platform's grader endpoint and grader's client registry]** → surfaced by a clear auth-failure error; `openspec`/ops docs to describe token provisioning (grader mints, platform stores).

## Migration Plan

1. Land the `.proto` + generated code and the shared `evict()` helper extraction with **no behavior change** (local mode still uses in-process everything).
2. Add the client stubs + grader RPC server; keep `mode = local` as default so existing deployments are untouched.
3. Stand up one remote grader with its own `grader.toml`, TLS, a client token, and `sshd` sftp subsystem chrooted to the scratch dir, on a segmented network.
4. Flip a staging platform to `mode = remote`; validate compile/execute/communication (multibox) end to end, then the orphan GC sweep.
5. **Rollback:** flip `mode` back to `local`; no schema/data migration is involved, so rollback is a config change.

## Open Questions

- Exact SFTP connection-pool size and redial/backoff policy — tune against real concurrency (`num_concurrent`).
- Max RPC message size ceiling (unary, so small, but pin it for large `RunConfig.Directories` / long command vectors).
- Whether language-metadata resync is purely manual (admin button) or also time-bounded (cache TTL) — leaning manual + on-connect fetch.
- Whether the grader's `sshd` is the system daemon (chrooted user) or an embedded Go SSH server for tighter scoping — leaning system `sshd` for the MVP.
