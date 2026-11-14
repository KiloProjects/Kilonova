# Proposal

## Why

Runtime flags are only editable through one flat dump on `/admin`: every flag of a type, sorted by key, with no grouping by what it controls. That works, but it gets harder to use with every new flag, and nothing makes a new flag get a sensible place in the admin UI. Two concrete gaps came up while running separate Docker deployments: a flash announcement written for kilonova.ro (the Girls' Programming Camp banner) shows on every instance with no way to turn it off, and seeding a new container means re-entering every flag by hand.

## What Changes

- **Flash registry and masking.** Flashes (dismissable banners such as `gpc_announcement` and `contest_disclaimer`) are registered in code with a name and a description. A new list flag holds the names hidden on this instance. A hidden flash is not rendered at all; the browser-side dismissal in `flash.ts` keeps working for the rest.
- **Flashes admin page.** A new `/admin/flashes` tab lists every registered flash with a "hide on this instance" toggle.
- **Flag export and import.** The admin UI can download the instance's flag values as a JSON file and import such a file from another instance. Flags marked secret (OIDC keys, webhook URLs, payment webhook secrets) are left out of the export unless an "include secrets" checkbox is ticked. An import applies each value through the normal flag update path, so it is persisted, and reports unknown keys and values of the wrong type without stopping.
- **Grouped flag sections.** The admin UI gains sections that group flags by what they control (for example front page, submissions, donations, integrations). The existing flat dump stays as a fallback that still lists every flag.
- **Coverage test.** A test renders the grouped admin sections and fails when a registered flag has no place in them, so a new flag cannot ship without an admin UI home. Flags not yet moved into a group are listed in an explicit allowlist that can only shrink.

- **Flag edits move to the admin v2 API.** All flag editors, including the existing flat form, save through a new `PATCH /api/admin/v2/flags`. The legacy `POST /api/admin/updateFlags` is removed; the admin UI was its only user.

No breaking changes: flag keys and stored values are unchanged.

## Capabilities

### New Capabilities
- `flash-masking`: registry of flash announcements, the per-instance list of hidden flashes, server-side suppression, and the admin page that edits it.
- `flag-transfer`: exporting flag values to a JSON file and importing them on another instance, including secret marking and the opt-in for secrets.
- `admin-flags-ui`: grouped flag sections in the admin UI, the flat fallback, and the rule that every registered flag appears in a group or in the shrinking allowlist.

### Modified Capabilities
<!-- None: flag-store's persistence and update rules are reused unchanged. -->

## Impact

- `domain/config`: flag definitions gain a "secret" marker and a way to enumerate every definition regardless of type.
- `sudoapi/flags`: the new masked-flashes flag; secret flags marked.
- `web/ui/modals`, `web/ui/problems`, `web/ui/adminviews`, `web/assets/flash.ts`: flash registry, suppression, new admin pages and sections.
- `api`: new flag update, export and import operations on the admin v2 (huma) API; legacy `POST /api/admin/updateFlags` removed; no new endpoints on the legacy router.
- `web/assets/api/client.ts`: a `getCallV2` helper next to `postCallV2`.
- Tests: a new HTML-parsing test for admin coverage. `golang.org/x/net/html` is already an indirect dependency; goquery would be a new one (decided in design).
