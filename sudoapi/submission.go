package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/shopspring/decimal"
)

// Submission stuff

func (s *BaseAPI) MaxScoreSubID(ctx context.Context, uid, pbID int) (int, error) {
	id, err := s.db.MaxScoreSubID(ctx, uid, pbID)
	if err != nil {
		return -1, fmt.Errorf("couldn't get max score ID: %w", err)
	}
	return id, nil
}

func (s *BaseAPI) MaxScore(ctx context.Context, uid, pbID int) decimal.Decimal {
	return s.db.MaxScore(ctx, uid, pbID)
}

func (s *BaseAPI) ContestMaxScore(ctx context.Context, uid, pbID, contestID int, freezeTime *time.Time) decimal.Decimal {
	return s.db.ContestMaxScore(ctx, uid, pbID, contestID, freezeTime)
}

func (s *BaseAPI) fillSubmissions(ctx context.Context, cnt int, subs []*kilonova.Submission, look bool, lookingUser *kilonova.UserBrief, truncatedCnt bool) (*Submissions, error) {
	usersMap := make(map[int]*kilonova.UserBrief)
	problemsMap := make(map[int]*kilonova.Problem)

	userIDs := make([]int, 0, len(subs))
	{
		userIDsMap := make(map[int]bool)
		for _, sub := range subs {
			if _, ok := userIDsMap[sub.UserID]; !ok {
				userIDsMap[sub.UserID] = true
				userIDs = append(userIDs, sub.UserID)
			}
		}
	}
	users, err := s.UsersBrief(ctx, kilonova.UserFilter{
		IDs: userIDs,
	})
	if err != nil {
		slog.WarnContext(ctx, "Error getting users", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get users: %w", err)
	}
	for _, user := range users {
		usersMap[user.ID] = user
	}

	problemIDs := make([]int, 0, len(subs))
	{
		problemIDsMap := make(map[int]bool)
		for _, sub := range subs {
			if _, ok := problemIDsMap[sub.ProblemID]; !ok {
				problemIDsMap[sub.ProblemID] = true
				problemIDs = append(problemIDs, sub.ProblemID)
			}
		}
	}
	problems, err := s.Problems(ctx, kilonova.ProblemFilter{
		IDs:         problemIDs,
		Look:        look,
		LookingUser: lookingUser,
	})
	if err != nil {
		slog.WarnContext(ctx, "Error getting problems", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get problems: %w", err)
	}
	for _, problem := range problems {
		problemsMap[problem.ID] = problem
	}

	for i, sub := range subs {
		if _, ok := usersMap[sub.UserID]; !ok {
			slog.WarnContext(ctx, "Couldn't find user in map", slog.Int("userID", sub.UserID))
			continue
		}

		pb, ok := problemsMap[sub.ProblemID]
		if !ok {
			slog.WarnContext(ctx, "Couldn't find problem in map. Something has gone terribly wrong", slog.Int("problemID", sub.ProblemID))
		}

		if look {
			// if pb is nil, then it will failsafe into old-style visibility
			s.filterSubmission(ctx, subs[i], pb, lookingUser)
		}
	}

	return &Submissions{
		Submissions: subs,
		Count:       cnt,
		Truncated:   truncatedCnt,
		Users:       usersMap,
		Problems:    problemsMap,
	}, nil
}

func (s *BaseAPI) Submissions(ctx context.Context, filter kilonova.SubmissionFilter, look bool, lookingUser *kilonova.UserBrief) (*Submissions, error) {
	if filter.Limit == 0 || filter.Limit > 50 {
		filter.Limit = 50
	}

	if filter.Ordering == "" {
		filter.Ordering = "id"
	}

	if look {
		filter.Look = true
		filter.LookingUser = lookingUser
	}

	subs, err := s.db.Submissions(ctx, filter)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get submissions", slog.Any("err", err))
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, ErrUnknownError
	}

	maxCnt := flags.LimitedSubCount.Value()
	if filter.ContestID != nil || filter.ProblemID != nil || filter.UserID != nil {
		// Never filter on these, for now.
		maxCnt = -100
	}

	cnt, err := s.db.SubmissionCount(ctx, filter, maxCnt+1)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get submission count", slog.Any("err", err))
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, ErrUnknownError
	}

	var truncated bool
	if maxCnt > 0 && cnt > maxCnt {
		cnt--
		truncated = true
	}

	return s.fillSubmissions(ctx, cnt, subs, look, lookingUser, truncated)
}

