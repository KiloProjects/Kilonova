## ADDED Requirements

### Requirement: Remote scratch is served over SFTP
In remote mode the platform SHALL implement `eval.Scratch` over SFTP against the grader's `sshd`, without any FUSE / sshfs mount. The grader SHALL own the scratch directory on its local disk; the platform SHALL read and write scratch files over SFTP. All connections SHALL be initiated platform→grader.

#### Scenario: Platform uploads an input file
- **WHEN** the platform-side adapter calls `SaveFile` with a reader in remote mode
- **THEN** the file is written to the grader's scratch directory over SFTP and a UUIDv7 identifier is returned, without any privileged mount on the platform

#### Scenario: Platform downloads an output file
- **WHEN** the platform calls `ReadFile` with an identifier produced by a `RunBox3` response
- **THEN** the file bytes are streamed back over SFTP and the identifier can then be deleted via `DeleteFile`

### Requirement: Write ordering before RPC
The platform SHALL wait for the SFTP file `Close` to return before issuing a `RunBox3`/`RunMultibox3` request that references the written identifier, guaranteeing the bytes are on the grader's disk before it reads them locally.

#### Scenario: Input is durable before execution
- **WHEN** the platform writes an input scratch file and then requests a box run referencing it
- **THEN** the SFTP write has been flushed to the grader's disk before the run request is sent, so the grader's local read never races the transfer

### Requirement: SFTP connection failures are explicit and recoverable
The platform SHALL bound SFTP operations with a context deadline and SHALL redial via a connection pool on failure. A connection drop mid-evaluation SHALL surface as a Go error that fails the evaluation fast (eligible for retry), never as an indefinitely blocked operation.

#### Scenario: Grader restarts mid-evaluation
- **WHEN** the SFTP connection drops while an evaluation is reading or writing scratch
- **THEN** the operation returns an error within its deadline, the evaluation fails fast, and subsequent evaluations redial a fresh connection

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
