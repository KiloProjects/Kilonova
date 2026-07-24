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

## Data plane (SFTP)

Bytes move over the operator's `sshd`, **not** through the RPC. Configure an sftp
subsystem chrooted to the scratch dir so the platform's ssh user can only see
scratch:

```
# /etc/ssh/sshd_config
Match User kilonova-scratch
    ChrootDirectory /var/lib/kilonova/scratch
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
    PermitTunnel no
```

Chroot requires the chroot dir be root-owned; put the writable scratch in a
subdirectory owned by `kilonova-scratch`, and set `scratch_dir` (grader) +
`scratch_base` (platform) to that subdir. Add the platform's ssh public key to
`kilonova-scratch`'s `authorized_keys`.

## Platform side

```toml
[eval]
mode = "remote"

[eval.remote]
endpoint = "https://grader.internal:9000"
token    = "<the token minted for this instance>"

[eval.remote.sftp]
addr          = "grader.internal:22"
user          = "kilonova-scratch"
key_path      = "/etc/kilonova/scratch_id_ed25519"
host_key_path = "/etc/kilonova/grader_host_key.pub"   # optional; pins the host key
scratch_base  = ""                                    # "" when chrooted
max_conns     = 4
timeout_sec   = 30
```

Leaving `host_key_path` empty skips host-key verification (relying on segmentation);
set it in production.

## Rollback

Flip `mode` back to `local`. No schema or data migration is involved.
