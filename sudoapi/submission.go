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

func (s *BaseAPI) MaxScore(ctx context.Context, uid, pbID int) int {
	return s.db.MaxScore(ctx, uid, pbID)
}

func (s *BaseAPI) MaxContestScore(ctx context.Context, uid, pbID, contestID int) int {
	return s.db.ContestMaxScore(ctx, uid, pbID, contestID)
}

func (s *BaseAPI) MaxScores(ctx context.Context, uid int, pbIDs []int) map[int]int {
	return s.db.MaxScores(ctx, uid, pbIDs)
}

func (s *BaseAPI) NumSolved(ctx context.Context, uid int, pbIDs []int) int {
	scores := s.MaxScores(context.Background(), uid, pbIDs)
	var rez int
	for _, v := range scores {
		if v == 100 {
			rez++
		}
	}
	return rez
}

func (s *BaseAPI) Submissions(ctx context.Context, filter kilonova.SubmissionFilter, lookingUser *UserBrief) (*Submissions, *StatusError) {
	if filter.Limit == 0 || filter.Limit > 50 {
		filter.Limit = 50
	}

	if filter.Ordering == "" {
		filter.Ordering = "id"
	}

	subs, err := s.db.Submissions(ctx, filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, ErrUnknownError
		}
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	cnt, err1 := s.CountSubmissions(ctx, filter)
	if err1 != nil {
		zap.S().Warn(err1)
		return nil, err1
	}

	users := make(map[int]*UserBrief)
	problems := make(map[int]*kilonova.Problem)

	for i, sub := range subs {
		if _, ok := users[sub.UserID]; !ok {
			user, err := s.UserBrief(ctx, sub.UserID)
			if err != nil {
				zap.S().Infof("Error getting user %d: %v", sub.UserID, err)
				continue
			}
			users[sub.UserID] = user
		}

		if _, ok := problems[sub.ProblemID]; !ok {
			problem, err := s.Problem(ctx, sub.ProblemID)
			if err != nil || problem == nil {
				zap.S().Infof("Error getting problem %d: %v", sub.ProblemID, err)
				continue
			}
			if s.IsProblemVisible(lookingUser, problem) {
				problems[sub.ProblemID] = problem
			}
		}

		s.filterSubmission(ctx, subs[i], lookingUser)
	}

	return &Submissions{
		Submissions: subs,
		Count:       cnt,
		Users:       users,
		Problems:    problems,
	}, nil
}

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

func (s *BaseAPI) RawSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) ([]*kilonova.Submission, *StatusError) {
	subs, err := s.db.Submissions(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return subs, nil
}

func (s *BaseAPI) CountSubmissions(ctx context.Context, filter kilonova.SubmissionFilter) (int, *StatusError) {
	count, err := s.db.CountSubmissions(ctx, filter)
	if err != nil {
		return -1, WrapError(err, "Couldn't count submissions")
	}
	return count, nil
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
	sub, err := s.db.Submission(ctx, subid)
	if err != nil || sub == nil {
		return nil, Statusf(404, "Submission not found")
	}
	if isLooking {
		s.filterSubmission(ctx, sub, lookingUser)
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
		zap.S().Warn(err)
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
	if !s.IsProblemVisible(author, problem) {
		return -1, Statusf(400, "Submitter can't see the problem!")
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

	// Initialize subtests
	if err := s.db.InitSubTests(ctx, author.ID, id, problem.ID); err != nil {
		zap.S().Warn("Couldn't create submission tests:", err)
		return -1, Statusf(500, "Couldn't create submission tests")
	}

	// After subtests, initialize subtasks
	if err := s.db.InitSubmissionSubtasks(ctx, author.ID, id, problem.ID); err != nil {
		zap.S().Warn("Couldn't create submission subtasks:", err)
		return -1, Statusf(500, "Couldn't create submission subtasks")
	}

	if err := s.db.UpdateSubmission(ctx, id, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		zap.S().Warn("Couldn't update submission status:", err)
		return -1, Statusf(500, "Failed to create submission")
	}

	return id, nil
}

func (s *BaseAPI) DeleteSubmission(ctx context.Context, subID int) *StatusError {
	if err := s.db.DeleteSubmission(ctx, subID); err != nil {
		return Statusf(500, "Failed to delete submission")
	}
	return nil
}

// Feature request by liviu
func (s *BaseAPI) ResetProblemSubmissions(ctx context.Context, pbid int) *StatusError {
	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{ProblemID: &pbid}, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}
	s.LogUserAction(ctx, "Reset submissions for problem %d", pbid)
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
