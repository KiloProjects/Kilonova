## MODIFIED Requirements

### Requirement: Grader execution and language inventory are reachable over ConnectRPC
The grader SHALL expose the `Box3Scheduler` surface (`RunBox3`, `RunMultibox3`) and the `LanguageManager` surface (language list and versions) as JSON-over-HTTP endpoints on its listener: `POST /run/box3`, `POST /run/multibox3`, and `GET /languages`. Request and response bodies SHALL be `encoding/json` encodings of the existing `eval` structs (`Box3Request`, `Multibox3Request`, `Box3Response`, `RunStats`) wrapped in envelope structs that carry the memory quotas; no separate wire schema, codegen, or conversion layer SHALL exist. All calls SHALL be request/response (no streaming) and SHALL carry only scratch identifiers, commands, run configuration, run statistics, and language metadata — never file bytes. A non-2xx response SHALL be surfaced to the platform as a Go error carrying the HTTP status and the response body. Platform and grader SHALL be deployed from the same build, since the `eval` struct shape is the wire contract; `GET /languages` SHALL return the grader's build identifier and the platform SHALL refuse to use a grader whose identifier differs from its own.

#### Scenario: Platform runs a box on the remote grader
- **WHEN** the platform's `Box3Scheduler` client stub calls `RunBox3` with a request referencing input scratch identifiers
- **THEN** the client POSTs the JSON-encoded request to `/run/box3`, the grader executes the sandbox against its local scratch, and the decoded `Box3Response` references output scratch identifiers with no file bytes in the payload

#### Scenario: Communication (multibox) problem over HTTP
- **WHEN** the platform calls `RunMultibox3` with a manager sandbox config and one or more user sandbox configs
- **THEN** the client POSTs to `/run/multibox3` and the grader runs them in parallel with its local FIFO setup, returning the manager response plus per-user-sandbox stats

#### Scenario: Language inventory fetch
- **WHEN** the platform's remote language manager (re)syncs
- **THEN** it GETs `/languages` and receives a JSON object with the grader's build identifier and a map of supported language name to installed version string

#### Scenario: Build mismatch is refused
- **WHEN** the build identifier returned by `/languages` differs from the platform's own build identifier
- **THEN** the remote language manager's fetch returns an error naming both identifiers, remote startup (or resync) fails, and no run request is sent to that grader

#### Scenario: Grader-side execution error is surfaced
- **WHEN** the grader's in-process scheduler returns an error for a run
- **THEN** the grader responds with a 5xx status and the error text as the body, and the platform's call returns an error containing that status and text

#### Scenario: Round-trip is lossless
- **WHEN** a `Box3Request` with input files, command, full `RunConfig` (including directories and env maps), and output paths is encoded by the client and decoded by the grader
- **THEN** the decoded struct is deeply equal to the original, and the `Box3Response` returned survives the reverse trip likewise

### Requirement: Mode switch selects local or remote grader
The platform SHALL select between an in-process grader (`mode = local`) and remote client stubs (`mode = remote`) at the existing wiring point. In `local` mode behavior SHALL be identical to the pre-change in-process path, with no token, no remote scratch endpoint, and no new runtime dependency exercised.

#### Scenario: Local mode preserves current behavior
- **WHEN** the platform starts with `mode = local`
- **THEN** it constructs the in-process `BoxManager` and local scratch exactly as before, requiring no grader config file, token, or remote scratch connection

#### Scenario: Remote mode uses client stubs
- **WHEN** the platform starts with `mode = remote`
- **THEN** it constructs client stubs implementing `eval.Box3Scheduler`, `eval.LanguageManager`, and `eval.Scratch`, all speaking plain HTTP(S) to the configured grader endpoint with no RPC framework or generated code involved
