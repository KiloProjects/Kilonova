# Spec Delta

## Purpose

Makes runtime flags editable from admin sections grouped by what they control, while keeping a complete fallback list, and guarantees that every new flag gets a place in the admin UI.

## ADDED Requirements

### Requirement: Flags are editable in grouped sections
The admin area SHALL present flags in sections grouped by function (for example front page, submissions, donations, integrations), each showing the flag's human name and an input suited to its type. Editing a flag in a grouped section SHALL go through the normal flag update path, with the same persistence and error reporting as the existing flags form. Each flag SHALL appear in at most one grouped section.

#### Scenario: Grouped edit persists
- **WHEN** an admin changes a donations flag in the donations section and saves
- **THEN** the new value is active and survives a restart

#### Scenario: Failed save is reported
- **WHEN** saving a grouped section fails to persist a flag
- **THEN** the admin sees an error naming that flag

### Requirement: The flat flag list stays as a collapsed fallback
The existing complete list of all flags, organised by value type, SHALL remain available to admins and SHALL keep listing every registered flag, including ones that also appear in a grouped section. The list SHALL be collapsed when the page loads and expand on request. Within it, secret flags SHALL be kept apart from the others in their own section, which is also collapsed by default, so that secret values are not on screen until an admin deliberately opens that section.

#### Scenario: Every flag is reachable
- **WHEN** an admin expands the fallback list
- **THEN** every registered flag is listed there and can be edited

#### Scenario: Fallback starts collapsed
- **WHEN** an admin opens the admin page
- **THEN** the fallback list is collapsed and no flag inputs from it are visible

#### Scenario: Secrets need a second expansion
- **WHEN** an admin expands the fallback list but not its secrets section
- **THEN** non-secret flags are visible and no secret flag value is visible

### Requirement: Every flag has a home in the grouped UI
The build SHALL include a check that renders the grouped admin sections and fails when a registered flag appears in neither a grouped section nor an explicit list of flags still waiting to be grouped. The check SHALL also fail when a flag is both in a grouped section and on the waiting list, or appears in more than one grouped section, so the waiting list only shrinks and stays accurate.

#### Scenario: New flag without a section fails the build
- **WHEN** a developer adds a flag and puts it neither in a section nor on the waiting list
- **THEN** the check fails and names the flag

#### Scenario: Grouping a flag requires removing it from the waiting list
- **WHEN** a flag on the waiting list is added to a grouped section
- **THEN** the check fails until the flag is removed from the waiting list

#### Scenario: Duplicate placement fails
- **WHEN** a flag is rendered in two grouped sections
- **THEN** the check fails and names the flag
