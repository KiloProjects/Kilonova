package sudoapi

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/integrations/moss"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var (
	NormalUserVirtualContests = config.GenFlag[bool]("behavior.contests.anyone_virtual", false, "Anyone can create virtual contests")
	NormalUserVCLimit         = config.GenFlag[int]("behavior.contests.normal_user_max_day", 10, "Number of maximum contests a non-proposer can create per day")
)

func (s *BaseAPI) CreateContest(ctx context.Context, name string, cType kilonova.ContestType, author *kilonova.UserBrief) (int, error) {
	if author == nil {
		return -1, kilonova.ErrMissingRequired
	}
	if !(cType == kilonova.ContestTypeNone || cType == kilonova.ContestTypeOfficial || cType == kilonova.ContestTypeVirtual) {
		return -1, Statusf(400, "Invalid contest type")
	}
	if cType == kilonova.ContestTypeNone {
		cType = kilonova.ContestTypeVirtual
	}
	if cType == kilonova.ContestTypeOfficial && !author.IsAdmin() {
		cType = kilonova.ContestTypeVirtual
	}

	if !author.IsProposer() {
		if !NormalUserVirtualContests.Value() {
			return -1, Statusf(403, "Creation of contests by non-proposers has been disabled")
		}

		// Enforce stricter limits for non-proposers
		since := time.Now().Add(-24 * time.Hour) // rolling day
		cnt, err := s.db.ContestCount(ctx, kilonova.ContestFilter{
			Since:    &since,
			EditorID: &author.ID,
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
		return -1, fmt.Errorf("couldn't create contest: %w", err)
	}
	if err := s.db.AddContestEditor(ctx, id, author.ID); err != nil {
		zap.S().Warn(err)
		return id, fmt.Errorf("couldn't add author to contest editors: %w", err)
	}
	return id, nil
}

func (s *BaseAPI) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) error {
	if err := s.db.UpdateContest(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't update contest: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateContestProblems(ctx context.Context, id int, list []int) error {
	if err := s.db.UpdateContestProblems(ctx, id, list); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't update contest problems: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteContest(ctx context.Context, contest *kilonova.Contest) error {
	if contest == nil {
		return Statusf(400, "Invalid contest")
	}
	if err := s.db.DeleteContest(ctx, contest.ID); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't delete contest: %w", err)
	}
	s.LogUserAction(ctx, "Removed contest", slog.Any("contest", contest))
	return nil
}

func (s *BaseAPI) Contest(ctx context.Context, id int) (*kilonova.Contest, error) {
	contest, err := s.db.Contest(ctx, id)
	if err != nil || contest == nil {
		return nil, fmt.Errorf("contest not found: %w", ErrNotFound)
	}
	return contest, nil
}

func (s *BaseAPI) Contests(ctx context.Context, filter kilonova.ContestFilter) ([]*kilonova.Contest, error) {
	contests, err := s.db.Contests(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch contests: %w", err)
	}
	return contests, nil
}

func (s *BaseAPI) ContestCount(ctx context.Context, filter kilonova.ContestFilter) (int, error) {
	cnt, err := s.db.ContestCount(ctx, filter)
	if err != nil {
		return -1, fmt.Errorf("couldn't fetch contests: %w", err)
	}
	return cnt, nil
}

func (s *BaseAPI) VisibleFutureContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, error) {
	filter := kilonova.ContestFilter{
		Future:      true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
	}
	var uid = -1
	if user != nil {
		uid = user.ID
	}
	filter.ImportantContestsUID = &uid
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) VisibleRunningContests(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.Contest, error) {
	filter := kilonova.ContestFilter{
		Running:     true,
		Look:        true,
		LookingUser: user,
		Ascending:   true,
		Ordering:    "end_time",
	}
	var uid = -1
	if user != nil {
		uid = user.ID
	}
	filter.ImportantContestsUID = &uid
	return s.Contests(ctx, filter)
}

func (s *BaseAPI) ProblemRunningContests(ctx context.Context, problemID int) ([]*kilonova.Contest, error) {
	return s.Contests(ctx, kilonova.ContestFilter{
		Running:   true,
		ProblemID: &problemID,
		Ascending: true,
		Ordering:  "end_time",
	})
}

