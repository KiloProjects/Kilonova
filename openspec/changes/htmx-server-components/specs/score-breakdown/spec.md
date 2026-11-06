# Spec Delta

## Purpose

Defines the max-score breakdown: the view explaining how a user's best score on a problem is made up, who may see it, and how it is scoped when opened from a contest.

## ADDED Requirements

### Requirement: Breakdown opens as a modal from any score link
Wherever a user's best score on a problem is shown as a breakdown link (problem info sidebar, problem-list sidebar, contest sidebar, and editor cells on the contest leaderboard), activating it SHALL open a modal titled with the score-breakdown title, rendered by the server, without a full page navigation. Closing the modal SHALL remove it, so opening breakdowns repeatedly does not accumulate hidden dialogs.

#### Scenario: Opening from the problem sidebar
- **WHEN** a logged-in user clicks their score on a problem page
- **THEN** a modal shows "Problem <name>: <score>" and the breakdown body, and the URL does not change

#### Scenario: Repeated opens
- **WHEN** the user opens and closes the breakdown five times
- **THEN** at most one breakdown dialog exists in the page at any time

### Requirement: Breakdown content follows the scoring strategy
For a never-attempted problem the breakdown SHALL say it was not attempted. For max-submission problems it SHALL name the best submission (linked) and show its subtasks and test table. For ICPC problems it SHALL do the same, with the test table in ICPC form (verdicts, no per-test score column). For sum-of-subtasks problems it SHALL list every subtask with its best score and link, per subtask, the submission that score came from.

#### Scenario: Sum of subtasks
- **WHEN** a user's best subtask 1 came from submission 12 and subtask 2 from submission 15
- **THEN** subtask 1 links "#12" and subtask 2 links "#15"

#### Scenario: Not attempted
- **WHEN** the user has no submissions on the problem
- **THEN** the modal shows the header with "-" and the not-attempted message

#### Scenario: ICPC problem
- **WHEN** the problem uses ICPC scoring
- **THEN** the test table shows verdicts and no score column

### Requirement: Only editors and testers see other users' breakdowns
A viewer SHALL see the breakdown of another user only if they are an editor or tester of the problem (or, when opened from a contest, an editor or tester of the contest). Any other request for another user's breakdown SHALL silently show the viewer's own breakdown instead. An anonymous viewer SHALL receive an error and no modal.

#### Scenario: Contestant asks for someone else
- **WHEN** a contestant requests the breakdown with another user's id
- **THEN** they receive their own breakdown

#### Scenario: Contest editor on the leaderboard
- **WHEN** a contest editor clicks a participant's cell on the leaderboard
- **THEN** that participant's breakdown is shown

### Requirement: Contest breakdowns are scoped to the contest
When opened from a contest, the header score, the chosen best submission and every subtask score SHALL all consider only submissions made in that contest, and SHALL respect the contest's leaderboard freeze for viewers who cannot see past it. The header and the body SHALL never disagree.

#### Scenario: Better score outside the contest
- **WHEN** a user scored 100 on a problem in the archive but 40 during the contest
- **THEN** the breakdown opened from the contest shows 40 and the 40-point submission

#### Scenario: Frozen leaderboard
- **WHEN** the leaderboard is frozen and a contestant improved their score after the freeze
- **THEN** a non-editor viewer sees the pre-freeze score in both header and subtasks
