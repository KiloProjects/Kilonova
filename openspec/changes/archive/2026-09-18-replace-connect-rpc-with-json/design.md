## Context

The remote grader today runs two transports on one TLS listener:

```
  platform                                   grader (kn grader-serve)
  ────────                                   ─────────────────────────
  GraderClient ──ConnectRPC/protobuf──▶ GraderServer  ──▶ BoxManager
      (rpc_client.go, rpc_conv.go)     (rpc_server.go, interceptor auth)
  httpScratch  ──PUT/GET/DELETE────▶ scratchServer ──▶ scratch dir
      (eval/scratch/httpscratch.go)  (scratch_server.go, authMiddleware)
```

The control plane is three unary procedures: `RunBox3`, `RunMultibox3`, `Languages`. Its messages mirror `eval.Box3Request`, `eval.Multibox3Request`, `eval.Box3Response`, `eval.RunStats`, `eval.RunConfig`, `eval.ScratchFile`, `language.Directory` — all exported-field Go structs with JSON-friendly types (`string`, `bool`, `int`, `float64`, `[]string`, `map[string]string`, `fs.FileMode` = `uint32`). `RunStats` already has JSON tags. The proto mirror exists only because ConnectRPC needs it, and `rpc_conv.go` exists only to copy fields between the two.

Verified constraints:
- `ClientRegistry.authMiddleware` (rpc_server.go) is already a plain `http.Handler` wrapper doing the same bearer check as the interceptor.
- `GraderClient` is used only through `eval.Box3Scheduler` (grader.go:665) and `languageVersions` (langmgr.go:107); its constructor signature is the only public surface the platform touches.
- `google.golang.org/protobuf` has no importers outside `eval/scheduler/proto/`; it remains an indirect dependency of the OTLP gRPC exporters.
- Nothing in `.github/` or scripts invokes `buf`.

## Goals / Non-Goals

**Goals:**
- One transport: `net/http` + `encoding/json` for control, `net/http` streaming for data. One mux, one auth middleware, one error convention.
- Zero conversion code: encode `eval.*` structs directly.
- Delete the proto package, the `buf` toolchain, and the ConnectRPC/protobuf module deps.
- Keep `eval.Box3Scheduler`, `NewRemoteLanguageManager`, `grader.toml`, the `mode` switch, the trust model, and `/scratch` byte-for-byte unchanged.
- Keep the security tests: unauthenticated and wrong-token requests never reach execution.

**Non-Goals:**
- Wire versioning / cross-version compatibility between platform and grader (same-build deployment is the contract, as it effectively was already).
- Streaming or long-poll for `RunBox3` progress.
- Changing the scratch data plane, GC, or config schema.
- Adding JSON tags to every `eval.*` struct for cosmetic snake_case. Default Go field names are the wire names; `RunStats` keeps its existing tags.

## Decisions

### D1. Control plane is three JSON endpoints on the existing mux
```
POST /run/box3        {"request": eval.Box3Request, "mem_quota": int64}
                   →  200 eval.Box3Response
POST /run/multibox3   {"request": eval.Multibox3Request, "manager_mem_quota": int64, "individual_mem_quota": int64}
                   →  200 {"manager_response": eval.Box3Response, "user_stats": []eval.RunStats}
GET  /languages    →  200 {"build": string, "versions": map[string]string}
```
Envelope structs are defined once in `eval/scheduler` and used by both client and server, so the wire shape is a Go type, not a doc. Handlers `json.NewDecoder(r.Body).Decode(&in)` → call the in-process `eval.Box3Scheduler` / `eval.LanguageManager` → `json.NewEncoder(w).Encode(out)`. Client is one `post(ctx, path, in, out)` helper using `http.NewRequestWithContext` so cancellation still propagates to a running box.

**Why not keep ConnectRPC?** Its value is streaming, typed codegen across languages, and gRPC interop. None apply: unary only, one language, one binary. It cost a second codec, a generated package larger than the whole hand-written transport, a hand-rolled conversion layer, and four tool deps.

**Why not encode `eval.*` inside proto (e.g. `google.protobuf.Struct`)?** That keeps the toolchain for nothing.

