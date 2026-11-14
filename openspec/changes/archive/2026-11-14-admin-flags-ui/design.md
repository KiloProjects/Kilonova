# Design

## Context

- Flag definitions are registered in a process-wide registry in `domain/config` (`GenFlag`); values live in a per-instance `Flagset`. A flagset only holds values that were loaded from the store or set at runtime, so "explicitly set" is exactly the set of keys in it. Reads go through `Get(ctx)` or `Of(fs)`, writes through `Update(ctx, v) error`.
- The admin flags form (`web/ui/adminviews/pages.go` `Panel` plus `scripts/flags.js`) posts `{bool_flags, string_flags, int_flags, int_array_flags, str_array_flags}` to the legacy `POST /api/admin/updateFlags`, which joins per-flag errors. `flags.js` is its only caller.
- `/api/admin/v2` (huma) already exists with session/OAuth/API-key auth (`SetupSessionV2`, `MustBeAdminV2`) and one operation, `generateUser`, called from the admin UI through `postCallV2` in `web/assets/api/client.ts`. v2 authenticates through the `Authorization` header carrying the session ID, not the session cookie.
- Flashes are two hand-written components (`modals.GPCAnnouncement`, `problems.ContestDisclaimer`) with a `data-flashname` attribute. They start hidden by CSS, and `web/assets/flash.ts` reveals them unless dismissed in `localStorage`.
- Requirements are in `specs/flash-masking`, `specs/flag-transfer` and `specs/admin-flags-ui`.

## Goals / Non-Goals

**Goals:**
- One generic path for type-erased flag access (enumerate, encode, decode-and-update), used by export, import and the coverage test.
- Grouped sections that reuse the existing update endpoint and input renderers, so adding a group is mostly data.

**Non-Goals:**
- Moving every flag into a group in this change. A few groups prove the shape, and the rest go on the waiting list.
- Bespoke widgets (problem-list pickers, previews). Groups start with the generic inputs; specialized widgets can replace them later without touching the coverage rule.
- Renaming flag keys or migrating stored rows.
- Showing flashes to some visitors but not others. Hiding is all-or-nothing per instance.

## Decisions

**D1. Secret marking is a separate constructor, `config.GenSecretFlag`.** It is the same as `GenFlag` but marks the definition secret, and `Flag[T]` gains `Secret() bool`. The alternatives were a variadic options argument on `GenFlag` or a field set after construction. A second constructor touches only the six secret declarations, keeps the other ~60 unchanged, and is impossible to forget halfway.

**D2. Type-erased access lives on the registry and `Flagset`.**
- The registry gains `Names()`: all registered keys, sorted.
- `Flagset` gains `Export(includeSecrets bool) map[string]json.RawMessage`, which encodes only keys present in the flagset, and `Import(ctx, map[string]json.RawMessage) ImportResult`.
- `Import` decodes each value with the definition's existing `decode` and then goes through the same set-and-persist path `Update` uses, but on the given flagset instead of the one in `ctx`. `Update(ctx, v)` becomes a thin wrapper over that path.

Alternative considered: putting export and import in `api`, using `GetFlags[T]` per type. That would mean five copies of the same loop, and `api` can't see which flags are explicitly set.

**D3. The file format is a small versioned envelope:** `{"version": 1, "flags": {"<key>": <value>, ...}}`, with values in the same JSON encoding as the `flags` table. Any other `version` is rejected before anything is applied. The version makes later format changes detectable.

**D4. All new endpoints go on `/api/admin/v2` (huma); the legacy `/api/admin` router gets nothing new.**
- `GET /flags/export?secrets=true` returns the envelope as the JSON response body.
- `POST /flags/import` takes the envelope as the request body, capped at 1 MiB by huma's body limit, and returns `{applied: [...], skipped: {key: reason}, errors: [...], applied_secrets: [...]}`.
- `PATCH /flags` takes the same five typed maps as the legacy `updateFlags` and applies them through `Update`, returning per-flag errors. Every flag editor saves through it: the flat form, the flashes page and the settings page.

The legacy `POST /api/admin/updateFlags` is removed in this change. Its only caller is `flags.js`, which moves to `PATCH /flags`. Its `updateFlagMap` helper moves with it into the v2 handler.

The handlers take the flagset from the request context, which the instance middleware sets, rather than from `s.base`. That way they can be tested with `humatest` and a memStore-backed flagset, without a database. The admin check is not per operation: `MustBeAdminV2` applies to the whole admin v2 router.

Because v2 authenticates by header, a plain `<a href>` download can't carry the session. The export button fetches through a new `getCallV2` helper (the GET counterpart of `postCallV2`) and saves the JSON with a `Blob` and an object URL. The same header-based auth also means CSRF doesn't apply to these operations.