// Remember to do proper authorization when using this
func (s *BaseAPI) RawSubmission(ctx context.Context, id int) (*kilonova.Submission, error) {
	sub, err := s.db.Submission(ctx, id)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get raw submission", slog.Any("err", err))
		return nil, ErrUnknownError
	}
	if sub == nil {
		return nil, Statusf(404, "Couldn't find submission")
	}
	return sub, nil
}

// Should only ever be used for grader stuff
func (s *BaseAPI) RawSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, error) {
	subs, err := s.db.Submissions(ctx, filter)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get raw submissions", slog.Any("err", err))
		return nil, ErrUnknownError
	}
	return subs, nil
}

func (s *BaseAPI) RawSubmissionFiles(ctx context.Context, subid int) ([]*kilonova.SubmissionFile, error) {
	files, err := s.db.SubmissionFiles(ctx, subid)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get raw submission files", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get submission files: %w", err)
	}
	return files, nil
}

func (s *BaseAPI) RawSubmissionCode(ctx context.Context, subid int) ([]byte, error) {
	data, err := s.db.SubmissionCode(ctx, subid)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get raw submission code", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get submission code: %w", err)
	}
	return data, nil
}

func (s *BaseAPI) SubmissionCode(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, lookingUser *kilonova.UserBrief, isLooking bool) ([]byte, error) {
	if sub == nil || subProblem == nil || sub.ProblemID != subProblem.ID {
		return nil, Statusf(400, "Invalid source code parameters")
	}
	if isLooking && !s.isSubmissionVisible(ctx, sub, subProblem, lookingUser) {
		return []byte{}, nil
	}
	data, err := s.db.SubmissionCode(ctx, sub.ID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get submission code", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get submission code: %w", err)
	}
	return data, nil
}

func (s *BaseAPI) SubmissionFiles(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, lookingUser *kilonova.UserBrief, isLooking bool) ([]*kilonova.SubmissionFile, error) {
	if sub == nil || subProblem == nil || sub.ProblemID != subProblem.ID {
		return nil, Statusf(400, "Invalid submission file parameters")
	}
	if isLooking && !s.isSubmissionVisible(ctx, sub, subProblem, lookingUser) {
		return []*kilonova.SubmissionFile{}, nil
	}
	files, err := s.RawSubmissionFiles(ctx, sub.ID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get submission files", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get submission files: %w", err)
	}
	return files, nil
}

type FullSubmission = kilonova.FullSubmission

func (s *BaseAPI) Submission(ctx context.Context, subid int, lookingUser *kilonova.UserBrief) (*FullSubmission, error) {
	return s.getSubmission(ctx, subid, lookingUser, true)
}

// FullSubmission gets the submission regardless of if there is a user watching or not
func (s *BaseAPI) FullSubmission(ctx context.Context, subid int) (*FullSubmission, error) {
	return s.getSubmission(ctx, subid, nil, false)
}

func (s *BaseAPI) getSubmission(ctx context.Context, subid int, lookingUser *kilonova.UserBrief, isLooking bool) (*FullSubmission, error) {
	var sub *kilonova.Submission
	var problem *kilonova.Problem
	if isLooking {
		sub2, err := s.db.SubmissionLookingUser(ctx, subid, lookingUser)
		if err != nil || sub2 == nil {
			return nil, Statusf(404, "Submission not found or user may not have access")
		}

		problem2, err := s.Problem(ctx, sub2.ProblemID)
		if err != nil {
			return nil, err
		}

		if !s.IsProblemVisible(lookingUser, problem2) {
			return nil, Statusf(403, "Submission hidden because problem is not visible.")
		}

		s.filterSubmission(ctx, sub2, problem2, lookingUser)
		sub = sub2
		problem = problem2
	} else {
		sub2, err := s.db.Submission(ctx, subid)
		if err != nil || sub2 == nil {
			return nil, Statusf(404, "Submission not found")
		}

		problem2, err := s.Problem(ctx, sub2.ProblemID)
		if err != nil {
			return nil, err
		}

		sub = sub2
		problem = problem2
	}

	rez := &FullSubmission{Submission: *sub, CodeTrulyVisible: s.subVisibleRegardless(ctx, sub, lookingUser, problem)}
	author, err := s.UserBrief(ctx, sub.UserID)
	if err != nil {
		return nil, err
	}
	rez.Author = author

	rez.Problem = problem
	rez.ProblemEditor = s.IsProblemEditor(lookingUser, rez.Problem)

	rez.SubTests, err = s.SubTests(ctx, subid)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get subtests", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't fetch subtests: %w", err)
	}

	rez.SubTasks, err = s.SubmissionSubTasks(ctx, subid)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get subtasks", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't fetch subtasks: %w", err)
	}

	return rez, nil
}

