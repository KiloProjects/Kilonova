# Spec Delta

## Purpose

Lets an admin copy runtime flag settings from one instance to another, typically to seed a new Docker deployment, without re-entering them by hand and without leaking secrets by accident.

## ADDED Requirements

### Requirement: Flags can be marked secret
A flag definition SHALL be able to declare itself secret. At least these flags SHALL be secret: the OIDC private key, key ID and crypto key, the important and verbose admin webhook URLs, and the Buy Me a Coffee webhook secret.

#### Scenario: OIDC key is secret
- **WHEN** the flag definitions are inspected
- **THEN** `oidc.rsaKey.private`, `oidc.rsaKey.id`, `oidc.cryptoKey`, `admin.important_webhook`, `admin.verbose_webhook` and `frontend.donation.bmac_webhook_secret` report as secret

### Requirement: Admins can export flags to a file
An admin SHALL be able to download the instance's flags as a single JSON file. The file SHALL contain a format version and, for every flag explicitly set on this instance, its key and current value; flags still at their compiled-in default SHALL NOT be included. Secret flags SHALL be left out unless the admin explicitly opts in when exporting. Non-admins SHALL NOT be able to export.

#### Scenario: Default export leaves secrets out
- **WHEN** an admin exports without opting in to secrets
- **THEN** the file holds every non-secret flag set on this instance and no secret flag

#### Scenario: Opt-in export includes secrets
- **WHEN** an admin exports with the "include secrets" option
- **THEN** secret flags set on this instance are included as well

#### Scenario: Defaults are not pinned
- **WHEN** `feature.platform.signup` has never been changed on this instance
- **THEN** the export does not contain it, so an instance importing the file keeps following the default

### Requirement: Admins can import an exported file
An admin SHALL be able to upload a file produced by the export. Each known flag in the file whose value decodes into the flag's type SHALL be applied through the normal flag update path, so it persists like an admin edit. Keys that match no registered flag, and values of the wrong type, SHALL be skipped and reported back by key, without stopping the rest of the import. A file with an unknown format version, or that is not valid JSON, SHALL be rejected as a whole and change nothing. Flags absent from the file SHALL keep their current values. The response SHALL list what was applied, what was skipped and why, and any persistence failures.

#### Scenario: Seeding a new container
- **WHEN** an admin imports a file exported from another instance into a fresh container
- **THEN** every flag in the file takes the exported value, and the values survive a restart

#### Scenario: Partial problems do not block the rest
- **WHEN** the file contains a key removed in this build and an integer flag holding `"abc"`
- **THEN** both are reported as skipped and all other flags in the file are applied

#### Scenario: Garbage file changes nothing
- **WHEN** an admin uploads a file that is not JSON or has an unsupported format version
- **THEN** the import is rejected with an error and no flag changes

#### Scenario: Unlisted flags are untouched
- **WHEN** the file does not contain `frontend.navbar.branding`
- **THEN** that flag keeps the value it had before the import
