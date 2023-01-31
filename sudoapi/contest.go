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

func (s *BaseAPI) DeleteContest(ctx context.Context, id int) *StatusError {
	if err := s.db.DeleteContest(ctx, id); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete contest")
	}
	s.LogUserAction(ctx, "Removed contest %d", id)
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

func (s *BaseAPI) CanJoinContest(c *kilonova.Contest) bool {
	return c.PublicJoin && time.Now().Before(c.StartTime)
}

// CanSubmitInContest checks if the user is either a contestant and the contest is running, or a tester/editor/admin.
// Ended contests cannot have submissions created by anyone
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
	return reg != nil
}

// CanViewContestProblems checks if the user can see a contest's problems.
// Note that this does not neccesairly mean that he can submit in them!
// A problem may be viewable because the contest is running and visible, but only registered people should submit
// It's a bit frustrating but it's an important distinction
// If you think about it, all submitters can view problems, but not all problem viewers can submit
func (s *BaseAPI) CanViewContestProblems(ctx context.Context, user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !contest.Started() {
		return s.IsContestTester(user, contest) // Tester + Editor + Admin
	}
	if contest.Visible {
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