func (s *BaseAPI) UpdateSubmission(ctx context.Context, id int, status kilonova.SubmissionUpdate) error {
	if err := s.db.UpdateSubmission(ctx, id, status); err != nil {
		slog.WarnContext(ctx, "Couldn't update submission", slog.Any("err", err), slog.Int("subID", id))
		return fmt.Errorf("couldn't update submission: %w", err)
	}
	return nil
}

// RemainingSubmissionCount calculates how many more submissions a user can send
// The first return value shows how many submissions they can still send
// The second return value shows whether there is a limit or not
func (s *BaseAPI) RemainingSubmissionCount(ctx context.Context, contest *kilonova.Contest, problemID int, userID int) (int, bool, error) {
	if contest.MaxSubs < 0 {
		return 1, false, nil
	}
	cnt, err := s.db.SubmissionCount(ctx, kilonova.SubmissionFilter{
		ContestID: &contest.ID,
		ProblemID: &problemID,
		UserID:    &userID,
	}, -1)
	if err != nil {
		return -1, true, fmt.Errorf("couldn't get submission count: %w", err)
	}
	return contest.MaxSubs - cnt, true, nil
}

func (s *BaseAPI) LastSubmissionTime(ctx context.Context, filter kilonova.SubmissionFilter) (*time.Time, error) {
	t, err := s.db.LastSubmissionTime(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("couldn't get last submission time: %w", err)
	}
	return t, nil
}