**Why POST instead of RPC-style `/GraderService/RunBox3` paths?** Plain paths read in logs and curl; there is no framework dictating the layout.

### D2. No conversion layer; `eval.*` structs are the wire
`rpc_conv.go` is deleted. `encoding/json` handles every field type in play. `fs.FileMode` marshals as its `uint32` value, matching what the proto carried. `language.Directory` has only `toml` tags, so JSON uses its Go field names — acceptable, both sides share the type.

Consequence: renaming an `eval.*` field changes the wire. Accepted (see Non-Goals) and cheaper than the drift the proto mirror invited — the old round-trip test existed precisely because two parallel struct sets could diverge. The new round-trip test encodes/decodes the real structs, so it guards `omitempty`-style surprises rather than field copying.

### D3. One auth middleware for the whole listener
`grader_serve.go` builds one `http.ServeMux` with `/run/`, `/languages`, and `/scratch/`, then wraps the mux once in `registry.authMiddleware`. The ConnectRPC interceptor, `errMissingToken`/`errUnknownToken`, and the `clientNameKey` context plumbing collapse into the middleware, which also stashes the client name in the context so `ClientName(ctx)` keeps working for logs.

### D4. Error convention: HTTP status + plain-text body
Server: execution errors → `500` with `err.Error()` as body; bad JSON → `400`; auth → `401` (already). Client: any non-2xx → `fmt.Errorf("grader %s %s: %s: %s", method, path, resp.Status, body)`. Tests assert on a sentinel `ErrUnauthenticated` the client returns for `401` (replacing `connect.CodeOf(err) == connect.CodeUnauthenticated`).

### D5. Dependency and toolchain removal is part of the change, not a follow-up
Delete `buf.yaml`, `buf.gen.yaml`, `eval/scheduler/proto/`, remove the four `tool` entries and `connectrpc.com/connect` from `go.mod`, run `go mod tidy`, drop the "Protobufs" line from `CLAUDE.md`. Leaving the toolchain "in case" is exactly the over-engineering being removed.

### D6. `GET /languages` carries the grader's build id; the client refuses a mismatch
The response includes `build`: `kilonova.Version` + the `vcs.revision` from `debug.ReadBuildInfo()` (`-dirty` appended when `vcs.modified`). The platform computes the same string for itself and `languageVersions` returns an error when they differ, so `NewRemoteLanguageManager` fails at startup and `Resync` fails on a later redeploy skew — before any `RunBox3` can be sent with a mismatched struct shape. It rides the call the platform already makes first, so no extra endpoint or round trip.

**Why not a separate `/version` endpoint?** One more path, one more handler, one more test, for a check that is always made right before `/languages` anyway.

**Ceiling:** two dirty trees at different local edits both report `-dirty` and pass. Dev-only; production builds are clean.

## Risks / Trade-offs

- **[Platform and grader on different builds silently disagree on field names]** → D6: the build id check on `/languages` turns skew into a startup/resync error. The JSON decoder is left at default (unknown fields ignored) so nothing else needs to be strict.
- **[`RunBox3` long request has no client timeout]** → unchanged from today; the control `http.Client` carries no `Timeout`, hang protection stays server-side (box time/wall/mem quotas). Context cancellation still aborts the request.
- **[Large `RunConfig` / command vectors]** → `encoding/json` has no message-size ceiling; `http.MaxBytesReader` at 1 MiB on the server bounds a hostile client. Legit requests are kilobytes.
- **[Losing typed errors]** → only one code (`Unauthenticated`) was ever inspected; it becomes a sentinel error. Everything else was `CodeInternal` wrapping `err.Error()`, which is what a `500` body carries now.

## Migration Plan

1. Land the JSON client/server + tests alongside the proto code (both compile), switch `grader_serve.go` and `getRemoteRunner` to the JSON pair.
2. Delete the proto package, conversion layer, `buf` files, and deps; `go mod tidy`; `go vet ./... && go test ./...`.
3. Deploy platform and grader from the same build. Local mode is untouched throughout.
4. **Rollback**: redeploy the previous build on both sides. No config, schema, or scratch-format change.

## Open Questions

- None.
