## Why

The archived `split-grader-rpc` change put the grader behind a network boundary, but chose ConnectRPC + protobuf for the control plane while the data plane already rides plain HTTP on the same listener. That buys a second serialization system, a ~1,300-line generated package, a `buf` toolchain with four `go tool` deps, and a 211-line hand-written proto↔`eval.*` conversion layer — for three unary calls whose payloads are already plain Go structs (`RunStats` even carries JSON tags today). The control plane should be what the data plane already is: `net/http` + `encoding/json`, one mux, one auth middleware.

## What Changes

- **Replace the ConnectRPC control plane with JSON-over-HTTP.** The grader exposes `POST /run/box3`, `POST /run/multibox3`, and `GET /languages`; request and response bodies are `encoding/json` encodings of the existing `eval.Box3Request` / `eval.Multibox3Request` / `eval.Box3Response` / `eval.RunStats` structs wrapped in thin envelope structs carrying the memory quotas. No conversion layer: `eval.*` is encoded directly.
- **Delete the proto/ConnectRPC surface**: `eval/scheduler/proto/**` (the `.proto`, `grader.pb.go`, `graderv1connect/`), `eval/scheduler/rpc_conv.go`, `buf.yaml`, `buf.gen.yaml`, and the `connectrpc.com/connect`, `google.golang.org/protobuf` (direct), `buf`, `protoc-gen-go`, `protoc-gen-connect-go` module/tool dependencies.
- **Unify auth**: the existing `ClientRegistry.authMiddleware` (already guarding `/scratch`) wraps the whole grader mux; the ConnectRPC interceptor is deleted. Unauthenticated calls get `401` before any sandbox runs, as today.
- **Error mapping**: a non-2xx response is a Go error on the client carrying the status and the plain-text body. `401` remains the auth signal for tests and operators.
- **Unchanged**: the `/scratch/{id}` data plane, `GraderClient` implementing `eval.Box3Scheduler`, `NewRemoteLanguageManager`, the `mode = local|remote` switch, the grader config schema, TLS + bearer-token + segmentation trust model, `Close()` no-op semantics, and the orphan GC sweep. Local mode is untouched.
- **Wire compatibility caveat**: platform and grader MUST run the same build. The wire shape is the `eval.*` struct shape; this is accepted because both halves ship from one binary (`kn main` / `kn grader-serve`) and the old proto had the same effective constraint (no versioning policy was ever exercised). `GET /languages` returns the grader's build identifier and the platform refuses a grader whose identifier differs from its own, so skew fails at connect time instead of mid-eval.

## Capabilities

### New Capabilities
<!-- None. -->

### Modified Capabilities
- `grader-transport`: the "reachable over ConnectRPC" requirement becomes "reachable over JSON-over-HTTP endpoints"; the mode-switch, `Close`, and language-cache requirements are unchanged in behavior.
- `grader-auth-config`: the "Authenticated, single-direction transport" requirement no longer references a ConnectRPC interceptor — one bearer-token middleware guards every path on the grader listener.

## Impact

- **Code**: `eval/scheduler/rpc_client.go` and `rpc_server.go` rewritten (~same size, fewer imports); `rpc_conv.go` and `eval/scheduler/proto/**` deleted; `rpc_auth_test.go` / `rpc_roundtrip_test.go` rewritten to drive JSON over `httptest`; `cmd/kn/grader_serve.go` mounts the new handlers on the same mux. `eval/grader/grader.go` call sites keep the same constructor shape.
- **Dependencies removed**: `connectrpc.com/connect`, direct `google.golang.org/protobuf` (stays indirect via OpenTelemetry exporters), `github.com/bufbuild/buf` and the two protoc plugins from the `tool` block, plus their transitive `buf.build/*` modules.
- **Build**: `go tool buf generate` step and `buf.*.yaml` are gone; `CLAUDE.md` "Protobufs" line is removed. `go generate ./...` is unaffected.
- **Ops**: the grader listener, port, TLS cert, token, and `grader.toml` are unchanged. Deploying requires updating platform and grader together (same build). Rollback is redeploying the previous build on both sides; no config or data migration.
- **Specs**: `openspec/specs/grader-transport` and `grader-auth-config` get delta specs; `grader-scratch-transport` is unaffected.
