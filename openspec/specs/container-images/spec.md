# container-images Specification

## Purpose

Defines the container images Kilonova ships — what each one contains, how it is built from the repository, what it requires from the host at runtime (volumes, privileges, cgroups), and the compose topologies operators can start from.

## Requirements

### Requirement: Two images with disjoint roles
The repository SHALL build two images from its own sources: a **platform image** whose default command is `kn main`, and a **grader image** whose default command is `kn grader-serve`. The platform image SHALL NOT contain language toolchains or the sandbox binary. The grader image SHALL contain the `isolate` binary and every toolchain backing a language the grader probes for, so that language auto-detection finds them: C/C++ (`gcc`, `g++`, `mold`), Pascal (`fpc`), Go, Kotlin (`kotlinc`) and a JDK, Python 3, Node, PHP and Rust. It SHALL also carry `uv`, which backs AI checkers rather than any submission language. A toolchain for a language disabled in the language table — Haskell at the time of writing — SHALL NOT be installed, since the language cannot appear however good the toolchain is. Every toolchain SHALL be installed somewhere the sandbox can see it, which excludes `/opt`. A toolchain missing from the grader image SHALL be understood to remove that language from the platform's language list, not to fail startup. The platform image SHALL contain `diff`, which the built-in diff checker executes on the platform side.

#### Scenario: Platform image is toolchain-free
- **WHEN** the platform image is started with `KN_EVAL_MODE=remote`
- **THEN** it serves the site and grades submissions through the remote grader without any compiler present in the image

#### Scenario: Grader reports its languages
- **WHEN** the grader image starts and the platform queries its language list
- **THEN** every language whose toolchain is listed above is reported with a probed version string, and none reports a probe error

#### Scenario: A compiler the sandbox cannot see is a missing compiler
- **WHEN** a toolchain is installed only under a path the sandbox does not mount, such as `/opt`
- **THEN** its version probe fails inside the sandbox and the language is unusable, even though the binary exists in the image

#### Scenario: Diff checker works in the platform image
- **WHEN** a submission to a problem using the default checker is graded
- **THEN** the platform compares outputs successfully, without a "checker internal error" caused by a missing `diff`

### Requirement: Images are built in the repository's generate → assets → compile order
The image build SHALL run code generation (translations, chroma stylesheet, templ components), then the frontend asset build, then `go build`, in that order, because generated files and the Vite manifests are embedded into the binary. The resulting runtime image SHALL NOT contain the Go toolchain, Node, pnpm or the asset sources. A build that skips generation or the asset step SHALL fail rather than produce an image with missing embedded assets.

#### Scenario: Built image serves hashed assets
- **WHEN** a container from the platform image serves any page
- **THEN** the referenced `/static/misc/...` URLs resolve to files embedded in the binary

#### Scenario: Skipping the asset build fails the image build
- **WHEN** the asset build stage is removed or fails
- **THEN** the image build fails at the Go stage or its tests, and no runtime image is produced

#### Scenario: Runtime image carries no build tooling
- **WHEN** an operator inspects the runtime image
- **THEN** neither `go`, `node` nor `pnpm` is present in it

### Requirement: The grader container requires privileged cgroup access
The grader image SHALL be documented and shipped to run with `--privileged` and `--cgroupns=host` (compose: `privileged: true`, `cgroup: host`), because `isolate` creates and manages cgroup v2 subtrees for each sandbox. The grader SHALL refuse to start when a secure sandbox is unavailable, rather than falling back to the insecure sandbox, unless `KN_SANDBOX_ALLOW_INSECURE` is explicitly set. The grader container SHALL have a writable data volume at `KN_DATA_DIR`, under which `scratch/`, `logs/` and the `uv/` cache used by Python submissions live.

#### Scenario: Unprivileged grader fails loudly
- **WHEN** the grader container is started without the privileges isolate needs
- **THEN** it exits non-zero at startup with an error stating that the secure sandbox is unavailable, and does not serve requests

#### Scenario: Sandbox cgroups are created
- **WHEN** a submission is evaluated in a correctly privileged grader container
- **THEN** isolate enforces the configured memory and time limits and the run reports resource usage

