# Proposal

## Why

Every page is now a gomponents view, but much of what users see is still drawn a second time in the browser by Preact. The server already holds the data, base64-encodes it into an `enc` attribute, and a component re-renders it (often fetching it again on mount). That doubles the rendering code, ships the 68 KB translation table and a 207 KB app bundle for read-only tables, and leaves some views with real bugs: the max-score breakdown disagrees with the header inside contests, the communication page makes one user request per question, and problem search fires an uncancelled request per keystroke. Now that `web/ui` exists, these views can move server-side with htmx and keep Alpine for the bit of local state they need.

## What Changes

- **Max-score breakdown** becomes a server-rendered modal (`GET …/problems/{id}/breakdown`), reusing the submission page's subtask/test views (`BreakdownMode`). `maxscore_breakdown.tsx` and `sub_mgr.tsx` are deleted. Breakdown logic moves from `api/problem.go` into sudoapi.
- **Contest leaderboard, registrations and communication** become gomponents views with htmx fragment endpoints; the topbar's "new" badge becomes a single htmx poller that toasts via `HX-Trigger`. Their Preact components in `contest.tsx` and the QnA polling in `api/contest.ts` are deleted.
- **Problem listings** (latest, tag page, profile solved/attempted) become a server-rendered table with server pagination; **problem search** moves results, filters and pagination server-side (`hx-get` + `hx-push-url`, debounced), keeping the tag-group picker client-side.
- **Small swaps:** `kn-tag` → `problems.Tag`; glossary and invite-QR → static server modals; navbar and attachment renamer → Alpine; submission waiter → htmx poller.
- **Shared prerequisites:** a `ui.Modal` helper (the out-of-band `#modals` pattern), score colouring for out-of-band swaps, `rezStr` promoted into `ui`.
- **Stays client-side, rewritten in Lit:** the contest countdown, upload progress and the `confirm()` dialog. `server-timestamp` is already Lit. Session, toast, flash and time handling stay as plain TS.
- **Out of scope: the tag picker** (`selectTags`, the problem tag editor and the tag chips inside them). It needs a full overhaul whose shape is still open, so it gets its own change. It stays as it is, in Preact, until then. That change decides whether it ends up in Lit or on the server, and removes Preact.
- **BREAKING (`/api`)**:
  - **Retired routes:** `problem/{id}/maxScoreBreakdown`, `GET /user/byID/{id}/`, and these contest routes: `leaderboard`, `registrations`, `kickUser`, `announcements`, `questions`, `allQuestions`, `{create,update,delete}Announcement`, `askQuestion` and `answerQuestion`. Each registration is commented out, not deleted, so the route can return in `/api/v2`.
  - **Kept behind an auth flag:** `POST /problem/search`, `GET /submissions/getByID` and `GET /problemList/{id}/` are gated by a runtime flag that requires authentication. The flag is on by default once the frontend no longer calls them anonymously.

## Capabilities

### New Capabilities
- `score-breakdown`: who may view a user's max-score breakdown, what it shows per scoring strategy, and how it is scoped to a contest.
- `contest-live-views`: leaderboard, registrations, and contest communication (announcements, questions, the "new" indicator) — their contents, permissions and refresh behaviour.
- `problem-listings`: problem tables and problem search — filtering, pagination, URL-addressable state.

### Modified Capabilities
<!-- none: no existing spec covers the web UI -->

## Impact

- **Code:** `web/assets/components/*` (most of `contest.tsx`, `problem_search.tsx`, `maxscore_breakdown.tsx`, `sub_mgr.tsx`, `glossary.tsx`, `att_renamer.tsx`, parts of `modal.tsx`/`common.tsx`), `web/assets/{navbar,lit,sub_waiter,util}.ts`, `web/assets/api/contest.ts`; views in `web/ui/{contests,problems,problempages,lists,tags,userviews,home,editors,navbar}`; new routes in `web/web.go`; new sudoapi methods (score breakdown, leaderboard/registration loaders).
- **APIs:** see the BREAKING item above; the per-endpoint table is in design.md.
- **Dependencies:** `lit` stays and is the library for new client components. Preact shrinks to the tag picker only and is removed by the tag picker overhaul, not here. `js-base64` is removed if no `enc=` attributes remain.
- **Bundle:** the app bundle shrinks; most of the translation table no longer needs to ship to every page.
