# Remote grader: JSON control plane

Supersedes the "Control plane (ConnectRPC)" section of the archived
`split-grader-rpc/deploy-remote-grader.md`. Everything else there (trust model,
`grader.toml`, `/scratch` data plane, platform `[eval.remote]` block, rollback)
still applies unchanged.

## Endpoints (all behind `Authorization: Bearer <token>` on the one TLS listener)

| Method | Path              | Body                                                                 |
|--------|-------------------|----------------------------------------------------------------------|
| POST   | `/run/box3`       | `{"request": Box3Request, "mem_quota": n}` → `Box3Response`           |
| POST   | `/run/multibox3`  | `{"request": Multibox3Request, "manager_mem_quota": n, "individual_mem_quota": n}` → `{"manager_response", "user_stats"}` |
| GET    | `/languages`      | → `{"build": "<id>", "versions": {"cpp": "14", ...}}`                 |
| PUT/GET/DELETE | `/scratch/{uuid}` | raw file bytes                                                |

Bodies are `encoding/json` of the `eval.*` Go structs. Non-2xx responses carry a
plain-text error; `401` means the token is missing or unregistered.

## Same build, both sides

The wire shape is the Go struct shape, so **platform and grader must run the same
build**. `GET /languages` returns the grader's build id
(`kilonova.Version` + VCS revision, `-dirty` if modified) and the platform refuses
to start in remote mode — or to resync — against a grader whose id differs from
its own. Deploy both from the same artifact; the error message names both ids.

Quick check from the platform host:

```sh
curl -sS -H "Authorization: Bearer $TOKEN" https://grader.internal:9000/languages
```
