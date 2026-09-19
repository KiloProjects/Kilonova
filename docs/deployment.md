# Deployment with containers

Kilonova ships two images built from one `Dockerfile`:

| Target | Command | Contents |
|---|---|---|
| `platform` | `kn main` | the web platform. No compilers, no sandbox |
| `grader` | `kn grader-serve` | isolate plus every language toolchain |

```sh
docker build --target platform -t kilonova .
docker build --target grader   -t kilonova-grader .
```

The platform image deliberately carries no toolchains, so it cannot grade
in-process. **`KN_EVAL_MODE=remote` is the only container topology**: every
containerised platform needs a grader to talk to.

## Quick start

```sh
cp .env.example .env     # set POSTGRES_PASSWORD and KN_GRADER_TOKEN
docker compose up --build
```

That brings up Postgres, one grader and the platform on
<http://localhost:8070>. The first account to sign up becomes admin and
proposer. Read *Going to production* below before pointing a domain at it.

To use a grader that lives elsewhere, set `KN_EVAL_REMOTE_ENDPOINT` and start
only the platform: `docker compose up platform`.

## The grader needs privileges

```yaml
grader:
  privileged: true
  cgroup: host        # docker run --cgroupns=host
```

isolate creates and manages a cgroup v2 subtree per sandbox, which needs more
than a default container gets. Two consequences:

- **The grader container is close to host root.** `/run/box3` is arbitrary code
  execution as a service, so a leaked token is RCE. Never publish its port:
  keep it reachable only from the platform, on a private network or behind an
  IP allowlist. The compose stack has no `ports:` on the grader for this reason.
- **It holds no platform credentials** — no database DSN, no datastore access,
  and it never connects back. Keep it that way.

The grader refuses to start rather than grade without limits: if the sandbox is
unusable you get `secure sandbox (isolate) is unavailable; refusing to start`
and a non-zero exit.

### No systemd, no cg-keeper

Upstream isolate ships `isolate-cg-keeper`, a daemon whose only job is to hold
a systemd-delegated cgroup open. `kn grader-serve` is already a long-lived
process, so it does that setup itself when `KN_SANDBOX_ENSURE_CG_KEEPER=true`
(baked into the grader image): it checks for cgroup v2, resolves its own cgroup,
publishes the path where isolate reads it, moves itself into a `daemon` leaf so
controllers can be enabled for the per-box groups, and enables `cpuset` and
`memory`. No init system is needed in the container.

Both cgroup namespace modes work — `--cgroupns=host` (recommended, and what
compose uses) and the default private namespace.

!!! warning "Do not run both keepers"
    On a host where isolate's own `isolate.service` manages the cgroup, leave
    `KN_SANDBOX_ENSURE_CG_KEEPER` at its default `false`. Two owners of one
    subtree is not a supported configuration.

Diagnose a host with the upstream checker, which is installed in the image:

```sh
docker compose exec grader isolate-check-environment
```

All cgroup checks should say `PASS`. Warnings about swap, address-space
randomisation and transparent hugepages are host kernel tuning that affect
measurement *variability*, not correctness; tune them on a dedicated judge host
if you care about reproducible timings.

## TLS between platform and grader

`kn grader-serve` will not start without `KN_GRADER_TLS_CERT` and
`KN_GRADER_TLS_KEY`. When those files are absent the grader image's entrypoint
generates a self-signed certificate for its own hostname, so the stack comes up
with no manual step.

Nothing trusts a self-signed certificate, so the compose stack also sets
`KN_SANDBOX_ALLOW_INSECURE=true` on the **platform**, which skips verifying the
grader's certificate. This is a development convenience:

- it disables verification only — the connection is still TLS, and the bearer
  token is still required and checked;
- it applies to the control plane and the `/scratch` data plane together, so the
  two can never disagree;
- the platform logs a warning naming the endpoint at every startup.

