package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) CreateContest(ctx context.Context, name string, author *UserBrief) (int, *StatusError) {
	if author == nil {
		return -1, ErrMissingRequired
	}
	id, err := s.db.CreateContest(ctx, name)
	if err != nil {
		return -1, WrapError(err, "Couldn't create contest")
	}
	if err := s.db.AddContestEditor(ctx, id, author.ID); err != nil {
		zap.S().Warn(err)
		return id, WrapError(err, "Couldn't add author to contest editors")
	}
	return id, nil
}

func (s *BaseAPI) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) *kilonova.StatusError {
	if err := s.db.UpdateContest(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest")
	}
	return nil
}

func (s *BaseAPI) UpdateContestProblems(ctx context.Context, id int, list []int) *StatusError {
	if err := s.db.UpdateContestProblems(ctx, id, list); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest problems")
	}
	return nil
}

func (s *BaseAPI) DeleteContest(ctx context.Context, contest *kilonova.Contest) *StatusError {
	if contest == nil {
		return Statusf(400, "Invalid contest")
	}
	if err := s.db.DeleteContest(ctx, contest.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete contest")
	}
	s.LogUserAction(ctx, "Removed contest #%d: %q", contest.ID, contest.Name)
	return nil
}

func (s *BaseAPI) Contest(ctx context.Context, id int) (*kilonova.Contest, *StatusError) {
	contest, err := s.db.Contest(ctx, id)
	if err != nil || contest == nil {
		return nil, WrapError(ErrNotFound, "Contest not found")
	}
	return contest, nil
}

func (s *BaseAPI) VisibleContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	userID := 0
	if user != nil {
		userID = user.ID
	}

	contests, err := s.db.VisibleContests(ctx, userID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) VisibleFutureContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	userID := 0
	if user != nil {
		userID = user.ID
	}

	contests, err := s.db.VisibleFutureContests(ctx, userID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) VisibleRunningContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	userID := 0
	if user != nil {
		userID = user.ID
	}

	contests, err := s.db.VisibleRunningContests(ctx, userID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) ProblemRunningContests(ctx context.Context, problemID int) ([]*kilonova.Contest, *StatusError) {
	contests, err := s.db.RunningContestsByProblem(ctx, problemID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) ContestLeaderboard(ctx context.Context, contestID int) (*kilonova.ContestLeaderboard, *StatusError) {
	leaderboard, err := s.db.ContestLeaderboard(ctx, contestID)
	if err != nil {
		return nil, WrapError(err, "Couldn't generate leaderboard")
	}
	return leaderboard, nil
}

func (s *BaseAPI) CanJoinContest(c *kilonova.Contest) bool {
	if !c.PublicJoin {
		return false
	}
	if c.RegisterDuringContest && time.Now().Before(c.EndTime) { // Registration during contest is enabled
		return true
	}
	return time.Now().Before(c.StartTime)
}

// CanSubmitInContest checks if the user is either a contestant and the contest is running, or a tester/editor/admin.
// Ended contests cannot have submissions created by anyone
// Also, USACO-style contests are fun to handle...
func (s *BaseAPI) CanSubmitInContest(user *kilonova.UserBrief, c *kilonova.Contest) bool {
	if c.Ended() {
		return false
	}
	if s.IsContestTester(user, c) {
		return true
	}
	if user == nil || c == nil {
		return false
	}
	if !c.Running() {
		return false
	}
	reg, err := s.db.ContestRegistration(context.Background(), c.ID, user.ID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}

	if reg == nil {
		return false
	}

	if c.PerUserTime == 0 { // Normal, non-USACO contest, only registration matters
		return true
	}

	// USACO contests are a bit more finnicky
	if reg.IndividualStartTime == nil || reg.IndividualEndTime == nil { // Hasn't pressed start yet
		return false
	}
	return time.Now().After(*reg.IndividualStartTime) && time.Now().Before(*reg.IndividualEndTime) // During window of visibility
}

// CanViewContestProblems checks if the user can see a contest's problems.
// Note that this does not neccesairly mean that he can submit in them!
// A problem may be viewable because the contest is running and visible, but only registered people should submit
// It's a bit frustrating but it's an important distinction
// If you think about it, all submitters can view problems, but not all problem viewers can submit
func (s *BaseAPI) CanViewContestProblems(ctx context.Context, user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if s.IsContestTester(user, contest) { // Tester + Editor + Admin
		return true
	}
	if !contest.Started() {
		return false
	}
	if contest.Ended() && contest.Visible { // Once ended and visible, it's free for all
		return true
	}
	if contest.PerUserTime == 0 && contest.Visible && !contest.RegisterDuringContest {
		// Problems can be seen by anyone only on visible, non-USACO contests that disallow registering during contest
		return true
	}
	return s.CanSubmitInContest(user, contest)
}

func (s *BaseAPI) AddContestEditor(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest editor: sanity strip failed")
	}
	if err := s.db.AddContestEditor(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest editor")
	}
	return nil
}

func (s *BaseAPI) AddContestTester(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest tester: sanity strip failed")
	}
	if err := s.db.AddContestTester(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add contest tester")
	}
	return nil
}

func (s *BaseAPI) StripContestAccess(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't strip contest access")
	}
	return nil
}