// CreateSubmission produces a new submission and also creates the necessary subtests
func (s *BaseAPI) CreateSubmission(ctx context.Context, author *kilonova.UserFull, problem *kilonova.Problem, code []byte, codeFilename string, lang *Language, contestID *int, bypassSubCount bool) (int, error) {
	if author == nil {
		return -1, Statusf(400, "Invalid submission author")
	}
	if problem == nil {
		return -1, Statusf(400, "Invalid submission problem")
	}
	if len(code) > problem.SourceSize { // Maximum admitted by problem
		return -1, Statusf(400, "Code exceeds %d characters", problem.SourceSize)
	}
	if !s.IsProblemVisible(author.Brief(), problem) {
		return -1, Statusf(400, "Submitter can't see the problem!")
	}

	if !bypassSubCount {
		// Get the number of waiting submissions
		cnt, err := s.db.SubmissionCount(ctx, kilonova.SubmissionFilter{
			UserID:  &author.ID,
			Waiting: true,
		}, -1)
		if err != nil {
			return -1, fmt.Errorf("couldn't get unfinished submission count")
		}

		if flags.WaitingSubLimit.Value() > 0 && cnt >= flags.WaitingSubLimit.Value() {
			return -1, Statusf(400, "You cannot have more than %d submissions to the evaluation queue at once", flags.WaitingSubLimit.Value())
		}

		t := time.Now().Add(-1 * time.Minute)
		cnt, err = s.db.SubmissionCount(ctx, kilonova.SubmissionFilter{
			UserID: &author.ID,
			Since:  &t,
		}, -1)
		if err != nil {
			return -1, fmt.Errorf("couldn't get recent submission count")
		}

		if flags.TotalSubLimit.Value() > 0 && cnt > flags.TotalSubLimit.Value() {
			s.LogToDiscord(ctx, "User tried to exceed submission send limit, something might be fishy")
			return -1, Statusf(401, "You cannot submit more than %d submissions in a minute, please wait a bit", flags.TotalSubLimit.Value())
		}

		if !author.VerifiedEmail && flags.UnverifiedSubLimit.Value() > 0 && cnt > flags.UnverifiedSubLimit.Value() {
			s.LogVerbose(ctx, "Unverified user exceeded their submission limit")
			return -1, Statusf(401, "Users with unverified email cannot submit more than %d times per minute, please verify your email or wait", flags.UnverifiedSubLimit.Value())
		}
	}

	if contestID != nil {
		contest, err := s.Contest(ctx, *contestID)
		if err != nil || !s.IsContestVisible(author.Brief(), contest) {
			return -1, Statusf(404, "Couldn't find contest")
		}
		if !s.CanSubmitInContest(author.Brief(), contest) {
			return -1, Statusf(400, "Submitter cannot submit to contest")
		}
		if pb, err := s.ContestProblem(ctx, contest, author.Brief(), problem.ID); err != nil || pb == nil {
			return -1, Statusf(400, "Problem is not in contest")
		}
		cnt, _, err := s.RemainingSubmissionCount(ctx, contest, problem.ID, author.ID)
		if err != nil {
			return -1, err
		}
		if cnt <= 0 {
			return -1, Statusf(http.StatusTooManyRequests, "Max submission count for problem reached")
		}
		if !contest.IsTester(author.Brief()) && contest.SubmissionCooldown > 0 {
			t, err := s.LastSubmissionTime(ctx, kilonova.SubmissionFilter{
				ContestID: &contest.ID,
				UserID:    &author.ID,
			})
			if err != nil {
				return -1, err
			}
			if t != nil {
				if d := contest.SubmissionCooldown - time.Since(*t); d > 0 {
					return -1, Statusf(http.StatusTooManyRequests, "You are going too fast! Please wait %d more second(s) before submitting again.", int(d.Seconds())+1)
				}
			}
		}
	} else {
		// Check that the problem is fully visible (ie. outside of a contest medium)
		// Users may be able to bypass icpc penalties otherwise
		if !s.IsProblemFullyVisible(author.Brief(), problem) {
			return -1, Statusf(400, "You cannot submit to a problem outside a contest while it's running")
		}
	}

	if len(code) == 0 {
		return -1, Statusf(400, "Empty code")
	}

	langs, err := s.ProblemLanguages(ctx, problem)
	if err != nil {
		return -1, fmt.Errorf("could not get problem languages: %w", err)
	}
	if !slices.ContainsFunc(langs, func(a *Language) bool { return a.InternalName == lang.InternalName }) {
		return -1, Statusf(400, "Language not supported by problem")
	}

	fname := lang.DefaultFilename
	// For now, accept this only for Java files
	if lang.InternalName == "java" && codeFilename != "" {
		fname = codeFilename
	}

	// Add submission
	id, err := s.db.CreateSubmission(ctx, author.ID, problem, lang.InternalName, code, fname, contestID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't create submission", slog.Any("err", err))
		return -1, fmt.Errorf("couldn't create submission")
	}

	if err := s.db.InitSubmission(ctx, id); err != nil {
		slog.WarnContext(ctx, "Couldn't initialize submission", slog.Any("err", err), slog.Int("subID", id))
		return -1, fmt.Errorf("couldn't initialize submission")
	}

	// Wake immediately to grade submission
	s.WakeGrader()

	return id, nil
}

func (s *BaseAPI) DeleteSubmission(ctx context.Context, subID int) error {
	if err := s.db.DeleteSubmission(ctx, subID); err != nil {
		slog.WarnContext(ctx, "Couldn't delete submission", slog.Any("err", err), slog.Int("subID", subID))
		return fmt.Errorf("failed to delete submission")
	}
	return nil
}

func (s *BaseAPI) ResetProblemSubmissions(ctx context.Context, problem *kilonova.Problem) error {
	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &problem.ID}, kilonova.SubmissionUpdate{
		Status: kilonova.StatusReevaling,
	}); err != nil {
		slog.WarnContext(ctx, "Couldn't mark submissions for reevaluation", slog.Any("err", err))
		return fmt.Errorf("couldn't mark submissions for reevaluation: %w", err)
	}

	s.LogUserAction(ctx, "Reset problem submissions", slog.Any("problem", problem))

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) subVisibleRegardless(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief, subProblem *kilonova.Problem) bool {
	if sub == nil {
		return false
	}

	if !user.IsAuthed() {
		return false
	}

	if sub.IsEditor(user) {
		return true
	}

	if subProblem != nil && s.IsProblemEditor(user, subProblem) {
		return true
	}

	score := s.db.MaxScore(ctx, user.ID, sub.ProblemID)
	return score.Equal(decimal.NewFromInt(100))
}

