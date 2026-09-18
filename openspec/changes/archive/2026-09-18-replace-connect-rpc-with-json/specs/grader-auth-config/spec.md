## MODIFIED Requirements

### Requirement: Authenticated, single-direction transport
Every request SHALL be authenticated by a single grader-minted bearer token presented over TLS in the `Authorization: Bearer` header. One HTTP middleware on the grader listener SHALL enforce this for every path — the `/run/*` and `/languages` control endpoints and the `/scratch/{id}` data endpoint alike — and SHALL attach the authenticated client name to the request context for attribution. The grader SHALL initiate no connections back to the platform and SHALL hold no platform credentials. Operators SHALL treat the token as insufficient on its own and MUST additionally restrict grader network reachability to the platform (segmentation / IP allowlist).

#### Scenario: Request with a valid token is served
- **WHEN** a platform presents a registered token over TLS on any grader path
- **THEN** the grader authenticates the client, records its name for the request, and serves the request

#### Scenario: Request with an invalid or missing token is rejected
- **WHEN** a caller presents no token or an unregistered token on any grader path
- **THEN** the grader responds `401 Unauthorized` without executing any sandbox or touching scratch, and the platform client surfaces a distinguishable authentication error

#### Scenario: Grader holds no reverse credentials
- **WHEN** a sandbox escape compromises the grader host
- **THEN** the attacker finds no platform DB DSN, datastore credentials, or outbound platform connections on that host
