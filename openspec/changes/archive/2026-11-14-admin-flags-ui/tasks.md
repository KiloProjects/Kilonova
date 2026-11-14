# Tasks

## 1. Flag registry groundwork (domain/config)

- [x] 1.1 Add `GenSecretFlag` and `Secret() bool` on `Flag[T]` (D1); verify with a `config_v2_test.go` case that a secret flag reports `true` and a `GenFlag` one `false`
- [x] 1.2 Add registry `Names()` returning every registered key sorted; verify with a test that registers two flags and gets both back in order
- [x] 1.3 Refactor `Update` onto an internal set-and-persist path that takes the flagset explicitly (D2); verify the existing config tests still pass under `go test -race ./domain/config/`
- [x] 1.4 Add `Flagset.Export(includeSecrets)` encoding only explicitly set keys; verify with tests for: default-valued flag absent, secret excluded without opt-in, secret included with opt-in
- [x] 1.5 Add `Flagset.Import(ctx, map)` returning applied / skipped-with-reason / persistence errors; verify with tests for unknown key, wrong type (skipped, others applied), absent flag untouched, values persisted to a memStore
- [x] 1.6 Mark `oidc.rsaKey.private`, `oidc.rsaKey.id`, `oidc.cryptoKey`, `admin.important_webhook`, `admin.verbose_webhook`, `frontend.donation.bmac_webhook_secret` with `GenSecretFlag`; verify with a test in `sudoapi/flags` asserting all six report secret

## 2. Admin v2 flag endpoints and export/import UI

- [x] 2.1 Add `PATCH /api/admin/v2/flags` taking the five typed maps and returning per-flag errors (D4). Handlers read the flagset from the request context only, so they can be tested without a database. Verify with a `humatest` test (operations registered without the auth middleware, a memStore-backed flagset in context) that a valid update persists and an unknown key is reported
- [x] 2.2 Add `GET /api/admin/v2/flags/export` (`?secrets=true` opt-in) returning the `{"version":1,"flags":{...}}` envelope; verify with a humatest case for secret filtering
- [x] 2.3 Add `POST /api/admin/v2/flags/import` (1 MiB cap, rejects an unknown version before applying anything; huma rejects invalid JSON); verify with humatest cases for a bad version, invalid JSON, and a partial file whose result lists applied, skipped and applied-secret keys
- [x] 2.4 Register the three operations in `AdminV2()` behind the existing `MustBeAdminV2` middleware; verify on a dev instance that a non-admin session is refused by each with 403
- [x] 2.5 Remove the legacy `POST /api/admin/updateFlags` route and handler from `api/api.go`/`api/admin.go`, moving `updateFlagMap` into the v2 handler; verify `grep -rn updateFlags api web` finds nothing but the v2 code and `go build ./...` passes (do this together with 4.2 so the flat form never points at a missing endpoint)
- [x] 2.6 Add `getCallV2` to `web/assets/api/client.ts` next to `postCallV2`; verify `pnpm -C web/assets build` succeeds
- [x] 2.7 Add export (with an "include secrets" checkbox, fetch + `Blob` download) and import (file picker, result list) controls to the admin panel; verify by exporting from a dev instance, importing into a fresh database, and seeing the values after a restart

## 3. Flash registry and masking

- [x] 3.1 Create `web/ui/flashes` with the unexported-field `Flash` type, `GPC` and `ContestDisclaimer` entries with descriptions, `All()`, and `Section(ctx, f, ...)` (D5); verify with a test that names are unique and that `Section` renders nothing for a hidden name
- [x] 3.2 Add the `frontend.flashes.hidden` `[]string` flag (default empty) and move `modals.GPCAnnouncement` and `problems.ContestDisclaimer` onto `flashes.Section`; verify view tests: the home page contains `data-flashname="gpc_announcement"` by default and doesn't when it's hidden
- [x] 3.3 Add the `/admin/flashes` page and tab with one toggle per registered flash, saving through `PATCH /api/admin/v2/flags` (D6); verify that a non-admin gets refused, and that toggling GPC on a dev instance hides it from the home page and survives a restart

## 4. Grouped admin settings

- [x] 4.1 Add the group data structure and the `/admin/settings` page and tab rendering groups with typed inputs carrying `data-flag` and `data-type` (D7); verify with a view test that a group renders one input per listed flag with the right type
- [x] 4.2 Replace the per-container collection in `flags.js` with a shared collector over `[data-type]` inputs in a form (D7), used by both the flat form and the settings page, both sending to `PATCH /api/admin/v2/flags`, with per-flag errors shown; verify on a dev instance by editing one flag of each type in the flat form and a donations flag in the settings page, saving, and reloading
- [x] 4.3 Fill the first groups (Donations, Front page, Navbar & footer); verify the settings page shows them
- [x] 4.4 Wrap the flat `/admin` flags form in a closed `<details>`, and move secret flags into a nested closed `<details>` within it; verify with a view test that the form sits inside a `<details>` without `open`, every secret flag's input is inside the nested one, no secret input is outside it, and every flat-form input carries `data-type`; then verify on a dev instance that editing a secret flag in the collapsed section still saves
- [x] 4.5 Add the coverage test with `golang.org/x/net/html` (made a direct dependency) and the `waitingForGroup` list holding every not-yet-grouped flag (D8); verify it passes, then confirm it fails when a new unlisted flag is added, when a waiting flag is also grouped, and when a flag is rendered twice

## 5. Wrap-up

- [x] 5.1 Add translation keys for new UI text to `translations.toml` in `en` and `ro`, run `go generate ./...`; verify there are no missing-key warnings while rendering the new pages
- [x] 5.2 Run `go vet ./...`, `go test -race ./...` and `golangci-lint run`; verify all are clean
