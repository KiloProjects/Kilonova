# grader-scratch-transport Specification

## Purpose

Defines how scratch files move between the platform and a remote grader over an HTTP `/scratch/{id}` endpoint, including write ordering, failure handling, and orphan garbage collection.

## Requirements

### Requirement: Remote scratch is served over an HTTP endpoint
In remote mode the platform SHALL implement `eval.Scratch` over the grader's HTTP `/scratch/{id}` endpoint (`PUT`/`GET`/`DELETE`), served on the same TLS listener and authenticated by the same bearer token as the RPC control plane. No separate data-plane service (SSH/`sshd`, FUSE/sshfs) SHALL be required. The grader SHALL own the scratch directory on its local disk; the platform SHALL read and write scratch files over HTTP. All connections SHALL be initiated platform→grader. The grader SHALL parse `{id}` as a UUID and reject any other path, so a client can only ever name a flat file in the scratch directory.

#### Scenario: Platform uploads an input file
- **WHEN** the platform-side adapter calls `SaveFile` with a reader in remote mode
- **THEN** the bytes are streamed via `PUT /scratch/{id}` to the grader's scratch directory and a UUIDv7 identifier is returned, with no privileged mount and no second listening service on either side

#### Scenario: Platform downloads an output file
- **WHEN** the platform calls `ReadFile` with an identifier produced by a `RunBox3` response
- **THEN** the file bytes are streamed back via `GET /scratch/{id}` and the identifier can then be deleted via `DELETE /scratch/{id}`

#### Scenario: Traversal-shaped identifier is rejected
- **WHEN** a request targets `/scratch/{id}` where `{id}` is not a valid UUID (e.g. contains `/` or `..`)
- **THEN** the grader responds `400` and serves no file, so the endpoint can only ever reach a flat file in the scratch directory

### Requirement: Write ordering before RPC
The platform SHALL wait for the scratch `PUT` to complete before issuing a `RunBox3`/`RunMultibox3` request that references the written identifier, guaranteeing the bytes are on the grader's disk before it reads them locally. The grader's `PUT` handler SHALL flush and close the file before responding.

#### Scenario: Input is durable before execution
- **WHEN** the platform writes an input scratch file and then requests a box run referencing it
- **THEN** the `PUT` has returned its success status (the grader having flushed and closed the file) before the run request is sent, so the grader's local read never races the transfer

### Requirement: Scratch connection failures are explicit and recoverable
The platform SHALL bound scratch HTTP operations with a client timeout. A connection drop mid-evaluation SHALL surface as a Go error that fails the evaluation fast (eligible for retry), never as an indefinitely blocked operation.

#### Scenario: Grader restarts mid-evaluation
- **WHEN** the scratch connection drops while an evaluation is reading or writing scratch
- **THEN** the operation returns an error within the client timeout, the evaluation fails fast, and subsequent evaluations open a fresh connection

### Requirement: Scratch garbage collection is an orphan janitor
The grader SHALL periodically sweep its scratch directory and delete only entries older than a configured TTL, reusing a `datastore`-free eviction helper shared with `domain/datastore`. The TTL SHALL be much larger than the maximum evaluation duration so the sweep can never remove a scratch file belonging to a live evaluation. Normal cleanup SHALL remain explicit (owner deletes when done).

#### Scenario: Crash-orphaned file is reclaimed
- **WHEN** a platform instance crashes after a box run but before deleting its output identifier, leaving a scratch file older than the TTL
- **THEN** the grader's sweep deletes it

#### Scenario: Live file is never swept
- **WHEN** the sweep runs while an evaluation holds an in-use scratch identifier
- **THEN** that file is younger than the TTL and is not deleted

#### Scenario: Content-addressed caching stays out of scope
- **WHEN** implementing scratch for this change
- **THEN** identifiers remain UUIDv7 with explicit delete, and no content-hash dedup or LRU-as-lifetime GC is introduced (deferred to avoid the documented eviction TOCTOU that would require a lease protocol)
