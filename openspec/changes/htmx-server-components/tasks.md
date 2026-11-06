# Tasks

## 1. Prerequisites

- [ ] 1.1 Add `ui.Modal(ctx, title, large, content...)` producing the out-of-band `#modals` dialog with `onclose="this.remove()"`; verify with a render test asserting `hx-swap-oob="beforeend:#modals"` and the close form
- [ ] 1.2 Switch `problempages.StatisticsModal` to `ui.Modal`; verify `/problems/1/statistics?hx_modal=true` still opens the modal and a second open leaves one dialog in `#modals`
- [ ] 1.3 Port `getGradient` to `ui.ScoreColor(score, max)` (same thresholds and colours); verify a table test covering 0, 20, 40, 60, 80, 99 and 100 percent
- [ ] 1.4 Render score-cell colours server-side as inline `background-color` in `web/ui/submissions/submission.go` and `web/ui/lists/lists.go`, dropping `score-cell`/`data-score` there; verify the submission page and checklist show the same colours as before with JS disabled
- [ ] 1.5 Add an `htmx:oobAfterSwap` → `bundled.initialLoad(target)` hook in the layout for the remaining (non-colour) `initialLoad` work; verify `data-paginate` buttons work inside an out-of-band fragment
- [ ] 1.6 Move `rezStr` from `web/ui/submissions` into package `ui` (e.g. `ui.ResultCount`); verify `go test ./web/ui/...` passes

## 2. Score breakdown

- [ ] 2.1 Add `sudoapi.MaxScoreBreakdown(ctx, viewer, problem, contest, userID)` with the editor/tester permission downgrade, contest scoping and freeze handling; verify with sudoapi tests for each scoring strategy and the downgrade case
- [ ] 2.2 Make `api.maxScoreBreakdown` a thin wrapper over 2.1; verify the JSON response is unchanged for a non-contest request
- [ ] 2.3 Add `submissions.ScoreBreakdown` view (unattempted, max-sub, ICPC table, sum-subtasks with `BreakdownMode`); verify render tests for all four cases
- [ ] 2.4 Add `GET /breakdown` to `problemRouter` rendering `ui.Modal` + the view; verify it works at `/problems/{id}/breakdown` and `/contests/{cid}/problems/{id}/breakdown`, and returns an error toast with no modal for anonymous users
- [ ] 2.5 Point the score links in the problem info sidebar, problem-list sidebar and contest sidebar at the route with `hx-get` + `hx-swap="none"`; update `summary_test.go` and contests tests to assert `hx-get`
- [ ] 2.6 Delete `maxscore_breakdown.tsx`, `sub_mgr.tsx` (move `icpcVerdictString` into `sub_waiter.ts`) and `initMaxscoreBreakdown`; verify `pnpm -C web/assets build` and a manual breakdown open in a browser with no console errors

## 3. Small server-side swaps

- [ ] 3.1 Replace `<kn-tag>` mounts on the tag pages with `problems.Tag` (keep `data-name` for the filter script); verify tag pages render tags with JS disabled
- [ ] 3.2 Move glossary definitions into `translations.toml` and serve them as `ui.Modal`s (or inline dialogs); delete `glossary.tsx`; verify the stdin and donation glossary links open their text
- [ ] 3.3 Replace `inviteQRCode` with a server modal showing the existing QR image route; verify from the contest edit page

## 4. Problem listings

- [ ] 4.1 Add `problems.ProblemTable` (name/id, publish badge, score, counts, source or tags column) with `ui.Paginator`; verify render tests for the latest, tag and profile variants
- [ ] 4.2 Render the latest-problems block, tag page list and profile solved/attempted lists with 4.1; profile lists paginate independently via their own query params; verify page 2 links and that `kn-pb-latest`, `kn-tag-pbs`, `kn-pb-solvedby`, `kn-pb-attemptedby` are gone from `web/ui`
- [ ] 4.3 Tag page "show tags" toggle as Alpine; verify toggling issues no request
- [ ] 4.4 Problem search: server-rendered results region and GET filter form (`hx-get`, `hx-select`, `hx-push-url`, `delay:250ms`, `hx-sync="this:replace"`); verify typing quickly ends on the right results and shared URLs reproduce filters
- [ ] 4.5 Wire the existing tag-group picker (unchanged, out of scope) to the search form through a hidden `tags` input it writes before dispatching `change`. Verify that simple and complex tag modes still filter.
- [ ] 4.6 Render the deep-list title server-side; verify no `/api/problemList/{id}` request from the search page

## 5. Contest live views

