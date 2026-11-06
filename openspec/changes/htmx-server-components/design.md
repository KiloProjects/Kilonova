# Design

## Context

All pages are gomponents views under `web/ui`; handlers live in `web/*_pages.go` and load data into params structs (see CLAUDE.md). The frontend (`web/assets`, Vite, three IIFE bundles) still mounts Preact custom elements inside those views, usually fed by `enc="<base64 JSON>"` attributes built with `ui.EncodeJSON`. Patterns this design reuses already exist:

- **Fragment on the same route:** `problemSubmissions` returns only the viewer node when `isHTMXRequest(r)` (`web/problem_pages.go`), rendered with `rt.componentModal`; `modals.OlderSubmissions` polls it with `hx-get` + `hx-select`.
- **Server modal:** the statistics tab uses `hx-get=".../statistics?hx_modal=true" hx-swap="none"`; the response is `<div hx-swap-oob="beforeend:#modals"><dialog class="modal-container">…` (`problempages.StatisticsModal`). `#modals` is in the layout and a `MutationObserver` in `modal.tsx` calls `showModal()` on added dialogs.
- **Server toasts:** `setHTMXToast` sets `HX-Trigger: {"showToast": …}`; `toast.ts` listens.
- **Already-ported views:** `submissions.subTask` (with the kept `BreakdownMode`), `testTable`, `ui.Paginator`, `ui.Score`, `problems.Tag`, `ui.ServerTimestamp`.

Mount points today: breakdown links in `problems/summary.go`, `problempages/statement.go`, `contests/contests.go`; `kn-leaderboard`/`kn-contest-registrations` in `contests/view.go`; `kn-announcements`/`kn-question-mgr`/`kn-questions` in `contests/communication.go`; `kn-comm-announcer` and `startReloadingQnA` in `problems/topbar.go`; `kn-pb-latest` in `home/home.go`; `kn-tag-pbs`/`kn-tag` in `tags/tags.go`; `kn-pb-solvedby`/`kn-pb-attemptedby` in `userviews/userviews.go`; `kn-pb-search` in `lists/lists.go`; `kn-glossary` in `static/donate.go` and `problempages/statement.go`; `kn-att-name` in `editors/attachments.go`; `inviteQRCode` in `contests/edit_tables.go`; `makeSubWaiter` in `problempages/submit.go`; `NavBarManager` in `navbar/navbar.go`.

The full per-component research (APIs called, translation keys, line-level sketches) is in the source plan this change was derived from; this document keeps the decisions.

## Goals / Non-Goals

**Goals:**
- Render every read-mostly, fetch-and-render component on the server, behind the same routes the pages already use.
- One modal mechanism, one fragment mechanism, one polling mechanism across the site.
- Delete the corresponding TS/TSX and the JSON calls that only they made.

**Non-Goals:**
- Moving upload progress, the contest countdown or `confirm()` to the server. They stay client-side, but are rewritten in Lit (Decision 9).
- **The tag picker** (`selectTags` and its modal, `kn-pb-tag-edit`, the `TagView` chips inside it). It needs a full overhaul, which belongs to its own change. That change decides the picker's interaction model, whether it is client-side (Lit) or server-side, and whether `GET /tags/` must stay public, and it removes Preact. This change leaves the picker working as it is: the search form only depends on it through the hidden `tags` input (Decision 7), and the tag editor on the problem edit page keeps its mount.
- Server-sent events / websockets for live contest data (polling stays).
- Redesigning any page's look; markup and classes carry over.
- The `/api/v2` surface: unchanged here.

## Decisions

1. **Same route, fragment when htmx.** Pages keep one URL; the handler renders the full layout normally and only the swappable region when `isHTMXRequest`. Clients use `hx-select` on that region's id. *Alternative:* dedicated `/fragment/...` routes — more routes and permission checks to duplicate, and no no-JS fallback. Write actions (kick, announcement CRUD, ask/answer) get small POST/DELETE routes under the page's path that return the re-rendered region.

2. **`ui.Modal` for every server modal.** A gomponents helper producing the existing out-of-band `#modals` markup with `onclose="this.remove()"`. Statistics migrates to it; breakdown, glossary and invite-QR use it. *Alternative:* inline hidden `<dialog>`s next to each link — fine for static glossary text, but breakdown needs per-click data; one mechanism is simpler to reason about.

