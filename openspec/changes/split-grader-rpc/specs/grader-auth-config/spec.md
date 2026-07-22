## ADDED Requirements

### Requirement: Role-split configuration
Configuration SHALL be split by role. In remote mode the grader SHALL run from its own config file holding box-execution settings (`num_concurrent`, `global_max_mem_kb`, `starting_box`), its scratch directory, its listen/TLS settings, and a per-client registry. The platform config SHALL hold a `mode = local|remote` switch and, in remote mode, the grader endpoint, its bearer token, and SFTP connection settings. In `local` mode the existing single `config.toml` (including in-process `[eval]` execution settings) SHALL continue to work unchanged.

#### Scenario: Grader owns execution settings in remote mode
- **WHEN** the grader starts in remote mode
- **THEN** it reads box-execution settings from its own config file, and the platform config no longer needs them

#### Scenario: Local mode config is unchanged
- **WHEN** the platform runs in `local` mode
- **THEN** it reads `[eval]` execution settings from the existing `config.toml` with no grader config file required

### Requirement: Per-client token registry with identity
The grader SHALL maintain a registry of allowed platform clients, each entry carrying a name and a token. The registry SHALL support multiple platform instances connecting to a single grader. Client identity SHALL be available for observability (attributing runs and metrics to a named client). The registry entry MAY carry a priority-class field that is documented but unconsumed by this change.

#### Scenario: Multiple platform instances share one grader
- **WHEN** two platform instances each present their own registered token
- **THEN** the grader accepts both and can attribute each run to the issuing client by name

#### Scenario: Priority field is reserved, not enforced
- **WHEN** a client registry entry declares a priority class
- **THEN** the grader accepts the config but does not yet alter admission ordering based on it

### Requirement: Authenticated, single-direction transport
Every RPC SHALL be authenticated by a grader-minted bearer token presented over TLS via a ConnectRPC interceptor; SFTP SHALL authenticate via SSH key. The grader SHALL initiate no connections back to the platform and SHALL hold no platform credentials. Operators SHALL treat the token as insufficient on its own and MUST additionally restrict grader network reachability to the platform (segmentation / IP allowlist).

#### Scenario: Request with a valid token is served
- **WHEN** a platform presents a registered token over TLS
- **THEN** the grader authenticates the client and serves the request

#### Scenario: Request with an invalid or missing token is rejected
- **WHEN** a caller presents no token or an unregistered token
- **THEN** the grader rejects the request without executing any sandbox

#### Scenario: Grader holds no reverse credentials
- **WHEN** a sandbox escape compromises the grader host
- **THEN** the attacker finds no platform DB DSN, datastore credentials, or outbound platform connections on that host
