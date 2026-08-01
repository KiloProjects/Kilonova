## 1. Foundations (no behavior change)

- [x] 1.1 Extract `RunEvictionPolicy` LRU+TTL logic from `domain/datastore/local_bucket.go` into a `datastore`-free helper (`internal/fsevict.Sweep(fs, dir, maxSize, maxTTL)`); rewire `localBucket` to use it
- [x] 1.2 Add unit test asserting the helper deletes only entries past TTL / over size and never touches younger entries
- [x] 1.3 ~~Add `github.com/pkg/sftp`~~ — dropped: the data plane is plain HTTP on the grader's own server, no new dependency

## 2. RPC contract

- [x] 2.1 Define the `.proto` for `GraderService`: `RunBox3`, `RunMultibox3`, `Languages` (unary); messages mirror `Box3Request`/`Box3Response`/`RunConfig`/`RunStats`/`ScratchFile`. `Languages` ships only supported-name→version (platform rebuilds behavior from compiled-in `language.Langs`)
- [x] 2.2 Generate ConnectRPC code via the existing `buf` setup
- [x] 2.3 Map generated types ↔ `eval` structs (conversion helpers in `eval/scheduler/rpc_conv.go`), keeping `eval` types authoritative

## 3. Grader server (remote mode)

- [x] 3.1 Implement the ConnectRPC server wrapping the existing `BoxManager` (`RunBox3`/`RunMultibox3`) — no changes to `manager.go`/`box/` (`eval/scheduler/rpc_server.go`)
- [x] 3.2 Implement `Languages` from the grader-side `LanguageManager`
- [x] 3.3 Add bearer-token auth interceptor validating against the client registry; reject unregistered/missing tokens before any execution (TLS is applied at the serve/http.Server layer, group 5)
- [x] 3.4 Serve the grader's scratch dir over an HTTP `PUT`/`GET`/`DELETE /scratch/{id}` endpoint on the same TLS+token server, ids UUID-validated against path traversal (`scheduler.ScratchHandler`); documented in `deploy-remote-grader.md`
- [x] 3.5 Scratch GC sweep helper on the grader using `fsevict.Sweep` with an orphan TTL ≫ max eval duration (`scratch.PeriodicSweep`; started by the serve wiring in group 5)

## 4. Platform client (remote mode)

- [x] 4.1 Implement `eval.Box3Scheduler` client stub over ConnectRPC; `Close` is a no-op (`GraderClient`)
- [x] 4.2 Implement `eval.LanguageManager` client stub with pull-once cache + manual resync (`NewRemoteLanguageManager` + `Resync`; rebuilds behavior from compiled-in `language.Langs`)
- [x] 4.3 Implement remote `eval.Scratch` over HTTP `PUT`/`GET`/`DELETE` against the grader's `/scratch` endpoint (`scratch.NewHTTP`)
- [x] 4.4 `http.Client` handles connection pooling/keep-alive; the scratch client carries a 60s `Timeout` so a dropped transfer fails fast; `SaveFile` (PUT) returns only after the server flushes+`Close`s the file
- [x] 4.5 Self-check: SaveFile→RunBox3→ReadFile→DeleteFile identifier round-trip through the proto conversion layer + RunConfig loss-less guard (`rpc_roundtrip_test.go`); plus a real HTTP scratch round-trip, path-traversal, and bad-token self-check (`scratch_server_test.go`)

## 5. Configuration & wiring

- [x] 5.1 Added `mode = local|remote` to `[eval]` plus `[eval.remote]` (`endpoint`, `token` — RPC and `/scratch` share the endpoint); `IsRemote()` helper (replaces the commented `// Address`)
- [x] 5.2 Defined the `grader.toml` schema + loader (`config.LoadGrader`): `[grader]` (listen, cert/key, scratch_dir, execution settings, scratch_ttl_sec) and `[[grader.client]]` (name, token, reserved `priority`)
- [x] 5.3 Branched at `eval/grader/grader.go`: `getLocalRunner` (in-process, unchanged) vs `getRemoteRunner` (RPC client + HTTP scratch + remote lang manager); added `grader-serve` command as the remote entrypoint
- [x] 5.4 Renamed `Box3Adapter` → `Box2Wrapper` (file `box2_wrapper.go`, ctor `NewBox2Wrapper`)

## 6. Validation

- [~] 6.1 Local mode unchanged: the local path is behavior-preserving by construction (`getLocalRunner` is the old body; same `NewLanguageManager(runner)`, `Box2Wrapper` == old `Box3Adapter`). Full compile/execute/communication regression still needs a live isolate+DB run.
- [ ] 6.2 Remote mode on one grader: end-to-end compile, execute, and multibox — **needs a deployed grader** (isolate + TLS); cannot run in this sandbox
- [~] 6.3 Orphan-GC invariant (deletes past-TTL, never a younger/live file) is covered by `internal/fsevict` unit tests; full crash-orphan reclaim on a live grader disk still needs a deployment run
- [x] 6.4 Auth verified end-to-end over httptest: valid token served, missing/invalid token rejected with `Unauthenticated` **before any box executes** (`rpc_auth_test.go`)
- [~] 6.5 Scratch drop mid-eval fails fast: the scratch `http.Client` 60s `Timeout` turns a dropped transfer into a Go error (no hang); the not-found/error status paths are unit-covered in `scratch_server_test.go`, but a real mid-transfer drop under load still wants a deployment run
