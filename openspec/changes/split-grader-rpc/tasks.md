## 1. Foundations (no behavior change)

- [ ] 1.1 Extract `RunEvictionPolicy` LRU+TTL logic from `domain/datastore/local_bucket.go` into a `datastore`-free helper `evict(fs afero.Fs, maxSize int64, maxTTL time.Duration)`; rewire `localBucket` to use it
- [ ] 1.2 Add unit test asserting the helper deletes only entries past TTL / over size and never touches younger entries
- [ ] 1.3 Add `github.com/pkg/sftp` to `go.mod`

## 2. RPC contract

- [ ] 2.1 Define the `.proto` for `GraderService`: `RunBox3`, `RunMultibox3`, `Languages` (unary); messages mirror `Box3Request`/`Box3Response`/`RunConfig`/`RunStats`/`ScratchFile` and the language map
- [ ] 2.2 Generate ConnectRPC code via the existing `buf` setup
- [ ] 2.3 Map generated types ↔ `eval` structs (conversion helpers), keeping `eval` types authoritative

## 3. Grader server (remote mode)

- [ ] 3.1 Implement the ConnectRPC server wrapping the existing `BoxManager` (`RunBox3`/`RunMultibox3`) — no changes to `manager.go`/`box/`
- [ ] 3.2 Implement `Languages` from the grader-side `LanguageManager`
- [ ] 3.3 Add TLS + bearer-token auth interceptor validating against the client registry; reject unregistered/missing tokens before any execution
- [ ] 3.4 Configure `sshd` sftp subsystem chrooted to the scratch dir (ops/docs), and point the grader's local scratch at that dir
- [ ] 3.5 Wire the scratch GC sweep on the grader using the `evict()` helper with an orphan TTL ≫ max eval duration

## 4. Platform client (remote mode)

- [ ] 4.1 Implement `eval.Box3Scheduler` client stub over ConnectRPC; `Close` is a no-op
- [ ] 4.2 Implement `eval.LanguageManager` client stub with pull-once cache + manual resync action
- [ ] 4.3 Implement remote `eval.Scratch` over SFTP (`sftpfs` behind `afero.NewBasePathFs`, or a direct 3-method impl on `*sftp.Client`)
- [ ] 4.4 Add SFTP connection pool with context deadlines and redial; ensure `SaveFile` waits for `Close` before returning
- [ ] 4.5 Add a self-check (assert-based `demo`/test) covering the SaveFile→RunBox3→ReadFile→DeleteFile identifier round-trip against a stub scratch

## 5. Configuration & wiring

- [ ] 5.1 Add `mode = local|remote` to the platform `[eval]` config plus remote `endpoint`, `token`, and SFTP block; replace the commented `// Address` field
- [ ] 5.2 Define the `grader.toml` schema: `[grader]` (listen, cert/key, scratch_dir, num_concurrent, global_max_mem_kb, starting_box) and `[[grader.client]]` (name, token, reserved `priority`)
- [ ] 5.3 Branch at `eval/grader/grader.go` (~L603–632): `local` → in-process `BoxManager` + local scratch (unchanged); `remote` → client stubs
- [ ] 5.4 Rename `Box3Adapter` to reflect its lasting role as the platform-side Box2→Box3 convenience wrapper

## 6. Validation

- [ ] 6.1 Local mode: full compile/execute/communication regression is byte-for-byte unchanged
- [ ] 6.2 Remote mode on one grader: end-to-end compile, execute, and multibox (communication) evaluations pass
- [ ] 6.3 Verify orphan GC reclaims a crash-orphaned scratch file and never removes a live one
- [ ] 6.4 Verify auth: valid token served, missing/invalid token rejected before execution
- [ ] 6.5 Verify SFTP drop mid-eval fails fast with a Go error (no hang) and the next eval redials