Alternative considered: the legacy router, which takes cookie sessions and so would allow a direct download link. Rejected: v2 is where admin tooling is meant to grow, and it gives typed input and output plus OpenAPI for free.

**D5. Flash registry is a package with unexported fields.**
- A new `web/ui/flashes` package declares `type Flash struct{ name, description string }` with getters, a package-level value per flash (`GPC`, `ContestDisclaimer`), and `All()`.
- Because the fields are unexported, no other package can create a flash, so an unregistered one can't be rendered (spec: "SHALL NOT be renderable").
- Rendering goes through `flashes.Panel(ctx, f, children...)` (not `Section`, which would clash with the dot-imported `html.Section`). It returns an empty `Group` when `f` is in `frontend.flashes.hidden` (empty rather than nil, so callers can render it directly), and otherwise emits the `data-flashname` section it emits today. `flashes.DismissButton(f)` renders the matching close button. The two existing components move to call both.
- `flash.ts` is unchanged.
- A test checks that names in `All()` are unique.

**D6. Hidden list is a plain list flag, `frontend.flashes.hidden` (`[]string`, default empty).**
- The flashes page (`/admin/flashes`, a new admin tab) renders one toggle per `flashes.All()` entry.
- Saving sends the new list to `PATCH /api/admin/v2/flags` as `str_array_flags`.
- Names in the flag that aren't in the registry are ignored at render time and dropped the next time the page saves.

**D7. Grouped sections are data in `web/ui/adminviews`.**
- A group is `{TitleKey string, Flags []string}` (keys). The settings view renders each group with the existing input renderers chosen by the flag's type, and every input carries `data-flag="<key>"` and `data-type="<bool|string|int|int_array|str_array>"`.
- A new `/admin/settings` tab hosts the groups. The flat form stays on `/admin` as the fallback, wrapped in a closed `<details>`/`<summary>`. Inside it, secret flags (D1) are taken out of the per-type sections and rendered in a nested closed `<details>` of their own. Native `<details>` needs no JavaScript, and collapsed inputs stay in the form.
- Today `flags.js` collects inputs from fixed per-type containers (`boolFlagsDiv`, `stringFlagsDiv`, ...). Secret inputs moved out of those would silently stop being saved. So every flag input on both pages carries `data-type`, and one shared collector walks all `[data-type]` inputs inside a form and builds the typed-maps body. The flat form and the settings page both use it, so neither depends on where an input sits.
- Both the settings page and the flat form send the collector's body to `PATCH /api/admin/v2/flags`.
- First groups: Donations, Front page, Navbar & footer. Everything else starts on the waiting list. Secret flags stay out of grouped sections for now (the Buy Me a Coffee webhook secret waits instead of joining Donations), so the grouped page never shows a secret value outside a collapsed section.

Alternative: replacing the flat form on `/admin` in place. That's riskier while groups cover few flags, and the spec requires the flat list to keep existing anyway.

**D8. The coverage test uses `golang.org/x/net/html`, not goquery** (confirmed). The test only needs every `data-flag` attribute value, which is a ~15-line token walk. `x/net` is already in the module graph as an indirect dependency, so it only moves to direct. goquery would add two new dependencies (goquery, cascadia) for one selector.

The test:
- renders the settings view and the flashes page (which owns `frontend.flashes.hidden`) with `uitest.Ctx`;
- collects the `data-flag` values;
- compares them against `config` registry `Names()` and a `waitingForGroup` list kept in the test file.

It fails on any flag missing from both, any flag in both, and any duplicate `data-flag`. The test blank-imports `sudoapi/flags` so the registry is populated.

## Risks / Trade-offs

- **[Import of secrets copies an OIDC identity]** → Secrets are only in a file when the exporter opted in. The import response lists applied secret keys separately, so the admin sees it happened.
- **[Import is not atomic]** → Values are applied one by one. A database failure midway leaves the earlier flags applied, and every failure is reported by key. Since import is idempotent, re-running it is safe.
- **[Long waiting list at first, ~60 flags]** → Intended. It's a visible to-do list that can only shrink, and the test forces that.
- **[Hidden flashes are still in the JavaScript and CSS bundle]** → Harmless: no markup is rendered, so nothing can be revealed.
- **[A flag edited in both the grouped page and the flat list at the same time]** → Last write wins, same as today with two admins on the flat form.

## Migration Plan

Removing `POST /api/admin/updateFlags` affects nothing outside this repo: the admin UI was its only user. Rolling back restores it together with the old `flags.js`.

No data migration. The new `frontend.flashes.hidden` flag defaults to empty, so every flash keeps showing until an admin hides it. Rollback to the previous build ignores the new flag row (unknown keys are kept and ignored by the flag store).