3. **Score colours are computed on the server.** Port `getGradient` (`web/assets/util.ts`) to `ui.ScoreColor(score, max)` and have every server-rendered score cell carry its `background-color` as an inline style, instead of emitting `.score-cell` + `data-score` for `initialLoad` to colour. Today that is `web/ui/submissions/submission.go` and `web/ui/lists/lists.go`; every new view in this change (breakdown, leaderboard, problem tables) uses it from the start. Server-rendered content then looks right in fragments, out-of-band modals and without JS. `getGradient` and the `.score-cell` pass in `initialLoad` stay only for what is still rendered client-side (e.g. the submission waiter's toast until 6.4), and are deleted in the sweep once nothing emits `.score-cell`. An `htmx:oobAfterSwap` → `initialLoad` hook is still added for the non-colour parts of `initialLoad` (`data-paginate`, flashes). *Alternative:* keep colouring in JS and only add the OOB hook — smaller, but every fragment keeps depending on a client pass and flashes uncoloured.

4. **Breakdown logic moves to sudoapi.** `api/problem.go`'s handler body (permission downgrade, contest max score, per-strategy selection) becomes `sudoapi.MaxScoreBreakdown(ctx, viewer, problem, contest, userID)`, returning the submission id explicitly. It takes the contest so the best-submission and subtask queries can be contest-scoped and freeze-aware (fixing the header/body disagreement — see specs/score-breakdown). The view lives in `web/ui/submissions` because `subTask`/`testTable` are unexported there; it renders ICPC breakdowns with the ICPC test table. Route: `GET /breakdown` inside `problemRouter`, so it works at `/problems/{id}/breakdown` and `/contests/{cid}/problems/{id}/breakdown`; `?user_id=` optional.

5. **Leaderboard auto-refresh.** While the contest is running the leaderboard region polls itself (`hx-trigger="every 30s"` plus the manual reload button) with `hx-swap="morph:outerHTML"` so scroll position and the filter survive; the fragment response returns status 286 once the contest has ended, which stops polling. Viewers who cannot see past the freeze keep getting the frozen board, so polling during a freeze is cheap and leaks nothing. *Alternative:* manual reload only (today's behaviour) — rejected per maintainer.

6. **Polling with htmx, state in the URL.** The "new activity" poller re-renders itself with the last-seen counts in its own `hx-get` URL, so the server stays stateless; increases set an `HX-Trigger` toast; the endpoint answers 286 (htmx's stop-polling status) once the contest has ended. Communication lists poll with `hx-swap="morph:outerHTML"` (morph is already vendored) and skip a tick while any item is in edit mode (`hx-trigger="every Ns [!document.querySelector('[data-editing]')]"`), so typed text is never swapped away. *Alternative:* rely on morph preserving Alpine state — unverified, so the explicit guard is used.

7. **Problem search: a GET form.** Filters become a plain `<form hx-get="/problems" hx-target hx-select hx-push-url="true">`; the text box triggers on `input changed delay:250ms`; htmx's request queueing (`hx-sync="this:replace"`) aborts stale requests. The tag-group picker stays the existing widget, out of scope here, and writes a hidden `tags` input and dispatches `change`. That hidden input is the only contract with it, so the overhaul can replace the picker without touching the search form. *Alternative:* keep the Preact search and only add debounce/abort — fixes the race but keeps double rendering.

8. **Alpine for purely local UI.** Navbar menus, the attachment rename toggle, the tag page's show-tags column and per-item edit toggles in communication use Alpine (`x-data`/`x-show`), not htmx.

9. **Lit, not Preact, for what stays client-side.** Lit is already in the bundle for `server-timestamp`, and the TS config already enables the decorators it needs. Client components are custom elements, which Lit builds natively, while Preact needs the `preact-custom-element` wrapper and a separate JSX toolchain. Three of the remaining Preact pieces are ported to Lit:
   - `kn-contest-countdown`
   - `upload-progress`
   - the `confirm()` dialog

   After this change, Preact holds only the tag picker (see Non-Goals). Its removal, together with `preact-custom-element` and the JSX config, belongs to the tag picker overhaul.

   **Shared base class.** The components extend `KnElement`, a small Lit base class that renders into light DOM (`createRenderRoot() { return this; }`), so the site's Tailwind classes and `.btn`/`.form-input` styles apply without copying CSS into shadow roots. Translations come from the existing `getText`.

   **Dialogs.** `confirm()` renders a native `<dialog>` with the same markup as `ui.Modal`, so client-made and server-made modals look the same. `KNModal` stays only as long as the tag picker uses it.

   *Alternatives considered:*
   - Keep Preact for these three: two component models for small widgets that Lit already covers.
   - Plain vanilla custom elements: workable for the countdown, but the base class gives all three the same lifecycle and translation handling for little cost.

## Risks / Trade-offs

- [Polling swaps clobbering in-progress edits] → explicit edit-mode guard on the trigger (Decision 5) plus a scenario in specs/contest-live-views.
- [Search feels slower: local state → debounce + round trip] → 250 ms debounce, `hx-sync` replace, server already does one round trip per keystroke today.
- [External consumers of JSON endpoints] → endpoints are only removed after confirming no external use; otherwise kept as thin wrappers over the new sudoapi methods.
- [Large leaderboards as HTML, now refreshed every 30 s] → HTML is not larger than JSON + client render in practice; morph swap keeps scroll position; polling stops at contest end (286). If big contests make it heavy, cache the rendered board per (contest, freeze-visibility, filter) for the poll interval.
- [Breakdown scoping changes numbers users see in contests] → this is the bug fix; numbers now match the leaderboard header. Called out in release notes.

## Migration Plan

Ship in phases, each independently deployable and revertible by reverting its commit (old Preact components stay until their replacement lands):

0. Prerequisites: `ui.Modal`, OOB colouring hook, `rezStr` → `ui`.
1. Breakdown (+ delete `sub_mgr.tsx`), `kn-tag`, glossary, invite QR.
2. Problem tables, then problem search.
3. Leaderboard, registrations, communication + activity poller.
4. Alpine swaps (navbar, attachment renamer) and the submission waiter. Countdown, upload progress and `confirm()` are ported to Lit.
5. Sweep: delete dead TSX/TS (everything except the tag picker), the `api/contest.ts` QnA functions, and `js-base64` if no `enc=` remains; retire the JSON endpoints below and turn on the auth flag.

## JSON endpoints losing their last frontend caller

After this change nothing in `web/assets` or `web/ui` calls these (checked by grepping both trees, including inline scripts). Maintainer decision per endpoint:

| Endpoint (`/api` prefix) | Route (`api/api.go`) | Only caller today | Freed by | Decision |
|---|---|---|---|---|
| `GET /problem/{id}/maxScoreBreakdown` | `s.maxScoreBreakdown` | `maxscore_breakdown.tsx` | §2 | retire |
| `GET /contest/{id}/leaderboard` | `s.contestLeaderboard` | `contest.tsx` `ContestLeaderboard` | §5.1 | retire |
| `GET /contest/{id}/registrations` | `s.contestRegistrations` | `contest.tsx` `ContestRegistrations` | §5.3 | retire |
| `POST /contest/{id}/kickUser` | `s.stripContestRegistration` | `contest.tsx` `ContestRegistrations` | §5.3 | retire |
| `GET /contest/{id}/announcements` | `s.contestAnnouncements` | `api/contest.ts` | §5.4–5.6 | retire |
| `GET /contest/{id}/questions` | `s.contestUserQuestions` | `api/contest.ts` | §5.4–5.6 | retire |
| `GET /contest/{id}/allQuestions` | `s.contestAllQuestions` | `api/contest.ts` | §5.4–5.6 | retire |
| `POST /contest/{id}/createAnnouncement` | `s.createContestAnnouncement` | inline script, `contests/communication.go` | §5.4 | retire |
| `POST /contest/{id}/updateAnnouncement` | `s.updateContestAnnouncement` | `contest.tsx` | §5.4 | retire |
| `POST /contest/{id}/deleteAnnouncement` | `s.deleteContestAnnouncement` | `contest.tsx` | §5.4 | retire |
| `POST /contest/{id}/askQuestion` | `s.askContestQuestion` | inline script, `contests/communication.go` | §5.4 | retire |
| `POST /contest/{id}/answerQuestion` | `s.answerContestQuestion` | `contest.tsx` | §5.4 | retire |
| `GET /user/byID/{id}/` | `userRouter.Get("/")` | `client.getUser` | §5.4–5.6 | retire (only this GET) |
| `POST /problem/search` | `s.searchProblems` | `problem_search.tsx` | §4 | keep, auth-gated |
| `GET /submissions/getByID` | `s.getSubmissionByID()` | `sub_waiter.ts` | §6.4 | keep, auth-gated |
| `GET /problemList/{id}/` | `s.getProblemList` | `problem_search.tsx` (deep-list title) | §4.6 | keep, auth-gated |

**Retire** means commenting out the route registration in `api/api.go`, with a comment that it may come back in `/api/v2`. The handler functions stay, so restoring one is a one-line change. Once a route is commented out its handler becomes unused, so each one gets a `//nolint:unused // route retired, may return in /api/v2` marker instead of being deleted. Only the `GET /` on the `/user/byID/{cUID}` mount is retired. The same mount carries `moderation/*`, `deauthAll` and `setBio`, which are still used, and `/user/byName/...` is unaffected.

**Auth-gated** means a new runtime flag, `behavior.api.legacy_reads_require_auth` (bool, declared with `config.GenFlag` in `sudoapi/flags/backend_flags.go`, so it is editable from the admin panel). When it is on, every auth-gated route (the three above and the reads listed below) goes through `s.MustBeAuthed` and an anonymous call gets the usual 401. When it is off, the route behaves as today. It defaults to **off**: until §4 lands, anonymous visitors on `/problems` still search through `POST /problem/search`, so gating earlier would break search for logged-out users. The sweep turns the default on once nothing anonymous calls these routes. Gating is a small middleware, `s.authIfFlag(flag)`, wrapping `MustBeAuthed`, so other routes can opt in later.

**Further reads behind the same flag.** After this change these unauthenticated reads have no anonymous caller left in `web/assets` or `web/ui`. Their remaining callers are logged-in pages, editors or admins, or there is no caller at all. They go behind `authIfFlag` as well:

| Route (`/api` prefix) | Caller after the change |
|---|---|
| `POST /problem/get` (a read, despite the POST) | none |
| `GET /problem/{id}/` | none |
| `GET /problem/{id}/maxScore` | statement page refresh after a submit (logged in) |
| `GET /problem/{id}/languages` | none |
| `GET /problem/{id}/tags` | problem tag editor (editors) |
| `GET /problem/{id}/get/attachments` | none |
| `GET /problem/{id}/get/attachment/{aID}`, `attachmentByName/{name}` | statement/attachment editors |
| `GET /blogPosts/fromUser`, `/blogPosts/bySlug`, `/blogPosts/{id}/` | none |
| `GET /blogPosts/{id}/get/attachments`, `attachment/{aID}`, `attachmentByName/{name}` | blog editors |
| `GET /submissions/get` | none (the submissions pages render on the server) |
| `GET /paste/{id}/` | none |
| `GET /tags/getByID`, `/tags/getByName` | none |
| `GET /user/{self,byName/{name}}/` | admin roles panel; checklist "view as user" form, which is already hidden for visitors |
| `GET /user/…/solvedProblems` | none |
| `GET /problemList/filter`, `/problemList/byName` | none |
| `GET /contest/{id}/` | contest edit script (editors) |
| `GET /contest/{id}/problems` | none |

**Not gated:**
- `GET /tags/`: the tag picker on the search page loads it for logged-out visitors. It stays public until the tag picker overhaul decides how the picker gets its tags.
- `/auth/captcha`: signup needs it.
- `/webhook/discord_callback`: the OAuth callback has no session.
- Routes that already need a login or role:
  - `get/checklist`, `get/accessControl` (editor-only since commit 65b061e6), `/contest/{id}/invitations`, `checkRegistration`, `problemRemainingCount`, `/admin/*`.

**Still used after the change, so not retired:**
- `/contest/{id}/register` and `startRegistration` (the sidebar buttons)
- the leaderboard CSV route in `api/assets.go`
- `/tags/*`, `/problem/{id}/tags` and `/problem/{id}/update/tags` (the tag picker and tag editor)
- `bulkUpdateAttachmentInfo` (the attachment editors)
- `/problemList/create` and `/problemList/{id}/{update,delete,toggleProblems}`
- the `/user/byID/{id}/{moderation/*,deauthAll,setBio}` POSTs
