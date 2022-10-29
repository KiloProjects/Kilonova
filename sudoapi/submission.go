package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

// Submission stuff

func (s *BaseAPI) MaxScore(ctx context.Context, uid, pbid int) int {
	return s.db.MaxScore(ctx, uid, pbid)
}

func (s *BaseAPI) MaxScores(ctx context.Context, uid int, pbIDs []int) map[int]int {
	return s.db.MaxScores(ctx, uid, pbIDs)
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
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	cnt, err := s.db.CountSubmissions(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}

	users := make(map[int]*UserBrief)
	problems := make(map[int]*kilonova.Problem)

	for i, sub := range subs {
		if _, ok := users[sub.UserID]; !ok {
			user, err := s.UserBrief(ctx, sub.UserID)
			if err != nil {
				zap.S().Infof("Error getting user %d: %v\n", sub.UserID, err)
				continue
			}
			users[sub.UserID] = user
		}

		if _, ok := problems[sub.ProblemID]; !ok {
			problem, err := s.Problem(ctx, sub.ProblemID)
			if err != nil || problem == nil {
				zap.S().Infof("Error getting problem %d: %v\n", sub.ProblemID, err)
				continue
			}
			if util.IsProblemVisible(lookingUser, problem) {
				problems[sub.ProblemID] = problem
			}
		}

		util.FilterSubmission(subs[i], lookingUser)
	}

	return &Submissions{
		Submissions: subs,
		Count:       cnt,
		Users:       users,
		Problems:    problems,
	}, nil
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

type SubTest struct {
	kilonova.SubTest
	Test *kilonova.Test `json:"test"`
}

type FullSubmission struct {
	kilonova.Submission
	Author   *UserBrief          `json:"author"`
	Problem  *kilonova.Problem   `json:"problem"`
	SubTests []*SubTest          `json:"subtests"`
	SubTasks []*kilonova.SubTask `json:"subtasks"`

	// ProblemEditor returns wether the looking user is a problem editor
	ProblemEditor bool `json:"problem_editor"`
}

func (s *BaseAPI) Submission(ctx context.Context, subid int, lookingUser *UserBrief) (*FullSubmission, *StatusError) {
	sub, err := s.db.Submission(ctx, subid)
	if err != nil || sub == nil {
		return nil, Statusf(404, "Submission not found")
	}
	util.FilterSubmission(sub, lookingUser)

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
	if !util.IsProblemVisible(lookingUser, rez.Problem) {
		return nil, Statusf(403, "Submission hidden because problem is not visible.")
	}

	rez.ProblemEditor = util.IsProblemEditor(lookingUser, rez.Problem)

	rawSubtests, err := s.db.SubTestsBySubID(ctx, subid)
	if err != nil {
		return nil, Statusf(500, "Couldn't fetch subtests")
	}
	for _, test := range rawSubtests {
		t, err := s.db.TestByID(ctx, test.TestID)
		if err != nil {
			// TODO: Maybe don't be so pedantic?
			return nil, Statusf(500, "Couldn't fetch subtests' tests")
		}
		rez.SubTests = append(rez.SubTests, &SubTest{SubTest: *test, Test: t})
	}

	rez.SubTasks, err = s.db.SubTasks(ctx, sub.ProblemID)
	if err != nil {
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

// CreateSubmission produces a new submission and also creates the necessary subtests
func (s *BaseAPI) CreateSubmission(ctx context.Context, author *UserBrief, problem *kilonova.Problem, code string, lang eval.Language) (int, *StatusError) {
	if author == nil {
		return -1, Statusf(400, "Invalid submission author")
	}
	if problem == nil {
		return -1, Statusf(400, "Invalid submission problem")
	}
	if !util.IsProblemVisible(author, problem) {
		return -1, Statusf(400, "User can't see the problem!")
	}

	if code == "" {
		return -1, Statusf(400, "Empty code")
	}

	tests, err := s.db.Tests(ctx, problem.ID)
	if err != nil {
		zap.S().Warn("Couldn't get problem tests for submission:", err)
		return -1, Statusf(500, "Couldn't fetch problem tests")
	}

	// Add submission
	id, err := s.db.CreateSubmission(ctx, author.ID, problem, lang, code)
	if err != nil {
		zap.S().Warn("Couldn't create submission:", err)
		return -1, Statusf(500, "Couldn't create submission")
	}

	// Add subtests
	for _, test := range tests {
		if err := s.db.CreateSubTest(ctx, &kilonova.SubTest{UserID: author.ID, TestID: test.ID, SubmissionID: id}); err != nil {
			zap.S().Warn("Couldn't create submission test:", err)
			return -1, Statusf(500, "Couldn't create submission test")
		}
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

/*

// GraderSubmission returns a waiting submission that is not locked. Note that it must be closed
// TODO: Revamp this, it's not good
func (s *BaseAPI) GraderSubmission(ctx context.Context) (eval.GraderSubmission, *StatusError) {
	sub, err := s.db.FetchGraderSubmission(ctx)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return sub, nil
}

*/
