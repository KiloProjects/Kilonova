# grader-transport Specification

## Purpose

Defines how the platform reaches the grader's execution and language-inventory surfaces over ConnectRPC, and how the local/remote mode switch selects between an in-process grader and remote client stubs.

## Requirements

### Requirement: Grader execution and language inventory are reachable over ConnectRPC
The grader SHALL expose the `Box3Scheduler` surface (`RunBox3`, `RunMultibox3`) and the `LanguageManager` surface (language list and versions) as a ConnectRPC service. All request/response messages SHALL be unary and SHALL carry only scratch identifiers, commands, run configuration, run statistics, and language metadata — never file bytes.

#### Scenario: Platform runs a box on the remote grader
- **WHEN** the platform's `Box3Scheduler` client stub calls `RunBox3` with a request referencing input scratch identifiers
- **THEN** the grader executes the sandbox against its local scratch and returns a response whose `Files` map references output scratch identifiers, with no file bytes in the RPC payload

#### Scenario: Communication (multibox) problem over RPC
- **WHEN** the platform calls `RunMultibox3` with a manager sandbox config and one or more user sandbox configs
- **THEN** the grader runs them in parallel with its local FIFO setup and returns the manager response plus per-user-sandbox stats

### Requirement: Mode switch selects local or remote grader
The platform SHALL select between an in-process grader (`mode = local`) and remote client stubs (`mode = remote`) at the existing wiring point. In `local` mode behavior SHALL be identical to the pre-change in-process path, with no token, no remote scratch endpoint, and no new runtime dependency exercised.

#### Scenario: Local mode preserves current behavior
- **WHEN** the platform starts with `mode = local`
- **THEN** it constructs the in-process `BoxManager` and local scratch exactly as before, requiring no grader config file, token, or remote scratch connection

#### Scenario: Remote mode uses client stubs
- **WHEN** the platform starts with `mode = remote`
- **THEN** it constructs client stubs implementing `eval.Box3Scheduler`, `eval.LanguageManager`, and `eval.Scratch`, targeting the configured grader endpoint

### Requirement: Remote Close does not affect a shared grader
The remote `Box3Scheduler.Close` SHALL be a client-side no-op (or at most a per-session drain). One platform instance shutting down SHALL NOT cause the grader to drain boxes or stop serving other platform instances.

#### Scenario: One platform disconnects, grader keeps serving
- **WHEN** a platform instance calls `Close` on its remote scheduler and shuts down
- **THEN** the grader continues accepting and serving `RunBox3` requests from other connected platform instances

### Requirement: Language metadata is pulled and cached with manual resync
The platform SHALL obtain the grader's language list and versions over RPC, cache them, and expose a manual resync action. The grader remains authoritative about which languages and versions are installed.

#### Scenario: Platform caches language metadata
- **WHEN** the platform connects to the grader
- **THEN** it fetches the language inventory once and serves subsequent lookups from cache

#### Scenario: Manual resync after grader redeploy
- **WHEN** an operator triggers a language resync on the platform
- **THEN** the platform re-fetches the inventory from the grader and replaces its cache