#### Scenario: Python submissions have a writable uv cache
- **WHEN** a `uv` submission runs
- **THEN** the sandbox mount backed by `<KN_DATA_DIR>/uv` is writable and the run succeeds

### Requirement: The grader prepares its own sandbox cgroup
When `KN_SANDBOX_ENSURE_CG_KEEPER` is enabled, the grader SHALL perform the cgroup preparation that isolate's separate keeper daemon performs under systemd, so that no systemd, supervisor or second process is needed in a container: it SHALL verify the host presents cgroup v2 (and refuse a v1 or hybrid layout), determine the control-group root isolate is configured to use, publish that path where isolate expects to read it, move itself into a leaf subgroup so that controllers can be enabled for isolate's per-box groups, and enable the `cpuset` and `memory` controllers there. Preparation SHALL run once per process before the first sandbox is created, SHALL be idempotent across restarts that reuse the same cgroup, and SHALL fail startup with a diagnostic naming the failed step rather than proceeding to evaluate submissions. The setting SHALL default to disabled, so a host running isolate's own keeper service is unaffected; enabling both SHALL be documented as unsupported.

#### Scenario: Grader in a container grades without systemd
- **WHEN** the grader container starts with the setting enabled and no init system, keeper daemon or supervisor present
- **THEN** it prepares the cgroup itself and evaluates submissions with time and memory limits enforced

#### Scenario: Restart in a live container cgroup
- **WHEN** `kn grader-serve` is restarted inside a container whose cgroup and leaf subgroup already exist
- **THEN** preparation succeeds without an "already exists" failure and grading resumes

#### Scenario: Cgroup v1 host is refused
- **WHEN** the setting is enabled on a host that does not present cgroup v2
- **THEN** the grader exits with an error saying so, and does not fall back to running sandboxes without resource limits

#### Scenario: Host with the keeper service is untouched
- **WHEN** an operator runs the grader on a host where isolate's own keeper service manages the cgroup, leaving the setting at its default
- **THEN** the grader performs no cgroup preparation of its own and uses the existing subtree

### Requirement: A compose stack covers the whole platform
The repository SHALL provide a compose stack containing Postgres, the platform in `KN_EVAL_MODE=remote` and the grader, since the toolchain-free platform image cannot sandbox locally and remote eval is therefore the only container topology. It SHALL start from a clean checkout with no manual file creation beyond an environment file, SHALL persist Postgres data and the platform data directory in named volumes, and SHALL NOT bind-mount any configuration file into a container. The grader SHALL NOT be published on a host port; the platform SHALL reach it over the compose network with its bearer token. An operator SHALL be able to run the platform service against an already-running grader elsewhere by pointing `KN_EVAL_REMOTE_ENDPOINT` at it and not starting the grader service.

#### Scenario: Stack comes up
- **WHEN** an operator runs the compose stack on a fresh checkout with an environment file
- **THEN** the database is migrated, the platform becomes healthy, and the site answers on the published port

#### Scenario: Stack grades a submission
- **WHEN** the stack is up and a submission is sent
- **THEN** the platform ships it to the grader over the compose network and records the verdict

#### Scenario: Restart preserves state
- **WHEN** the stack is stopped and started again
- **THEN** users, problems, uploaded tests and runtime flags are all still present

#### Scenario: Grader is not exposed
- **WHEN** the stack is running
- **THEN** the grader's port is reachable only from the platform container, not from the host or outside

### Requirement: Image build context excludes generated and local artifacts
The build context SHALL exclude `node_modules`, generated assets and manifests, the local data directory, `.env`, the compiled `kn` binary, and version-control and editor metadata, so that stale host artifacts cannot leak into an image and the build is reproducible from sources alone.

#### Scenario: Stale host assets do not reach the image
- **WHEN** the host checkout contains a previously built `web/static/misc` and a compiled `./kn`
- **THEN** the image is built from freshly generated assets and a freshly compiled binary

#### Scenario: Local secrets do not reach the image
- **WHEN** the host checkout contains a `.env` with production credentials
- **THEN** that file is not present in any image layer
