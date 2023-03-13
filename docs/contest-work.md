Kilonova will soon be used for organizing an official contest. In order to achieve this, multiple features need to be implemented.

High priority:
- [x] Contest system:
	- [x] Problem visibility;
	- [x] Questions (Yes/No/No comment/etc);
	- [x] General announcements;
    - [x] Auto reload questions/announcements and notify from any page;
	- [x] Leaderboard;
        - [x] Real-time leaderboard for admins;
    - [x] Disable pastes for contest problems during contest;
        - Can disable pastes in general now.
- [x] Allow to disable contestant access to others' submissions (EDIT: now implicit, cannot be disabled);
- [x] Endpoint for account generation;
- [x] Vendor all dependencies

Medium priority:
- [x] Move default custom grading to cms format
    - Also have legacy grader
    - Rename all current grader attachments to grader-legacy
- [ ] Websockets for live updates.
    - [ ] Also remember to put the time there, since contest end/start may change
    - [ ] Or maybe polling
- [x] Third checkbox for attachments to include whether they should count towards grading or not;
- [ ] Make problem score be sum of best subtasks if they exist
- [x] Max number of submissions during contest;
- [x] Allow to disable manual signup;
- [x] Show max score on problem page;
- [ ] See which other features should be disabled during contests;
- [ ] Disable forgot password things for generated accounts;

Nice-to-have:
- [ ] UI for generating accounts from a csv;
	- [x] Might just have a quick python script do it.
- [ ] Better notifications;
- [ ] Statistics page for after the contest and for organizers;
- [ ] Better telemetry and stats;
- [ ] Allow replacing submission list button with "Own submissions" button, to hide all the unavailable submissions
- [ ] Allow hiding advanced filters in submission list page
- [ ] ? Allow auto adding user id to submission list page, to disable circumventing and seeing the amount of total submissions

Ideas:
- [x] If a problem is hidden and the user sent a submission, they shouldn't see the submission anymore
    - Before the contest, there will be a basic problem (probably sum of 2 numbers) in order to accomodate the contestants with the system. When the round starts, it will no longer be visible, so neither should the submissions.
