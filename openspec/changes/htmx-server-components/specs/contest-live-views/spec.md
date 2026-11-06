# Spec Delta

## Purpose

Defines the contest views that change while a contest runs — leaderboard, registrations, announcements and questions, and the "new activity" indicator — including who sees what and how they refresh.

## ADDED Requirements

### Requirement: Leaderboard renders on the server and reloads in place
The leaderboard SHALL be rendered as part of the page response for anyone allowed to view it (the existing leaderboard visibility rules apply unchanged). It SHALL show classic or ICPC columns according to the contest, highlight first solves, show the last-updated time and the freeze time when one is set, and offer a CSV download. While the contest is running the leaderboard SHALL refresh itself automatically (every 30 seconds) without a full page navigation, keeping the viewer's scroll position and filter, and SHALL stop refreshing once the contest has ended. Reloading manually or changing the participants filter (all / official / unofficial, shown only when the contest has generated accounts and the advanced filter is enabled) SHALL update the table without a full page navigation and SHALL be reflected in the URL.

#### Scenario: Filter by official participants
- **WHEN** a viewer picks "official" in the participants filter
- **THEN** the table shows only non-generated accounts and the URL carries the filter, so reloading the page keeps it

#### Scenario: Automatic refresh during the contest
- **WHEN** a contestant's score changes while a viewer has the leaderboard open on a running contest
- **THEN** within 30 seconds the viewer's table shows the new score, without the page reloading or losing its scroll position

#### Scenario: Refresh stops at the end
- **WHEN** the contest's end time passes with the leaderboard open
- **THEN** the leaderboard stops polling after its next refresh

#### Scenario: Frozen leaderboard keeps refreshing safely
- **WHEN** the leaderboard is frozen and a non-editor has it open
- **THEN** refreshes keep showing the frozen standings

#### Scenario: No JavaScript
- **WHEN** the page is loaded with JavaScript disabled
- **THEN** the leaderboard table is still visible, with score cells coloured

### Requirement: Registrations are filterable, paginated and kickable by editors
Contest editors SHALL see registrations 50 per page, filterable by username (fuzzy), with the registration count, the individual start time for per-user-timed contests, and a kick action that asks for confirmation. Filtering and kicking SHALL update the list without a full page navigation. Non-editors SHALL NOT reach the page.

#### Scenario: Fuzzy filter
- **WHEN** an editor types "alex" in the filter
- **THEN** after a short pause only matching registrations are shown and the count updates

#### Scenario: Kick
- **WHEN** an editor confirms kicking a user
- **THEN** the user disappears from the list and a success toast is shown

### Requirement: Communication lists stay current without losing edits
The communication page SHALL show announcements to everyone who can view the contest, all questions to editors, and a participant's own questions to that participant, each question showing its author's name. Editors SHALL be able to create, edit and delete announcements and answer or re-answer questions; participants SHALL be able to ask questions. While the contest is running the lists SHALL refresh periodically (every 5 seconds for editors, 10 for others). A refresh SHALL NOT discard text the user is currently typing into an edit or answer form.

#### Scenario: Editor answers while new questions arrive
- **WHEN** an editor is halfway through typing an answer and a new question is asked
- **THEN** the new question appears and the half-typed answer is still there

#### Scenario: Ended contest
- **WHEN** the contest has ended
- **THEN** the lists are shown but no periodic refresh happens

### Requirement: New activity is announced on every contest tab
On every contest tab other than the communication page, while the contest is running, the platform SHALL indicate new announcements, new questions (for editors) and new answers (for the asking participant) that appeared since the page was loaded, with a "new" badge on the communication tab and a toast linking to the communication page. Checking for activity SHALL cost at most one request per refresh interval and SHALL stop once the contest ends.

#### Scenario: Announcement while on a problem page
- **WHEN** an editor posts an announcement while a contestant is on a contest problem
- **THEN** within one refresh interval the contestant sees the "new" badge and a toast

#### Scenario: Contest ends while open
- **WHEN** the contest's end time passes while a tab is open
- **THEN** the tab stops polling for activity
