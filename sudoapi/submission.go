package sudoapi

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	SubForEveryoneConfig = config.GenFlag("behavior.everyone_subs", true, "Anyone can view others' source code")

	PastesEnabled = config.GenFlag("feature.pastes.enabled", true, "Pastes")
)

// Submission stuff

func (s *BaseAPI) MaxScoreSubID(ctx context.Context, uid, pbID int) (int, *StatusError) {
	id, err := s.db.MaxScoreSubID(ctx, uid, pbID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get max score ID")
	}
	return id, nil
}

func (s *BaseAPI) ICPCMaxScoreSubID(ctx context.Context, uid, pbID int) (int, *StatusError) {
	id, err := s.db.ICPCMaxScoreSubID(ctx, uid, pbID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get max score ID")
	}
	return id, nil
}

func (s *BaseAPI) MaxScore(ctx context.Context, uid, pbID int) decimal.Decimal {
	return s.db.MaxScore(ctx, uid, pbID)
}

func (s *BaseAPI) ContestMaxScore(ctx context.Context, uid, pbID, contestID int, freezeTime *time.Time) decimal.Decimal {
	return s.db.ContestMaxScore(ctx, uid, pbID, contestID, freezeTime)
}

func (s *BaseAPI) fillSubmissions(ctx context.Context, cnt int, subs []*kilonova.Submission, look bool, lookingUser *UserBrief) (*Submissions, *StatusError) {
	usersMap := make(map[int]*UserBrief)
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
		zap.S().Warnf("Error getting users: %v", err)
		return nil, WrapError(err, "Couldn't get users")
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
		zap.S().Warnf("Error getting problems: %v", err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	for _, problem := range problems {
		problemsMap[problem.ID] = problem
	}

	for i, sub := range subs {
		if _, ok := usersMap[sub.UserID]; !ok {
			zap.S().Warnf("Couldn't find user %d in map", sub.UserID)
			continue
		}

		pb, ok := problemsMap[sub.ProblemID]
		if !ok {
			zap.S().Warnf("Couldn't find problem %d in map. Something has gone terribly wrong", sub.ProblemID)
		}

		if look {
			// if pb is nil, then it will failsafe into old-style visibility
			s.filterSubmission(ctx, subs[i], pb, lookingUser)
		}
	}

	return &Submissions{
		Submissions: subs,
		Count:       cnt,
		Users:       usersMap,
		Problems:    problemsMap,
	}, nil
}

func (s *BaseAPI) Submissions(ctx context.Context, filter kilonova.SubmissionFilter, look bool, lookingUser *UserBrief) (*Submissions, *StatusError) {
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
		if errors.Is(err, context.Canceled) {
			return nil, ErrUnknownError
		}
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}

	cnt, err := s.db.SubmissionCount(ctx, filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, ErrUnknownError
		}
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}

	return s.fillSubmissions(ctx, cnt, subs, look, lookingUser)
}

// Remember to do proper authorization when using this
func (s *BaseAPI) RawSubmission(ctx context.Context, id int) (*kilonova.Submission, *StatusError) {
	sub, err := s.db.Submission(ctx, id)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	if sub == nil {
		return nil, Statusf(404, "Couldn't find submission")
	}
	return sub, nil
}

