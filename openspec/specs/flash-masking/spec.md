# flash-masking Specification

## Purpose
Lets an instance hide specific flash announcements (dismissable banners) that do not apply to it, without a code change or a new build, and gives admins one page to see and toggle them.

## Requirements

### Requirement: Flashes are registered with a name and description
Every flash announcement the platform can render SHALL be registered once, with a unique name (the same name used for browser-side dismissal) and a human-readable description of what it announces and where it appears. A flash that is not registered SHALL NOT be renderable.

#### Scenario: Known flashes are listed
- **WHEN** the platform starts with the Girls' Programming Camp banner and the contest disclaimer in code
- **THEN** both are registered, as `gpc_announcement` and `contest_disclaimer`, each with a description

#### Scenario: Duplicate names are rejected
- **WHEN** two flashes are registered with the same name
- **THEN** the build's tests fail

### Requirement: An instance can hide flashes
Each instance SHALL keep a persisted list of hidden flash names, stored like any other runtime flag and empty by default. A hidden flash SHALL NOT be present in the rendered HTML for any visitor, whether or not they dismissed it. A flash that is not hidden SHALL behave as before: shown until the visitor dismisses it in their browser. A name in the list that matches no registered flash SHALL be ignored.

#### Scenario: Hidden flash is not rendered
- **WHEN** `gpc_announcement` is in the hidden list and anyone opens the home page
- **THEN** the page HTML contains no element for that flash

#### Scenario: Visible flash keeps browser dismissal
- **WHEN** `contest_disclaimer` is not hidden and a visitor who dismissed it earlier opens a contest problem
- **THEN** the flash is in the HTML and stays hidden for that visitor, as today

#### Scenario: Stale name is harmless
- **WHEN** the hidden list contains a name that is no longer registered
- **THEN** pages render normally and the name is ignored

### Requirement: Admins manage hidden flashes on a dedicated page
The admin area SHALL have a flashes page, reachable from the admin tab bar and only by admins, listing every registered flash with its name, description and whether it is hidden on this instance. Toggling a flash SHALL update the hidden list through the normal flag update path, so the change persists and takes effect on the next page load. If persisting fails, the page SHALL report the error.

#### Scenario: Admin hides a flash
- **WHEN** an admin toggles `gpc_announcement` to hidden
- **THEN** the hidden list contains it, the home page stops rendering it, and the setting survives a restart

#### Scenario: Non-admin is refused
- **WHEN** a logged-in non-admin opens the flashes page or submits a toggle
- **THEN** the request is refused and the hidden list is unchanged