In production, mount a real certificate over `/etc/kilonova/grader.crt` and
`/etc/kilonova/grader.key`, issued for the name the platform uses, and remove
`KN_SANDBOX_ALLOW_INSECURE`. Without verification, anything that can intercept
the connection can take the token, and the token is remote code execution.

The same variable means something different in `KN_EVAL_MODE=local`: there it
permits the insecure stupidbox fallback. It is never for production either way.

## Storage

| Volume | Holds |
|---|---|
| `db-data` | Postgres, including the runtime flag store |
| `platform-data` | datastore buckets: tests, subtests, attachments, avatars |
| `grader-data` | scratch transfers and the `uv` cache used by Python submissions |

`platform-data` is the one that matters for backups alongside the database.
Grader data is scratch and can be discarded. Object storage for the datastore
is not supported: buckets are local files.

Both containers run as root. Nothing in the platform needs it — set
`user:` in compose if you would rather it ran as your own uid, remembering that
the data directory must then be writable by that uid. The grader does need it,
for isolate.

## Logs

The images set `KN_LOG_FILE=false`, so everything goes to stdout and you read
it with `docker logs`. On a host deployment the variable defaults to `true` and
the rotating files under `$KN_DATA_DIR/logs` keep working as before.

With file logging off, the dedicated logs that normally get their own file
(`db.log`, `grader.log`, `sandbox_runs.log`, `eviction.log`, `email.log`) are
written to stdout too rather than being dropped.

## Lifecycle

Both processes handle SIGTERM and drain in-flight work before exiting, so
`docker stop` and rolling restarts are clean rather than 10-second kills.

Both serve `GET /healthz` unauthenticated, which is what the images' `HEALTHCHECK`
and compose's `depends_on: condition: service_healthy` use. The platform's
readiness includes a database ping and reports unhealthy while draining; the
grader's `/healthz` is the one path that bypasses its bearer-token middleware.

## Configuration

Everything is environment variables — see `.env.example` and
[Configuration](configuration.md). Nothing needs a mounted config file: runtime
flags live in the database, so a platform container needs no writable
configuration state at all.

A deployment upgrading from a `flags.json` should bring that file along once
(`KN_FLAGS_PATH`): the first start imports it into the database and never
touches it again.

## Going to production

The shipped compose file is a working stack, not a hardened one. Before real use:

- **Terminate TLS in front of the platform** and set `KN_HOST_PREFIX` to the
  https URL, plus `KN_TRUE_IP_HEADER` (for example `X-Forwarded-For`) so client
  IPs are correct. Then set `KN_DEBUG=false`.

    !!! note
        The compose file defaults `KN_DEBUG=true` because the OIDC provider
        refuses a non-https issuer unless debug is on, and the out-of-the-box
        stack runs on `http://localhost`. An https `KN_HOST_PREFIX` removes the
        need for it.

- Issue a real grader certificate and drop `KN_SANDBOX_ALLOW_INSECURE`.
- Generate a long random `KN_GRADER_TOKEN`; it is an RCE credential.
- Put the grader on an isolated network, and size `KN_SANDBOX_NUM_CONCURRENT`
  and `KN_SANDBOX_GLOBAL_MAX_MEM_KB` to the judge host.
- Pin image tags. Language versions come from the base image and are reported by
  probing, not asserted, so a rebuild can change the compiler contestants get.

## Notes on languages

The grader image is deliberately fat: languages are discovered by probing for
their binaries, so a missing toolchain silently removes that language rather
than failing anything. `GET /languages` on the grader lists what it found.

Two absences are deliberate: Haskell is disabled in the language table
(`eval/language/legacy.go`), so ghc is not installed; and `uv` is present for AI
checkers but is not a submission language.

JVM languages need real memory to *compile*: Kotlin fails with a killed compiler
at a 64 MB problem limit and wants a few hundred MB. That is a per-problem
limit, not an image setting.
