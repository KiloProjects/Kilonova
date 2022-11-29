Kilonova will soon be used for organizing an official contest. In order to achieve this, multiple features need to be implemented.

High priority:
- [ ] Contest system:
	- [ ] Problem visibility;
	- [ ] Questions (Yes/No/No comment/etc);
	- [ ] General announcements;
	- [ ] Leaderboard;
- [ ] Allow to disable contestant access to others' submissions;
- [ ] Endpoint for account generation;
- [ ] Websockets for live updates.

Medium priority:
- [x] Allow to disable manual signup;
- [ ] Real-time leaderboard for admins;
- [x] Show max score on problem page;
- [ ] Hide problem editors during contest if not admin;
- [ ] Disable pastes for contest problems during contest;

- [ ] See which other features should be disabled during contests;
- [ ] Disable forgot password things for generated accounts.

Nice-to-have:
- [ ] UI for generating accounts from a csv;
	- Might just have a quick python script do it.
- [ ] A separate system for evaluating tasks;
- [ ] Better notifications;
- [ ] Statistics page for after the contest and for organizers;
- [ ] Better telemetry and stats;
- [ ] Allow replacing submission list button with "Own submissions" button, to hide all the unavailable submissions
- [ ] Allow hiding advanced filters in submission list page
- [ ] ? Allow auto adding user id to submission list page, to disable circumventing and seeing the amount of total submissions

Ideas:
- [ ] If a problem is hidden and the user sent a submission, they shouldn't see the submission anymore
    - Before the contest, there will be a basic problem (probably sum of 2 numbers) in order to accomodate the contestants with the system. When the round starts, it will no longer be visible, so neither should the submissions.