# Deploying a remote grader

Reference for standing up the split grader (task 3.4 + migration plan). The grader
holds **no platform credentials** — keep it that way.

## Trust model (all three are mandatory)

1. **TLS** authenticates the grader to the platform (server cert).
2. A **grader-minted bearer token** per platform instance authenticates the platform
   to the grader (ConnectRPC `Authorization: Bearer` header).
3. **Network segmentation / IP allowlist.** `RunBox3` is arbitrary-code-execution
   as a service — a leaked token is RCE. The grader MUST only be reachable from the
   platform. Token + TLS are necessary, not sufficient.

## Control plane (ConnectRPC)

`kn grader-serve --grader-config /etc/kilonova/grader.toml`

```toml
[grader]
listen           = ":9000"
cert_file        = "/etc/kilonova/grader.crt"
key_file         = "/etc/kilonova/grader.key"
scratch_dir      = "/var/lib/kilonova/scratch"
num_concurrent   = 6
global_max_mem_kb = 2097152
starting_box     = 0
scratch_ttl_sec  = 3600            # orphan GC TTL; must be >> max eval duration

[[grader.client]]
name  = "platform-a"
token = "<64+ random bytes, base64>"   # mint here, store on the platform
# priority = "low"                       # reserved, unconsumed for now

[[grader.client]]
name  = "platform-b"
token = "<another token>"
```

The grader process also reads the standard `config.toml` (via the global `--config`
flag) for `[common] log_dir` etc. — ship it a minimal one; it needs no DB DSN.

## Data plane (HTTP /scratch)

Bytes move over the **same** `listen` endpoint as the RPC, behind the same TLS
cert and the same bearer token — `PUT`/`GET`/`DELETE https://grader:9000/scratch/{id}`,
where `{id}` is a platform-minted UUID. No `sshd`, no SSH keys, no host-key
management, no second listening service. Path traversal is impossible: the grader
parses `{id}` as a UUID and rejects anything else, so a client can only ever name
a flat file in the scratch dir.

## Platform side

```toml
[eval]
mode = "remote"

[eval.remote]
endpoint = "https://grader.internal:9000"   # RPC and /scratch both live here
token    = "<the token minted for this instance>"
```

## Rollback

Flip `mode` back to `local`. No schema or data migration is involved.
