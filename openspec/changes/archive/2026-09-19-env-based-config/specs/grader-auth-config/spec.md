## MODIFIED Requirements

### Requirement: Role-split configuration
Configuration SHALL be split by role and supplied through environment variables as defined by the `env-config` capability. In remote mode the grader SHALL run from its own variables holding box-execution settings (`KN_SANDBOX_NUM_CONCURRENT`, `KN_SANDBOX_GLOBAL_MAX_MEM_KB`, `KN_SANDBOX_STARTING_BOX`), its root directory and scratch TTL (`KN_DATA_DIR`, under which `scratch/` and `logs/` live, and `KN_GRADER_SCRATCH_TTL_SEC`)
, its listen/TLS settings (`KN_GRADER_LISTEN`, `KN_GRADER_TLS_CERT`, `KN_GRADER_TLS_KEY`), and a per-client registry (one `KN_GRADER_CLIENT_<NAME>=<token>` variable per client, `NAME` lowercased being the client name). The platform SHALL hold a `KN_EVAL_MODE=local|remote` switch and, in remote mode, only the grader endpoint and its bearer token (`KN_EVAL_REMOTE_ENDPOINT`, `KN_EVAL_REMOTE_TOKEN`) — the JSON control plane and the `/scratch` data plane share that one endpoint, so no separate data-plane connection settings are needed. In `local` mode the platform SHALL read the in-process box-execution settings from the same `KN_SANDBOX_*` variables, and no grader-specific variables are required. Neither process SHALL require a configuration file for these settings.

#### Scenario: Grader owns execution settings in remote mode
- **WHEN** the grader starts in remote mode
- **THEN** it reads box-execution settings from its own `KN_SANDBOX_*` variables, and the platform does not need them

#### Scenario: Local mode needs no grader variables
- **WHEN** the platform runs with `KN_EVAL_MODE=local`
- **THEN** it reads `KN_SANDBOX_*` for the in-process grader and starts without any `KN_GRADER_*` or `KN_EVAL_REMOTE_*` variable set

#### Scenario: Client registry from environment
- **WHEN** `KN_GRADER_CLIENT_KILONOVA=abc123` and `KN_GRADER_CLIENT_STAGING=def456` are set

- **THEN** the grader registers two clients, `kilonova` and `staging`, with those tokens
