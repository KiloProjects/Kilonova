# Changelog

## v0.25.2 (2025-05-10)

-   More zap -> slog conversions
-   Codebase-wide refactorings
-   Don't rely on zip package anymore for archive/test
-   Barebones external resources UI
-   Migrate more components to htmx
-   Autoformat LaTeX in Markdown editor
-   New LaTeX Math parser in Markdown (can now do `$1$ $2$` without issue)
-   Upgrade to tailwind v4
-   Start migrating html templates to [templ](https://templ.guide)
-   Add page for printing problem statements

## v0.25.1 (2024-12-11)

-   Latest problems view at the bottom of the page
-   Start using htmx for some page parts
-   Use slog context and move a lot of usages of zap over to slog
-   Grader page for compilation commands
-   Discord embeds for audit log
-   Initial linking with Discord integration
-   Slightly overhaul UI for adding tests
-   Updated README.md
-   User generation UI for admins
-   Contrib tool for contest user generation
-   Internal refactoring
-   Add english translations for emails
-   Add barebones support for PHP and Rust
-   Support for TWIPLA Analytics
-   Measure number of SQL queries for admins
-   Problem list filter for submissions
-   Form to view another user's progress checklist (exposes functionality already present)
-   Tool to warmup markdown statement cache
-   More semantic HTML + accessibility

## v0.25.0 (2024-05-18)

-   The number wasn't incremented, but many changes were made since January
-   List of changes by month:
    -   May
        -   Improve polygon archive parsing;
        -   Save history in problem search so pressing back in the browser goes to previous page/state;
        -   UI for downloading individual tests in archive generator;
        -   Bucket statistics improvements;
        -   Kotlin language support and setup script;
        -   !!! Huge refactoring of task execution, paving the way for remote graders and, finally, an official v1.0 release of the platform.
    -   April
        -   Better language matching for .cpp files;
        -   `zstd` compression in bucket data store;
        -   Custom statement types (for example `statement-ro-editorial.md`);
        -   Send submissions on Ctrl+Enter;
        -   Preference changes for default display for problem statements;
        -   Add eviction policies based on TTL and max file size for bucket files (in cache environments such as subtests);
            -   This deprecated and later removed a cron job that already tried to do this;
        -   Coalesce schema with existing migrations;
        -   !!! Automatic schema migrator.
    -   March
        -   Added variable number of max uses to contest invites;
        -   Created submission saver script, useful for on-prem post-contest requests;
        -   Changed the visibility requirements for problems to be added to virtual contests;
            -   For example, a problem from an official contest could, in the past, be added to a virtual contest and bypass the submission count limit.
        -   Significant submission visibility performance improvements;
        -   Better auto update for problem scores in problem sidebar;
        -   Display sandbox version on initialization;
        -   Huge performance improvements to calculating problem score by caching and recalculating on submission changes instead of storing in a view;
        -   Improve polygon archive importing;
        -   Require passwords for name change;
        -   !! Reorganize data storage (tests, subtests, etc) in an S3-like bucket system;
        -   Support for multiple submission languages to interactive problems;
        -   Experimental support for MOSS plagiarism checker.
    -   February
        -   API reorganization;
        -   Replace quickjs with pure go library goja for katex;
        -   Remove Code field in Submission struct;
        -   For non-default score scale, show percentage instead of points in multiple places (cosmetic change);
        -   More dynamic front page based on flags (for example, the subtitle can be changed);
        -   Problem IDs are now hidden in contest lists when user doesn't have edit rights;
        -   Show display name in leaderboard.csv;
        -   Remove misinterpretable characters from user account generation;
        -   Fix some polling issues that caused fail2ban triggers;
        -   Save session information such as user agent and ip, used mostly for official contests to protect against cheating.
    -   January
        -   Contest submission/question cooldown;
        -   Improved submissions page performance;
        -   Generated user account improvements like sending a mail with password;
        -   Score "scaling" feature (for example, you can have 1000 points show up in the contest leaderboard for a problem);
        -   Output only problem handling improvements;
            -   Variable max submission size;
            -   Default to file upload regardless of preference (avoids nasty lag when copy-pasting large text in codemirror).

## v0.24.0 (2024-01-13)

-   !!! Virtual Contests and better contest discovery mechanism;
-   !! Proper reevaluation queue;
-   Contest invitations system;
-   ICPC leaderboards go only after "finished"/"reevaling" submissions;
-   Compilation durations are now saved;
-   Better problem searching;
-   Better problem interface for tag pages;
-   Many bug fixes and performance improvements;
-   Admins can now force add users from the Contest UI (before it was only an API endpoint);
-   Can automatically create simulations of contests from problem lists;
-   Better problem list showing on the right of the problem page;
-   Per-minute submission limit (also reports fishy activity, that is, bypassing the limit, to webhook);
-   Show cumulative amount for monthly donations;
-   Misc:
    -   Initial, barebones user sessions page;
    -   Cgroups v2 support;
    -   Add og:image for better embeds of links;
    -   Convert more API endpoints to `webWrapper`;
    -   Can now resize image attachments using query parameters;
    -   Limit number of sessions per user;
    -   Mailer can now be disabled;
    -   Cache gravatars;
-   Administration and backend changes:
    -   Debug metrics and actions page;
    -   Bulk update of test visibility in problem list children problems;
    -   All flags are now visible in the Admin Panel;
    -   Can now block python requests;
    -   Compress tests and subtests;
    -   Cache checker compilations;
    -   Environment variable override for flags.json, for future Docker containerization;
    -   Update some endpoints for fail2ban support;
-   Internal refactoring:
    -   Refactor grader box/manager;
    -   Better task running;
    -   No more `GenContext` in code and `.Ctx` in templates.

## v0.23.0 (2023-10-28)

-   !! Much improved ICPC handling;
-   Add support for pascal and re-enable Go language support in grader;
-   Display names;
-   Problem import UI for proposers;
-   Login modal on front page option;
-   Glossary for frequently asked terms (for now, just stdin/stdout);
-   Add "Contests" button in navbar, enabled optionally;
    -   "Problems" button in navbar is now also toggleable.

## v0.22.0 (2023-09-27)

-   ICPC leaderboards for contests;
    -   Also includes a bit of a rework to how scores are handled.
-   New (for now hidden) query parameter for listing all problems inside a list;
-   Fixed bugs that didn't show fractional scores in leaderboards and in the notification when sending a submission;
-   Allow hiding problems from the trending tab.

## v0.21.0 (2023-09-14)

-   Rework front page;
-   Donations page:
    -   Actual page;
    -   Webhook notification for buymeacoffee;
    -   Manualy added donors.
-   Internal addition of sorting problems ascending/descending and by id, name and published time
    -   `/problems` UI will follow soon enough
-   Add button to make problem list from contest problems;
-   Performance improvements by materialization.

## v0.20.2 (2023-09-05)

-   You can now view a "checklist" summary for the progress inside a problem list;
-   Contests can now be deleted;
-   Add problem checklist API endpoint (not to be confused with list checklists);
-   Bug fixes.

## v0.20.1 (2023-08-21)

-   Visible problem tests toggle;
-   Advanced problem archive generator;
-   Allow updating single tests from files;
-   Fix subtask editor bug.

## v0.20.0 (2023-08-04)

-   !!! Decimal scores in submissions;
-   !! Allow changing usernames and permit admins to lockout people;
-   ! Allow problem editors to reevaluate/delete individual submissions;
-   ! CMS-style score parameters support when uploading test archives;
-   ! Polygon format;
-   Change submission viewing permissions (anyone can view, if option is toggled);
-   Move attachments and other files to `assets/` webserver route;
-   Bulk visibility updating dialog for problem lists;
-   Much improved configuration flag system;
-   Show problem source in search;
-   Huge database refactoring;
    -   Remove sqlx as a dependency;
    -   TODO: Refactor everything properly (tedious process).
-   Allow exporting editors in archive file;
-   Clean up `docs/`;

## v0.19.0 (2023-07-06)

-   !! Basic blog post support;
-   Proper tag filtering;
-   New format for the image extension in markdown;
-   Attachment upload improvements:
    -   Allow changing name before upload;
    -   Autocomplete flags for some file names.
-   Zip archive improvements:
    -   Hidden option to allow adding all submissions to archive;
    -   Tag support;
    -   Reorganization;
-   Log tag changes for better monitoring;
-   Rewrite filter/update queries in db/ in almost all places;
-   Adopt AGPL v3 license;
-   Problem checklist;
-   Various grader improvements and refactorings;
-   New logos (thanks Secret-chest for favicon);
-   Remove "orphaned" tests from database, since they are not needed anymore;
-   Allow public leaderboards on contests.

## v0.18.0 (2023-06-03)

-   !! Much better problem search;
    -   ~~Proper tag filtering is still missing for now~~ Fixed in 0.19.0.
-   Many improvements to KaTeX math server side rendering (SSR);
-   Remove highlightjs from application bundle (code blocks are now rendered server-side);
-   Markdown statements now require no browser javascript for proper rendering (CSS is required, though, for KaTeX expressions);
-   Audit logs are also sent to a discord webhook (and have more relevant information in them);
-   Add `contestant.txt` to checker environment, to verify contestant source code;
-   Add statement caching for increased load speeds;
-   Performance improvements.

## v0.17.0 (2023-05-20)

-   !! Problem tags;
-   Problem statistics;
-   UI changes;
-   Raise submission size limit to 30kb;
-   Internal changes (like starting to minimize `sqlx` usage);
-   Audit log pagination;
-   More preloading web stuff on the backend;
-   Big performance boosts for front page;
-   `grader.properties` improvements;
-   SSR for KaTeX math expressions;
-   Nicer confirmation dialogs;
-   Internal, for now: last updated date for attachments and by whom.

## v0.16.1 (2023-04-25)

-   Contest descriptions;
-   Warnings for contest problems once contest ended;
-   Various SQL improvements;
-   Bug fixes;

## v0.16.0 (2023-04-18)

-   Big UI redesign;
-   SEO optimizations;
-   All parent lists for a list will be shown when there are multiple parents;
-   Fix crash on go 1.21 nightly;
-   Update JS libraries;
-   Bug fixes;
-   Some small refactoring for the grader.

## v0.15.0 (2023-04-14)

-   BIG: added another scoring type: maximum of subtasks accross submissions;
-   Added new `<modal>` design for pop ups that will be slowly used in more and more places;\
-   Score breakdown for maximum subtasks scoring strategy;
-   Added position in contest leaderboards (along with links to problems);
-   Fixed annoying bug in statement page that reset content without warning.

## v0.14.2 (2023-04-09)

-   Show problem edit options on hover directly (on mobile, tap the "Edit Problem" link);
-   Make C++17 default preference for submitting;
-   Conditional automatic opening of editors list on problem statement page;
-   Some more paddings and margins for images and tables;
-   Add buttons for quick toggling problem visibility on problem lists (only shallow visibility toggle, it doesn't recurse).

## v0.14.1 (2023-03-28)

-   Added support for attachments inside test arhives;
-   Fix text casing.

## v0.14.0 (2023-03-28)

-   Dark mode toggle (instead of relying on browser preference);
-   Support for multiple languages in statements;
-   Show subtask tests in a better way on submission page;
-   Allow multiple parent problem lists to be showed on problem page (capped to 5);
-   More customizable preferences in cookies;
-   Fixed a few firefox autocomplete bugs;
-   CSS fixes.

## v0.13.2 (2023-03-25)

-   Attachment improvements:
    -   Editing/creating in UI;
    -   Renaming attachments in UI;
    -   `Exec` flag for attachments.
-   `grader.properties` improvements:
    -   Allow more information regarding problem (limits, authors, test name).

## v0.13.1 (2023-03-16)

-   More USACO contest work;
-   Proper tables in problem summaries;
-   Show sibling problems (from same problem list) in sidebar.
-   Allow styling CSS images.

## v0.13.0 (2023-03-04)

-   Major: Grader rework to support high-memory problems;
-   Better UX for USACO contests.

## v0.12.2 (2023-03-02)

-   USACO-style contests;
-   Add formatting for blockquotes in markdown;
-   Better submission hiding;
-   Bug fixes;

## v0.12.1 (2023-02-04)

-   Generated account support;
-   Functioning contest leaderboard;

## v0.11.1-v0.12.0 (2023-01-29)

-   Most of the contest functionality is done;
-   Future contests on main page;
-   New CMS-style default checker while maintaining legacy checker support;
-   Overhauled problem UI/UX;
-   UI/UX overhaul for lists of problems (such as the new attempted problems list on a profile page);
-   Better logging for grader;
-   Other small things such as more granular feature enabling/disabling from config.

## v0.11.0 (2022-11-29)

-   Simply mark new version since so many changes were made with 0.10
-   Start slowly working on support for proper contests, starting with the option to disable signup

## v0.10.x (2022-10-22)

-   Many frontend changes;
-   Backend cleanups;
-   Better problem access control;
-   Better submission memory handling;
-   Better profile page;
-   Broader test archive support;
-   Simply better in all ways;
-   Submission Pastes;
-   Forgot password forms;
-   Nested problem lists (also used them on the main page);
-   More restrictive usernames;
-   Button for reevaluating all submissions;
-   Simple barebones audit log;
-   Removed stack limit;
-   Removed sqlite db option;
-   Removed .kna support.

!!! info

    Changelog prior to v0.10 is available in the Git history.
