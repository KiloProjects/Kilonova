# Spec Delta

## MODIFIED Requirements

### Requirement: Authenticated, single-direction transport
Every request SHALL be authenticated by a single grader-minted bearer token presented over TLS in the `Authorization: Bearer` header. One HTTP middleware on the grader listener SHALL enforce this for every path — the `/run/*` and `/languages` control endpoints and the `/scratch/{id}` data endpoint alike — and SHALL attach the authenticated client name to the request context for attribution. The grader SHALL initiate no connections back to the platform and SHALL hold no platform credentials. Operators SHALL treat the token as insufficient on its own and MUST additionally restrict grader network reachability to the platform (segmentation / IP allowlist).

The platform MAY skip verification of the grader's certificate when `KN_SANDBOX_ALLOW_INSECURE` is enabled, for local and development deployments where the grader presents a self-signed certificate. This SHALL disable certificate verification only: the connection SHALL still be TLS and the bearer token SHALL still be sent and checked. It SHALL apply identically to the control plane and the `/scratch` data plane, so the two can never disagree about whether the grader is verified. The platform SHALL log a warning naming the grader endpoint at startup whenever verification is disabled. Because this removes the platform's only assurance that it is talking to the real grader, and a bearer token intercepted in transit is arbitrary code execution, it SHALL be documented as unsuitable for production.

#### Scenario: Request with a valid token is served
- **WHEN** a platform presents a registered token over TLS on any grader path
- **THEN** the grader authenticates the client, records its name for the request, and serves the request

#### Scenario: Request with an invalid or missing token is rejected
- **WHEN** a caller presents no token or an unregistered token on any grader path
- **THEN** the grader responds `401 Unauthorized` without executing any sandbox or touching scratch, and the platform client surfaces a distinguishable authentication error

#### Scenario: Grader holds no reverse credentials
- **WHEN** a sandbox escape compromises the grader host
- **THEN** the attacker finds no platform DB DSN, datastore credentials, or outbound platform connections on that host

#### Scenario: Self-signed grader is usable locally
- **WHEN** the platform runs with `KN_EVAL_MODE=remote` and `KN_SANDBOX_ALLOW_INSECURE` enabled against a grader with a self-signed certificate
- **THEN** both submission evaluation and scratch file transfer succeed, and startup logs a warning that grader certificate verification is disabled

#### Scenario: Verification is on by default
- **WHEN** the platform runs in remote mode without `KN_SANDBOX_ALLOW_INSECURE`
- **THEN** a grader presenting an untrusted certificate is rejected on both the control plane and the scratch data plane, and the failure is reported as a TLS error rather than an authentication error

#### Scenario: The token is still required
- **WHEN** certificate verification is disabled and a request carries no valid bearer token
- **THEN** the grader still responds `401 Unauthorized`