// Should only ever be used for grader stuff
func (s *BaseAPI) RawSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, *StatusError) {
	subs, err := s.db.Submissions(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return subs, nil
}

type FullSubmission = kilonova.FullSubmission

func (s *BaseAPI) Submission(ctx context.Context, subid int, lookingUser *UserBrief) (*FullSubmission, *StatusError) {
	return s.getSubmission(ctx, subid, lookingUser, true)
}

// FullSubmission gets the submission regardless of if there is a user watching or not
func (s *BaseAPI) FullSubmission(ctx context.Context, subid int) (*FullSubmission, *StatusError) {
	return s.getSubmission(ctx, subid, nil, false)
}

func (s *BaseAPI) getSubmission(ctx context.Context, subid int, lookingUser *UserBrief, isLooking bool) (*FullSubmission, *StatusError) {
	var sub *kilonova.Submission
	var problem *kilonova.Problem
	if isLooking {
		var userID int = 0
		if lookingUser != nil {
			userID = lookingUser.ID
		}
		sub2, err := s.db.SubmissionLookingUser(ctx, subid, userID)
		if err != nil || sub2 == nil {
			return nil, Statusf(404, "Submission not found or user may not have access")
		}

		problem2, err1 := s.Problem(ctx, sub2.ProblemID)
		if err1 != nil {
			return nil, err1
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

		problem2, err1 := s.Problem(ctx, sub2.ProblemID)
		if err1 != nil {
			return nil, err1
		}

		sub = sub2
		problem = problem2
	}

	rez := &FullSubmission{Submission: *sub, CodeTrulyVisible: s.subVisibleRegardless(ctx, sub, lookingUser, problem)}
	author, err1 := s.UserBrief(ctx, sub.UserID)
	if err1 != nil {
		return nil, err1
	}
	rez.Author = author

	rez.Problem = problem
	rez.ProblemEditor = s.IsProblemEditor(lookingUser, rez.Problem)

	rez.SubTests, err1 = s.SubTests(ctx, subid)
	if err1 != nil {
		zap.S().Warn(err1)
		return nil, Statusf(500, "Couldn't fetch subtests")
	}

	rez.SubTasks, err1 = s.SubmissionSubTasks(ctx, subid)
	if err1 != nil {
		zap.S().Warn(err1)
		return nil, Statusf(500, "Couldn't fetch subtasks")
	}

	return rez, nil
}

func (s *BaseAPI) UpdateSubmission(ctx context.Context, id int, status kilonova.SubmissionUpdate) *StatusError {
	if err := s.db.UpdateSubmission(ctx, id, status); err != nil {
		zap.S().Warn(err, id)
		return WrapError(err, "Couldn't update submission")
	}
	return nil
}

func (s *BaseAPI) RemainingSubmissionCount(ctx context.Context, contest *kilonova.Contest, problem *kilonova.Problem, user *kilonova.UserBrief) (int, *StatusError) {
	cnt, err := s.db.RemainingSubmissionCount(ctx, contest, problem.ID, user.ID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get submission count")
	}
	return cnt, nil
}

var (
	WaitingSubLimit    = config.GenFlag[int]("behavior.submissions.user_max_waiting", 5, "Maximum number of 'waiting' submissions in the eval queue (for a single user)")
	TotalSubLimit      = config.GenFlag[int]("behavior.submissions.user_max_minute", 20, "Maximum number of submissions uploaded per minute (for a single user with verified email)")
	UnverifiedSubLimit = config.GenFlag[int]("behavior.submissions.user_max_unverified", 5, "Maximum number of submissions uploaded per minute (for a single user with unverified email)")
)

// CreateSubmission produces a new submission and also creates the necessary subtests
func (s *BaseAPI) CreateSubmission(ctx context.Context, author *UserFull, problem *kilonova.Problem, code string, lang eval.Language, contestID *int, bypassSubCount bool) (int, *StatusError) {
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
		cnt, err := s.db.WaitingSubmissionCount(ctx, author.ID)
		if err != nil {
			return -1, Statusf(500, "Couldn't get unfinished submission count")
		}

		if WaitingSubLimit.Value() > 0 && cnt > WaitingSubLimit.Value() {
			return -1, Statusf(400, "You cannot have more than %d submissions to the evaluation queue at once", WaitingSubLimit.Value())
		}

		cnt, err = s.db.SubmissionCountSince(ctx, author.ID, time.Now().Add(-1*time.Minute))
		if err != nil {
			return -1, Statusf(500, "Couldn't get recent submission count")
		}

		if TotalSubLimit.Value() > 0 && cnt > TotalSubLimit.Value() {
			s.LogToDiscord(ctx, "User tried to exceed submission send limit, something might be fishy")
			return -1, Statusf(400, "You cannot submit more than %d submissions in a minute, please wait a bit", TotalSubLimit.Value())
		}

		if !author.VerifiedEmail && UnverifiedSubLimit.Value() > 0 && cnt > UnverifiedSubLimit.Value() {
			s.LogVerbose(ctx, "Unverified user exceeded their submission limit")
			return -1, Statusf(400, "Users with unverified email cannot submit more than %d times per minute, please verify your email or wait", UnverifiedSubLimit.Value())
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
		pb, err := s.ContestProblem(ctx, contest, author.Brief(), problem.ID)
		if err != nil || pb == nil {
			return -1, Statusf(400, "Problem is not in contest")
		}
		cnt, err := s.RemainingSubmissionCount(ctx, contest, pb, author.Brief())
		if err != nil {
			return -1, err
		}
		if cnt <= 0 {
			return -1, Statusf(400, "Max submission count for problem reached")
		}
	} else {
		// Check that the problem is fully visible (ie. outside of a contest medium)
		// Users may be able to bypass icpc penalties otherwise
		if !s.IsProblemFullyVisible(author.Brief(), problem) {
			return -1, Statusf(400, "You cannot submit to a problem outside a contest while it's running")
		}
	}

	if code == "" {
		return -1, Statusf(400, "Empty code")
	}

	// Add submission
	id, err := s.db.CreateSubmission(ctx, author.ID, problem, lang, code, contestID)
	if err != nil {
		zap.S().Warn("Couldn't create submission:", err)
		return -1, Statusf(500, "Couldn't create submission")
	}

	if err := s.db.InitSubmission(ctx, id); err != nil {
		zap.S().Warn("Couldn't initialize submission:", err)
		return -1, Statusf(500, "Couldn't initialize submission")
	}

	// Wake immediately to grade submission
	s.WakeGrader()

	return id, nil
}

func (s *BaseAPI) DeleteSubmission(ctx context.Context, subID int) *StatusError {
	if err := s.db.DeleteSubmission(ctx, subID); err != nil {
		zap.S().Warn("Couldn't delete submission:", err)
		return Statusf(500, "Failed to delete submission")
	}
	return nil
}

func (s *BaseAPI) ResetProblemSubmissions(ctx context.Context, problem *kilonova.Problem) *StatusError {
	if err := s.db.ResetProblemSubmissions(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions tests")
	}

	s.LogUserAction(ctx, "Reset submissions for problem #%d: %s", problem.ID, problem.Name)

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) subVisibleRegardless(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief, subProblem *kilonova.Problem) bool {
	if sub == nil {
		return false
	}

	if !s.IsAuthed(user) {
		return false
	}

	if s.IsSubmissionEditor(sub, user) {
		return true
	}

	if subProblem != nil && s.IsProblemEditor(user, subProblem) {
		return true
	}

	score := s.db.MaxScore(context.Background(), user.ID, sub.ProblemID)
	return score.Equal(decimal.NewFromInt(100))
}

func (s *BaseAPI) isSubmissionVisible(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, user *kilonova.UserBrief) bool {
	if sub == nil {
		return false
	}

	if !s.IsAuthed(user) {
		return false
	}

	// If enabled that people see all source code
	// IsProblemFullyVisible is a workaround when a contest is running but there are submissions that were not sent in the contest
	if SubForEveryoneConfig.Value() && sub.ContestID == nil && s.IsProblemFullyVisible(user, subProblem) {
		return true
	}

	return s.subVisibleRegardless(ctx, sub, user, subProblem)
}

func (s *BaseAPI) filterSubmission(ctx context.Context, sub *kilonova.Submission, subProblem *kilonova.Problem, user *kilonova.UserBrief) {
	if sub == nil {
		return
	}
	if !s.isSubmissionVisible(ctx, sub, subProblem, user) {
		sub.Code = ""
		sub.CompileMessage = nil
		sub.CodeSize = 0
	}
	if !s.IsProblemEditor(user, subProblem) {
		sub.CompileTime = nil
	}
}

func (s *BaseAPI) CreatePaste(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief) (string, *StatusError) {
	if !PastesEnabled.Value() {
		return "", kilonova.ErrFeatureDisabled
	}
	paste := &kilonova.SubmissionPaste{Submission: sub, Author: user}
	if err := s.db.CreatePaste(ctx, paste); err != nil {
		return "", WrapError(err, "Couldn't create paste")
	}
	return paste.ID, nil
}

func (s *BaseAPI) SubmissionPaste(ctx context.Context, id string) (*kilonova.SubmissionPaste, *StatusError) {
	if !PastesEnabled.Value() {
		return nil, kilonova.ErrFeatureDisabled
	}
	paste, err := s.db.SubmissionPaste(ctx, id)
	if err != nil {
		return nil, WrapError(err, "Couldn't get paste")
	}
	if paste == nil {
		return nil, Statusf(404, "Couldn't find paste")
	}
	return paste, nil
}

func (s *BaseAPI) DeletePaste(ctx context.Context, id string) *StatusError {
	if !PastesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if err := s.db.DeleteSubPaste(ctx, id); err != nil {
		return WrapError(err, "Couldn't delete paste")
	}
	return nil
}

func (s *BaseAPI) SubmissionSubTasks(ctx context.Context, subID int) ([]*kilonova.SubmissionSubTask, *StatusError) {
	subs, err := s.db.SubmissionSubTasksBySubID(ctx, subID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get submission subtasks")
	}
	return subs, nil
}

func (s *BaseAPI) MaximumScoreSubTasks(ctx context.Context, problemID, userID int, contestID *int) ([]*kilonova.SubmissionSubTask, *StatusError) {
	subs, err := s.db.MaximumScoreSubTasks(ctx, problemID, userID, contestID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get maximum subtasks")
	}
	return subs, nil
}

func (s *BaseAPI) UpdateSubmissionSubtaskPercentage(ctx context.Context, id int, percentage decimal.Decimal) *kilonova.StatusError {
	if err := s.db.UpdateSubmissionSubtaskPercentage(ctx, id, percentage); err != nil {
		return WrapError(err, "Couldn't update subtask percentage")
	}
	return nil
}