- [ ] 5.1 Server-rendered leaderboard (classic + ICPC, first solves, freeze/last-updated, CSV link, colours via `ui.ScoreColor`) with htmx reload and participants filter in the URL; editor cells open the breakdown (2.4); verify with render tests and with JS disabled
- [ ] 5.2 Leaderboard auto-refresh: `every 30s` + morph while running, 286 from the fragment once ended; verify a score change shows up within 30 s, scroll/filter survive, and polling stops after the end time
- [ ] 5.3 Server-rendered registrations with fuzzy filter, pagination and a kick POST route (editor-only) returning the list; verify filter, page 2 and kick with confirmation
- [ ] 5.4 Communication page: announcement and question lists as gomponents, author names loaded in one query, create/edit/delete/ask/answer routes returning the re-rendered list; verify each action in a browser
- [ ] 5.5 Periodic refresh of the communication lists (5 s editors / 10 s others) with the edit-mode guard; verify a half-typed answer survives a refresh and no refresh happens after the contest ends
- [ ] 5.6 Activity poller replacing `kn-comm-announcer` and `startReloadingQnA`: counts in the URL, `HX-Trigger` toast, 286 after contest end; verify badge + toast after posting an announcement from another session
- [ ] 5.7 Delete the leaderboard, registrations, communication and announcer components from `contest.tsx` and the QnA functions from `api/contest.ts`; verify `pnpm -C web/assets build` and a browser pass over all contest tabs

## 6. Client-side cleanups

- [ ] 6.1 Navbar menus and dropdown in Alpine; delete `navbar.ts`; verify mobile menu, dropdown, Escape and outside-click
- [ ] 6.2 Attachment renamer in Alpine; delete `att_renamer.tsx`; verify renaming an attachment on problem and blog editors
- [ ] 6.3 Add a `KnElement` Lit base class that renders into light DOM and move `server-timestamp` onto it. Verify that timestamps on the audit log and footer render unchanged, and that a Tailwind class used inside a `KnElement` applies.
- [ ] 6.4 Submission waiter as an htmx status poller with toast and 286 stop; delete `sub_waiter.ts`; verify submitting shows the final-score toast and refreshes older submissions

## 7. Client components in Lit

- [ ] 7.1 Port `kn-contest-countdown` to Lit, keeping its attributes (`target_time`, `type`) and the reload at zero, and adding the `noreload` attribute for the registrations table. Verify that the countdowns on the home page, contest sidebar and invite page tick and reload as before.
- [ ] 7.2 Port `upload-progress` (`common.tsx` `ProgressChecker`) to Lit. Verify that uploading a test archive shows the progress toast.
- [ ] 7.3 Port the `confirm()` dialog to a Lit element rendering a native `<dialog>` with `ui.Modal`'s markup, keeping the `htmx:confirm` bridge and the Promise API used by inline scripts. Verify that deleting an attachment and kicking a registration both ask for confirmation.
- [ ] 7.4 Remove the Preact code made dead by this change: `components/index.ts` exports for ported components, the `preact/debug` import, and every `.tsx` except the tag picker (`tags.tsx`) and the parts of `modal.tsx` and `common.tsx` it still imports. Verify that `pnpm -C web/assets build` succeeds, that the tag picker still works on the problem edit page and the search page, and that a headless-Chrome pass over all pages shows no console errors.

## 8. Sweep

- [ ] 8.1 Add the `behavior.api.legacy_reads_require_auth` flag (default off) and an `s.authIfFlag(flag)` middleware wrapping `MustBeAuthed`. Apply it to `POST /problem/search`, `GET /submissions/getByID`, `GET /problemList/{id}/` and every route in design.md's "Further reads behind the same flag" table. Verify with an api test that walks the gated routes: with the flag on, an anonymous call gets a 401 and a logged-in call succeeds; with it off, both succeed.
- [ ] 8.2 Comment out the route registrations of the retired endpoints in design.md's table, with a "may return in /api/v2" note. For `/user/byID`, comment out only `userRouter.Get("/")`. Mark the now-unused handlers `//nolint:unused`. Verify:
  - `go build ./... && golangci-lint run ./api/...` is clean;
  - a grep of `web/assets` and `web/ui` finds no caller of any retired path;
  - the moderation, `deauthAll` and `setBio` POSTs still work from the profile page.
- [ ] 8.3 Switch the flag's default to on, since §4 and §6.4 have moved the remaining frontend callers to logged-in or server-side paths. Verify with the flag on:
  - `/problems` search works logged out, through the server-rendered search;
  - a logged-out headless-Chrome pass over the public pages shows no 401 in the network log.
- [ ] 8.4 Remove dead exports from `components/index.ts`, `common.tsx` leftovers (`Paginator`, `BigSpinner`, `rezStr`), `getGradient` and the `.score-cell` pass in `initialLoad` once nothing emits `score-cell`, and `js-base64` if no `enc=` attributes remain; verify `pnpm -C web/assets build` and compare bundle size before/after
- [ ] 8.5 Full browser sweep of every page touched (headless Chrome console check, no JS errors); update CLAUDE.md's web section if conventions changed