func (s *BaseAPI) isSubmissionVisible(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, user *kilonova.UserBrief) bool {
	if sub == nil {
		return false
	}

	if user == nil || !user.IsAuthed() {
		return false
	}

	// If enabled that people see all source code
	// IsProblemFullyVisible is a workaround for when a contest is running but there are submissions that were not sent in the contest
	if flags.SubForEveryoneConfig.Value() && s.IsProblemFullyVisible(user, subProblem) &&
		!slices.Contains(flags.SubForEveryoneBlacklist.Value(), sub.ProblemID) {

		// Get number of running virtual contests where user is participant for this problem and the person trying to look is not a contest editor
		// If it returns a nonzero number of results, then we need to filter out
		numContests, err := s.ContestCount(ctx, kilonova.ContestFilter{
			Running:      true,
			ContestantID: &user.ID,
			ProblemID:    &subProblem.ID,
			Type:         kilonova.ContestTypeVirtual,

			EditorID:  &user.ID,
			NotEditor: true,
		})
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get running contests", slog.Any("err", err))
		} else if numContests > 0 {
			return false
		}

		if sub.ContestID == nil {
			// If problem fully visible and submission not in contest, just show the source code
			return true
		}
		contest, err := s.Contest(ctx, *sub.ContestID)
		if err == nil {
			if contest.Ended() {
				// Or if it's from a contest that ended
				return true
			}
			if contest.IsEditor(user) {
				// Contest editors should always be able to view submissions
				return true
			}
		}
	}

	return s.subVisibleRegardless(ctx, sub, user, subProblem)
}

func (s *BaseAPI) filterSubmission(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, user *kilonova.UserBrief) {
	if sub == nil {
		return
	}
	if !s.isSubmissionVisible(ctx, sub, subProblem, user) {
		// sub.Code = ""
		sub.CompileMessage = nil
		sub.CodeSize = 0
	}
	if !s.IsProblemEditor(user, subProblem) {
		sub.CompileTime = nil
	}
}

func (s *BaseAPI) CreatePaste(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief) (string, error) {
	if !flags.PastesEnabled.Value() {
		return "", kilonova.ErrFeatureDisabled
	}
	paste := &kilonova.SubmissionPaste{Submission: sub, Author: user}
	if err := s.db.CreatePaste(ctx, paste); err != nil {
		return "", fmt.Errorf("couldn't create paste: %w", err)
	}
	return paste.ID, nil
}

func (s *BaseAPI) SubmissionPaste(ctx context.Context, id string) (*kilonova.SubmissionPaste, error) {
	if !flags.PastesEnabled.Value() {
		return nil, kilonova.ErrFeatureDisabled
	}
	paste, err := s.db.SubmissionPaste(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't get paste: %w", err)
	}
	if paste == nil {
		return nil, Statusf(404, "Couldn't find paste")
	}
	return paste, nil
}

func (s *BaseAPI) DeletePaste(ctx context.Context, id string) error {
	if !flags.PastesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if err := s.db.DeleteSubPaste(ctx, id); err != nil {
		return fmt.Errorf("couldn't delete paste: %w", err)
	}
	return nil
}

func (s *BaseAPI) SubmissionSubTasks(ctx context.Context, subID int) ([]*kilonova.SubmissionSubTask, error) {
	subs, err := s.db.SubmissionSubTasksBySubID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get submission subtasks: %w", err)
	}
	return subs, nil
}

func (s *BaseAPI) MaximumScoreSubTasks(ctx context.Context, problemID, userID int, contestID *int) ([]*kilonova.SubmissionSubTask, error) {
	subs, err := s.db.MaximumScoreSubTasks(ctx, problemID, userID, contestID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get maximum subtasks: %w", err)
	}
	return subs, nil
}

func (s *BaseAPI) UpdateSubmissionSubtaskPercentage(ctx context.Context, id int, percentage decimal.Decimal) error {
	if err := s.db.UpdateSubmissionSubtaskPercentage(ctx, id, percentage); err != nil {
		return fmt.Errorf("couldn't update subtask percentage: %w", err)
	}
	return nil
}
