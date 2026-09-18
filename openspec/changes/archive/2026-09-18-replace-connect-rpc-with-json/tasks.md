## 1. JSON control plane

- [x] 1.1 Define the wire envelope structs in `eval/scheduler` (`runBox3Req`, `runMultibox3Req`, `runMultibox3Resp`) wrapping the existing `eval.*` types with JSON tags for the quota fields
- [x] 1.2 Rewrite `eval/scheduler/rpc_server.go`: `GraderServer.Handler()` returns an `http.Handler` serving `POST /run/box3`, `POST /run/multibox3`, `GET /languages` via `encoding/json`, with `http.MaxBytesReader` (1 MiB) on bodies, `400` on bad JSON, `500` + error text on scheduler errors; drop the interceptor, keep `ClientRegistry`, `authMiddleware` (now setting `clientNameKey`), and `ClientName`
- [x] 1.3 Rewrite `eval/scheduler/rpc_client.go`: `GraderClient{client *http.Client, baseURL, token}` with one `post`/`get` helper using `http.NewRequestWithContext`; non-2xx → error with status and body; `401` → exported `ErrUnauthenticated` sentinel; `RunBox3`, `RunMultibox3`, `languageVersions`, no-op `Close` unchanged in signature
- [x] 1.4 Update `cmd/kn/grader_serve.go`: one mux with `/run/`, `/languages`, `/scratch/`, wrapped once in `registry.authMiddleware`; simplify `ScratchHandler` to no longer take the registry
- [x] 1.5 Add `BuildID()` (`kilonova.Version` + `vcs.revision`, `-dirty` when modified) in `eval/scheduler`; `GET /languages` returns `{build, versions}`; `languageVersions` errors on mismatch
- [x] 1.6 Update `eval/grader/grader.go` `getRemoteRunner` to the new `NewGraderClient(http.DefaultClient, endpoint, token)` signature (drop connect options)

## 2. Tests

- [x] 2.1 Rewrite `rpc_auth_test.go`: valid token served and executes; wrong token → `ErrUnauthenticated`, no execution; missing token via raw `http.Post` → `401`, no execution
- [x] 2.2 Rewrite `rpc_roundtrip_test.go`: drive `SaveFile → RunBox3 → ReadFile → DeleteFile` through `GraderClient` against an `httptest` server hosting `GraderServer` + `echoSched`; keep `TestRunConfigConversionIsLossless` as a JSON encode/decode `reflect.DeepEqual` over the full `RunConfig`
- [x] 2.3 Adjust `scratch_server_test.go` for the auth-middleware relocation (mount `ScratchHandler` under `authMiddleware` in the test helper)
- [x] 2.4 Add a test that a grader reporting a different build id makes `languageVersions` fail and a matching one succeed
- [x] 2.5 `go test ./eval/...` green

## 3. Remove ConnectRPC / protobuf

- [x] 3.1 Delete `eval/scheduler/rpc_conv.go` and `eval/scheduler/proto/` (proto source, `grader.pb.go`, `graderv1connect/`)
- [x] 3.2 Delete `buf.yaml` and `buf.gen.yaml`
- [x] 3.3 Remove `connectrpc.com/connect` and the `tool` entries for `buf`, `protoc-gen-go`, `protoc-gen-connect-go` from `go.mod`; `go mod tidy` (expect `google.golang.org/protobuf` to become indirect and the `buf.build/*` modules to drop)
- [x] 3.4 `grep -rn connectrpc\|graderv1\|protobuf --include='*.go'` returns nothing outside `go.sum`/indirect deps

## 4. Docs and verification

- [x] 4.1 Remove the "Protobufs: `go tool buf generate`" line from `CLAUDE.md` and reword the grading-mode bullet (Connect RPC → JSON over HTTP)
- [x] 4.2 Add a short `openspec/changes/replace-connect-rpc-with-json/deploy-notes.md` stating the same-build requirement and the endpoint list, superseding the "Control plane (ConnectRPC)" section of the archived deploy doc
- [x] 4.3 `go generate ./... && go build ./cmd/kn && go vet ./... && go test ./... && golangci-lint run` all pass
