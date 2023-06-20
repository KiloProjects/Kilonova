package sudoapi

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

// Submission stuff

func (s *BaseAPI) MaxScoreSubID(ctx context.Context, uid, pbID int) (int, *StatusError) {
	id, err := s.db.MaxScoreSubID(ctx, uid, pbID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get max score ID")
	}
	return id, nil
}

func (s *BaseAPI) MaxScore(ctx context.Context, uid, pbID int) int {
	return s.db.MaxScore(ctx, uid, pbID)
}

func (s *BaseAPI) ContestMaxScore(ctx context.Context, uid, pbID, contestID int) int {
	return s.db.ContestMaxScore(ctx, uid, pbID, contestID)
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

		if _, ok := problemsMap[sub.ProblemID]; !ok {
			zap.S().Warnf("Couldn't find problem %d in map. Something has gone terribly wrong", sub.ProblemID)
		}

		if look {
			s.filterSubmission(ctx, subs[i], lookingUser)
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

type FullSubmission struct {
	kilonova.Submission
	Author   *UserBrief          `json:"author"`
	Problem  *kilonova.Problem   `json:"problem"`
	SubTests []*kilonova.SubTest `json:"subtests"`

	SubTasks []*kilonova.SubmissionSubTask `json:"subtasks"`

	// ProblemEditor returns whether the looking user is a problem editor
	ProblemEditor bool `json:"problem_editor"`
}

func (s *BaseAPI) Submission(ctx context.Context, subid int, lookingUser *UserBrief) (*FullSubmission, *StatusError) {
	return s.getSubmission(ctx, subid, lookingUser, true)
}

// FullSubmission gets the submission regardless of if there is a user watching or not
func (s *BaseAPI) FullSubmission(ctx context.Context, subid int) (*FullSubmission, *StatusError) {
	return s.getSubmission(ctx, subid, nil, false)
}

func (s *BaseAPI) getSubmission(ctx context.Context, subid int, lookingUser *UserBrief, isLooking bool) (*FullSubmission, *StatusError) {
	var sub *kilonova.Submission
	if isLooking {
		var userID int = 0
		if lookingUser != nil {
			userID = lookingUser.ID
		}
		sub2, err := s.db.SubmissionLookingUser(ctx, subid, userID)
		if err != nil || sub2 == nil {
			return nil, Statusf(404, "Submission not found or user may not have access")
		}

		s.filterSubmission(ctx, sub2, lookingUser)
		sub = sub2
	} else {
		sub2, err := s.db.Submission(ctx, subid)
		if err != nil || sub2 == nil {
			return nil, Statusf(404, "Submission not found")
		}
		sub = sub2
	}

	rez := &FullSubmission{Submission: *sub}
	author, err1 := s.UserBrief(ctx, sub.UserID)
	if err1 != nil {
		return nil, err1
	}
	rez.Author = author

	rez.Problem, err1 = s.Problem(ctx, sub.ProblemID)
	if err1 != nil {
		return nil, err1
	}
	if isLooking && !s.IsProblemVisible(lookingUser, rez.Problem) {
		return nil, Statusf(403, "Submission hidden because problem is not visible.")
	}

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

// CreateSubmission produces a new submission and also creates the necessary subtests
func (s *BaseAPI) CreateSubmission(ctx context.Context, author *UserBrief, problem *kilonova.Problem, code string, lang eval.Language, contestID *int) (int, *StatusError) {
	if author == nil {
		return -1, Statusf(400, "Invalid submission author")
	}
	if problem == nil {
		return -1, Statusf(400, "Invalid submission problem")
	}
	if len(code) > problem.SourceSize { // Maximum admitted by problem
		return -1, Statusf(400, "Code exceeds %d characters", problem.SourceSize)
	}
	if !s.IsProblemVisible(author, problem) {
		return -1, Statusf(400, "Submitter can't see the problem!")
	}

	cnt, err := s.db.WaitingSubmissionCount(ctx, author.ID)
	if err != nil {
		return -1, Statusf(500, "Couldn't get unfinished submission count")
	}

	if cnt > 5 {
		return -1, Statusf(400, "You cannot have more than 5 submissions in the evaluation queue at once")
	}

	if contestID != nil {
		contest, err := s.Contest(ctx, *contestID)
		if err != nil || !s.IsContestVisible(author, contest) {
			return -1, Statusf(404, "Couldn't find contest")
		}
		if !s.CanSubmitInContest(author, contest) {
			return -1, Statusf(400, "Submitter cannot submit to contest")
		}
		pb, err := s.ContestProblem(ctx, contest, author, problem.ID)
		if err != nil || pb == nil {
			return -1, Statusf(400, "Problem is not in contest")
		}
		cnt, err := s.RemainingSubmissionCount(ctx, contest, pb, author)
		if err != nil {
			return -1, err
		}
		if cnt <= 0 {
			return -1, Statusf(400, "Max submission count for problem reached")
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

	if err := s.initSubmission(ctx, id); err != nil {
		return -1, err
	}

	// Wake immediately to grade submission
	s.WakeGrader()

	return id, nil
}

func (s *BaseAPI) clearSubmission(ctx context.Context, submissionID int) *StatusError {
	if err := s.db.ClearSubTests(ctx, submissionID); err != nil {
		return Statusf(500, "Couldn't remove submission tests")
	}
	if err := s.db.ClearSubmissionSubtasks(ctx, submissionID); err != nil {
		return Statusf(500, "Couldn't remove submission tests")
	}
	zs := 0
	zm := -1
	zf := -1.0
	f := false
	e := ""
	if err := s.db.UpdateSubmission(ctx, submissionID, kilonova.SubmissionUpdate{
		Status:         kilonova.StatusCreating,
		Score:          &zs,
		MaxTime:        &zf,
		MaxMemory:      &zm,
		CompileError:   &f,
		CompileMessage: &e,
	}); err != nil {
		return WrapError(err, "Couldn't fully clear submission")
	}
	return nil
}

func (s *BaseAPI) initSubmission(ctx context.Context, subID int) *StatusError {
	// Initialize subtests
	if err := s.db.InitSubTests(ctx, subID); err != nil {
		zap.S().Warn("Couldn't create submission tests:", err)
		return Statusf(500, "Couldn't create submission tests")
	}

	// After subtests, initialize subtasks
	if err := s.db.InitSubmissionSubtasks(ctx, subID); err != nil {
		zap.S().Warn("Couldn't create submission subtasks:", err)
		return Statusf(500, "Couldn't create submission subtasks")
	}

	if err := s.db.UpdateSubmission(ctx, subID, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		zap.S().Warn("Couldn't update submission status:", err)
		return Statusf(500, "Failed to create submission")
	}
	return nil
}

func (s *BaseAPI) DeleteSubmission(ctx context.Context, subID int) *StatusError {
	if err := s.db.DeleteSubmission(ctx, subID); err != nil {
		zap.S().Warn("Couldn't delete submission:", err)
		return Statusf(500, "Failed to delete submission")
	}
	return nil
}

func (s *BaseAPI) ResetProblemSubmissions(ctx context.Context, problem *kilonova.Problem) *StatusError {
	if err := s.db.ClearProblemSubTests(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't clean submissions tests")
	}
	if err := s.db.ClearProblemSubmissionsSubtasks(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't clean submissions subtasks")
	}

	zs := 0
	zm := -1
	zf := -1.0
	f := false
	e := ""
	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &problem.ID}, kilonova.SubmissionUpdate{
		Status:         kilonova.StatusCreating,
		Score:          &zs,
		MaxTime:        &zf,
		MaxMemory:      &zm,
		CompileError:   &f,
		CompileMessage: &e,
	}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}

	if err := s.db.InitProblemSubTests(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't initialize submissions tests")
	}

	if err := s.db.InitProblemSubmissionsSubtasks(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't initialize submissions subtasks")
	}

	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &problem.ID}, kilonova.SubmissionUpdate{
		Status: kilonova.StatusWaiting,
	}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}

	s.LogUserAction(ctx, "Reset submissions for problem #%d: %s", problem.ID, problem.Name)

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) isSubmissionVisible(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief) bool {
	if sub == nil {
		return false
	}
	if s.IsSubmissionEditor(sub, user) {
		return true
	}

	if pb, err := s.Problem(ctx, sub.ProblemID); err == nil && pb != nil && s.IsProblemEditor(user, pb) {
		return true
	}

	if !s.IsAuthed(user) {
		return false
	}

	score := s.db.MaxScore(context.Background(), user.ID, sub.ProblemID)
	return score == 100
}

func (s *BaseAPI) filterSubmission(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief) {
	if sub != nil && !s.isSubmissionVisible(ctx, sub, user) {
		sub.Code = ""
		sub.CompileMessage = nil
		sub.CodeSize = 0
	}
}

func (s *BaseAPI) CreatePaste(ctx context.Context, sub *kilonova.Submission, user *kilonova.UserBrief) (string, *StatusError) {
	if !config.Features.Pastes {
		return "", kilonova.ErrFeatureDisabled
	}
	paste := &kilonova.SubmissionPaste{Submission: sub, Author: user}
	if err := s.db.CreatePaste(ctx, paste); err != nil {
		return "", WrapError(err, "Couldn't create paste")
	}
	return paste.ID, nil
}

func (s *BaseAPI) SubmissionPaste(ctx context.Context, id string) (*kilonova.SubmissionPaste, *StatusError) {
	if !config.Features.Pastes {
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
	if !config.Features.Pastes {
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

func (s *BaseAPI) UpdateSubmissionSubtaskPercentage(ctx context.Context, id int, score int) *kilonova.StatusError {
	if err := s.db.UpdateSubmissionSubtaskPercentage(ctx, id, score); err != nil {
		return WrapError(err, "Couldn't update subtask score")
	}
	return nil
}