func (s *BaseAPI) ContestLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time, filter kilonova.UserFilter) (*kilonova.ContestLeaderboard, error) {
	switch contest.LeaderboardStyle {
	case kilonova.LeaderboardTypeClassic:
		leaderboard, err := s.db.ContestClassicLeaderboard(ctx, contest, freezeTime, &filter)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate leaderboard: %w", err)
		}
		return leaderboard, nil
	case kilonova.LeaderboardTypeICPC:
		leaderboard, err := s.db.ContestICPCLeaderboard(ctx, contest, freezeTime, &filter)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate leaderboard: %w", err)
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
	if c.IsTester(user) {
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

	if contest.IsTester(user) { // Tester + Editor + Admin
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

func (s *BaseAPI) CanViewContestLeaderboard(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !s.IsContestVisible(user, contest) {
		return false
	}
	if contest.IsTester(user) { // Tester + Editor + Admin
		return true
	}
	// Otherwise, normal contestant
	if !contest.Started() {
		// Non-started contests can leak problem IDs/names
		return false
	}
	return contest.PublicLeaderboard
}

func (s *BaseAPI) AddContestEditor(ctx context.Context, pbid int, uid int) error {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return fmt.Errorf("couldn't add contest editor: sanity strip failed: %w", err)
	}
	if err := s.db.AddContestEditor(ctx, pbid, uid); err != nil {
		return fmt.Errorf("couldn't add contest editor: %w", err)
	}
	return nil
}

func (s *BaseAPI) AddContestTester(ctx context.Context, pbid int, uid int) error {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return fmt.Errorf("couldn't add contest tester: sanity strip failed: %w", err)
	}
	if err := s.db.AddContestTester(ctx, pbid, uid); err != nil {
		return fmt.Errorf("couldn't add contest tester: %w", err)
	}
	return nil
}

func (s *BaseAPI) StripContestAccess(ctx context.Context, pbid int, uid int) error {
	if err := s.db.StripContestAccess(ctx, pbid, uid); err != nil {
		return fmt.Errorf("couldn't strip contest access: %w", err)
	}
	return nil
}

func (s *BaseAPI) RunMOSS(ctx context.Context, contest *kilonova.Contest) error {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{ContestID: &contest.ID})
	if err != nil {
		return err
	}

	for _, pb := range pbs {
		subs, err := s.RawSubmissions(ctx, kilonova.SubmissionFilter{
			ProblemID: &pb.ID,
			ContestID: &contest.ID,

			Ordering:  "score",
			Ascending: false,
		})
		if err != nil {
			return err
		}
		if len(subs) == 0 {
			continue
		}
		users := make(map[int]bool)
		mossSubs := make(map[string][]*kilonova.Submission)
		for _, sub := range subs {
			if _, ok := users[sub.UserID]; ok {
				continue
			}
			users[sub.UserID] = true

			name := s.Language(ctx, sub.Language).MOSSName()
			// TODO: See if this can be simplified?
			_, ok := mossSubs[name]
			if !ok {
				mossSubs[name] = []*kilonova.Submission{sub}
			} else {
				mossSubs[name] = append(mossSubs[name], sub)
			}
		}

		for mossLang, subs := range mossSubs {
			lang := s.LanguageFromMOSS(ctx, mossLang)

			slog.InfoContext(ctx, "Running MOSS", slog.Any("problem", pb), slog.Any("lang", lang), slog.Int("sub_count", len(subs)))
			mossID, err := s.db.InsertMossSubmission(ctx, contest.ID, pb.ID, lang.InternalName, len(subs))
			if err != nil {
				return fmt.Errorf("could not add MOSS stub to DB: %w", err)
			}

			go func(mossID int, lang *Language, mossLang string, subs []*kilonova.Submission) {
				conn, err := moss.New(ctx)
				if err != nil {
					slog.WarnContext(ctx, "Could not initialize MOSS", slog.Any("err", err))
				}
				defer conn.Close()

				url, err := conn.Process(&moss.Options{
					LanguageName: mossLang,
					Comment:      fmt.Sprintf("%s - %s (%s)", contest.Name, pb.Name, lang.PrintableName),

					Files: iter.Seq[*moss.File](func(yield func(*moss.File) bool) {
						for _, sub := range subs {
							user, err := s.UserBrief(ctx, sub.UserID)
							if err != nil {
								slog.WarnContext(ctx, "Could not get user", slog.Any("err", err))
								return
							}

							code, err := s.RawSubmissionCode(ctx, sub.ID)
							if err != nil {
								slog.WarnContext(ctx, "Could not get submission code", slog.Any("err", err))
								return
							}

							if !yield(moss.NewFile(mossLang, user.Name, code)) {
								return
							}
						}
					}),
				})
				if err != nil {
					slog.WarnContext(ctx, "Could not get MOSS result", slog.Any("err", err))
				}
				if err := s.db.SetMossURL(ctx, mossID, url); err != nil {
					slog.WarnContext(ctx, "Could not set MOSS results URL", slog.Any("err", err))
				}
			}(mossID, lang, mossLang, subs)
		}
	}
	return nil
}

func (s *BaseAPI) MOSSSubmissions(ctx context.Context, contestID int) ([]*kilonova.MOSSSubmission, error) {
	subs, err := s.db.MossSubmissions(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch MOSS submissions: %w", err)
	}
	return subs, nil
}
