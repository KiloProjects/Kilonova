# flag-store Specification

## Purpose

Defines where runtime-editable flags are persisted now that the platform must run as a disposable container: the database-backed flag store, its load and update semantics, the one-time import of a legacy flags file, and how processes without a database behave.

## Requirements

### Requirement: Runtime flags are persisted in the database
Runtime-editable flags SHALL be stored in the platform's PostgreSQL database, one row per flag keyed by its dotted internal name, holding a JSON-encoded value. The platform SHALL load stored values into the in-memory flag registry once at startup, after the database connection is established and schema migrations have run, and before the web server accepts requests. A flag with no stored row SHALL keep its compiled-in default. A stored row whose value does not decode into the flag's type SHALL be logged and skipped, leaving the default in place, and SHALL NOT abort startup. A stored row whose key matches no registered flag SHALL be logged and ignored, and SHALL NOT be deleted.

#### Scenario: Stored flags are applied at boot
- **WHEN** the database holds `frontend.default_language = "ro"` and the platform starts
- **THEN** the flag registry reports `ro` before the first request is served

#### Scenario: Unset flag keeps its default
- **WHEN** no row exists for `feature.platform.signup`
- **THEN** the flag reads as its compiled-in default

#### Scenario: Corrupt stored value is survivable
- **WHEN** the row for an integer flag holds `"abc"`
- **THEN** startup logs a warning naming the flag, the flag keeps its default, and the platform starts normally

#### Scenario: Unknown key is preserved
- **WHEN** the table holds a key for a flag that no longer exists in this build
- **THEN** startup logs it, ignores it, and leaves the row in place so a rollback still sees it

### Requirement: A flag update writes exactly that flag
When a flag is updated at runtime — from the admin UI or any other caller — the platform SHALL persist that single flag's new value to its own row, SHALL NOT rewrite unrelated flags, and SHALL NOT write any file. A failed write SHALL be logged and SHALL NOT crash the process or revert the in-memory value.

#### Scenario: Admin edit survives a restart
- **WHEN** an admin changes a flag in the admin UI and the container is replaced
- **THEN** the new container starts with the changed value

#### Scenario: Concurrent edits do not clobber each other
- **WHEN** two flags are changed at nearly the same time by different requests
- **THEN** both values are persisted; neither overwrites the other with a stale value

#### Scenario: No file is written
- **WHEN** a flag is updated
- **THEN** no file on disk is created or modified, and a read-only filesystem outside the data volume does not break the update

#### Scenario: Write failure is non-fatal
- **WHEN** the database rejects the write
- **THEN** an error is logged, the process keeps running, and the new value remains active until the next restart

### Requirement: A legacy flags file is imported once
On startup, when the flag table is empty and the file named by `KN_FLAGS_PATH` exists and parses, the platform SHALL import its entries into the flag store, log what it imported, and SHALL NOT write to that file afterwards. On every subsequent startup the file SHALL be ignored, including when the table has been deliberately emptied back to defaults; the import SHALL be recorded so that it happens at most once per database. A missing, empty or unparsable file SHALL be reported and SHALL NOT prevent startup.

#### Scenario: First boot imports the file
- **WHEN** a pre-existing `flags.json` is present and the database has no flags yet
- **THEN** its values are visible in the admin UI after startup and the file is left byte-identical on disk

#### Scenario: Second boot ignores the file
- **WHEN** the same platform is restarted after an admin has changed a flag
- **THEN** the admin's value wins and the file's value is not re-applied

#### Scenario: No file is fine
- **WHEN** the container has no flags file at all
- **THEN** startup proceeds on compiled-in defaults with no error

### Requirement: Per-process flag overrides still work without the database
The `KN_FLAG_OVERRIDES` variable SHALL continue to override flag values for a single process. Overrides SHALL be applied after values are loaded from the store, SHALL NOT be persisted back to it, and SHALL apply even in a process that has no database connection. A process that consumes no flags — `kn grader-serve` — SHALL start and run correctly with no flag store and no database.

#### Scenario: Override beats the stored value
- **WHEN** the store holds `behavior.db.log_sql = false` and the process starts with `KN_FLAG_OVERRIDES=behavior.db.log_sql=true`
- **THEN** SQL logging is on for that process only, and the stored value is unchanged

#### Scenario: Grader needs no flag store
- **WHEN** `kn grader-serve` starts with no `KN_DB_DSN` set
- **THEN** it starts normally and never attempts to reach a flag store
