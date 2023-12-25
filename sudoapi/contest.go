package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var (
	NormalUserVirtualContests = config.GenFlag[bool]("behavior.contests.anyone_virtual", false, "Anyone can create virtual contests")
	NormalUserVCLimit         = config.GenFlag[int]("behavior.contests.normal_user_max_day", 10, "Number of maximum contests a non-proposer can create per day")
)

func (s *BaseAPI) CreateContest(ctx context.Context, name string, cType kilonova.ContestType, author *UserBrief) (int, *StatusError) {
	if author == nil {
		return -1, ErrMissingRequired
	}
	if !(cType == kilonova.ContestTypeNone || cType == kilonova.ContestTypeOfficial || cType == kilonova.ContestTypeVirtual) {
		return -1, Statusf(400, "Invalid contest type")
	}
	if cType == kilonova.ContestTypeNone {
		cType = kilonova.ContestTypeVirtual
	}
	if cType == kilonova.ContestTypeOfficial && !s.IsAdmin(author) {
		cType = kilonova.ContestTypeVirtual
	}

	if !s.IsProposer(author) {
		if !NormalUserVirtualContests.Value() {
			return -1, Statusf(403, "Creation of contests by non-proposers has been disabled")
		}

		// Enforce stricter limits for non-proposers
		since := time.Now().Add(-24 * time.Hour) // rolling day
		cnt, err := s.db.ContestCount(ctx, kilonova.ContestFilter{
			Since: &since,
		})
		if err != nil || (cnt >= NormalUserVCLimit.Value() && NormalUserVCLimit.Value() >= 0) {
			if err != nil {
				zap.S().Warn(err)
			}
			return -1, Statusf(400, "You can create at most %d contests per day", NormalUserVCLimit.Value())
		}
	}
	id, err := s.db.CreateContest(ctx, name, cType)
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

func (s *BaseAPI) Contests(ctx context.Context, filter kilonova.ContestFilter) ([]*kilonova.Contest, *StatusError) {
	contests, err := s.db.Contests(ctx, filter)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) VisibleFutureContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	filter := kilonova.ContestFilter{
		Future:      true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
	}
	if user != nil {
		filter.ImportantContestsUID = &user.ID
	}
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) VisibleRunningContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, *StatusError) {
	filter := kilonova.ContestFilter{
		Running:     true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
		Ordering:    "end_time",
	}
	if user != nil {
		filter.ImportantContestsUID = &user.ID
	}
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) ProblemRunningContests(ctx context.Context, problemID int) ([]*kilonova.Contest, *StatusError) {
	return s.Contests(ctx, kilonova.ContestFilter{
		Running:   true,
		ProblemID: &problemID,
		Ascending: true,
		Ordering:  "end_time",
	})
}

func (s *BaseAPI) ContestLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time) (*kilonova.ContestLeaderboard, *StatusError) {
	switch contest.LeaderboardStyle {
	case kilonova.LeaderboardTypeClassic:
		leaderboard, err := s.db.ContestClassicLeaderboard(ctx, contest, freezeTime)
		if err != nil {
			return nil, WrapError(err, "Couldn't generate leaderboard")
		}
		return leaderboard, nil
	case kilonova.LeaderboardTypeICPC:
		leaderboard, err := s.db.ContestICPCLeaderboard(ctx, contest, freezeTime)
		if err != nil {
			return nil, WrapError(err, "Couldn't generate leaderboard")
		}
		return leaderboard, nil
	default:
		return nil, Statusf(400, "Invalid contest leaderboard type")
	}
}

func (s *BaseAPI) CanJoinContest(c *kilonova.Contest) bool {
	if !c.PublicJoin {
		return false
	}
	if c.RegisterDuringContest && !c.Ended() { // Registration during contest is enabled
		return true
	}
	return !c.Started()
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
