# Spec Delta

## Purpose

Defines how the platform and grader processes behave as supervised, disposable workloads: which signals they honour, how they shut down, what an orchestrator can probe to decide they are healthy, and where their logs go.

## ADDED Requirements

### Requirement: Both processes shut down on SIGTERM and SIGINT
`kn main` and `kn grader-serve` SHALL begin an orderly shutdown on SIGTERM as well as on SIGINT. Shutdown SHALL stop accepting new work, wait for in-flight work to finish up to a bounded grace period, and then exit zero. Exceeding the grace period SHALL log what was still running and exit anyway. Neither process SHALL depend on SIGKILL to terminate.

#### Scenario: Container stop is graceful
- **WHEN** the runtime sends SIGTERM to a running platform container
- **THEN** the process finishes in-flight HTTP requests, closes the database pool, and exits zero well within the runtime's default kill timeout

#### Scenario: Grader stops on SIGTERM
- **WHEN** the grader container receives SIGTERM
- **THEN** its listener stops accepting requests and the process exits zero

#### Scenario: Shutdown drains rather than aborting instantly
- **WHEN** a request is in flight at the moment SIGTERM arrives
- **THEN** that request is allowed to complete and its client receives a normal response, not a dropped connection

#### Scenario: Hung work does not hang the container
- **WHEN** work is still running when the grace period expires
- **THEN** the process logs the outstanding work and exits without waiting further

### Requirement: Both processes expose an unauthenticated health endpoint
`kn main` and `kn grader-serve` SHALL each serve `GET /healthz` on their main listener, returning `200` with a short plain-text or JSON body once the process is ready to serve, and a non-`2xx` status when it is not. The endpoint SHALL require no authentication, SHALL be excluded from the grader's bearer-token middleware, SHALL NOT leak configuration, version-unrelated internals or credentials, and SHALL answer quickly enough to be polled every few seconds. For the platform, readiness SHALL include a bounded-timeout database check; for the grader, it SHALL include that the sandbox is usable. During shutdown the endpoint SHALL report unhealthy before the listener stops.

#### Scenario: Healthy platform answers
- **WHEN** the platform has connected to the database and started serving
- **THEN** `GET /healthz` returns `200`

#### Scenario: Unreachable database is reported
- **WHEN** the database becomes unreachable
- **THEN** `GET /healthz` returns a non-`2xx` status within its timeout instead of hanging

#### Scenario: Grader health needs no token
- **WHEN** `GET /healthz` is requested from the grader without an `Authorization` header
- **THEN** it returns `200` and is not rejected as unauthorized, while every other path still requires a valid token

#### Scenario: Draining instance is taken out of rotation
- **WHEN** the platform has received SIGTERM and is draining
- **THEN** `GET /healthz` reports unhealthy while in-flight requests still complete

### Requirement: Logs go to standard output, file logging is optional
Both processes SHALL write their logs to standard output as their primary destination. The rotating log file under `<KN_DATA_DIR>/logs` SHALL be controlled by `KN_LOG_FILE`, defaulting to enabled so that existing host deployments are unaffected, and the shipped container images SHALL set it to disabled. With file logging disabled, no log directory SHALL be created and stdout SHALL carry the full log stream at the configured level. Log destinations SHALL be independent of the OpenTelemetry exporter, which stays governed by `KN_OTEL_ENABLED`.

#### Scenario: Container logs are visible to the runtime
- **WHEN** a container from either image is running
- **THEN** `docker logs` shows startup and request logs with no configuration beyond the image defaults

#### Scenario: File logging off creates no directory
- **WHEN** a process starts with `KN_LOG_FILE=false`
- **THEN** it does not create `<KN_DATA_DIR>/logs` and does not fail if that path is not writable

#### Scenario: Host deployments keep their log file
- **WHEN** an operator starts the binary directly without setting `KN_LOG_FILE`
- **THEN** the rotating file under `<KN_DATA_DIR>/logs` is written as before
