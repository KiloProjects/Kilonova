package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) IsAuthed(user *kilonova.UserBrief) bool {
	return user != nil && user.ID != 0
}

func (s *BaseAPI) IsAdmin(user *kilonova.UserBrief) bool {
	if !s.IsAuthed(user) {
		return false
	}
	return user.Admin
}

func (s *BaseAPI) IsProposer(user *kilonova.UserBrief) bool {
	if !s.IsAuthed(user) {
		return false
	}
	return user.Admin || user.Proposer
}

func (s *BaseAPI) IsProblemEditor(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if s.IsAdmin(user) {
		return true
	}
	if problem == nil {
		return false
	}
	ok, err := s.db.IsProblemEditor(context.Background(), problem.ID, user.ID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	return ok
}

func (s *BaseAPI) IsProblemVisible(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if problem.Visible {
		return true
	}
	userID := 0
	if user != nil {
		userID = user.ID
	}

	ok, err := s.db.IsProblemViewer(context.Background(), problem.ID, userID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	return ok
}

// Full visibility is currently used for problem statistics
func (s *BaseAPI) IsProblemFullyVisible(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if problem.Visible {
		return true
	}
	userID := 0
	if user != nil {
		userID = user.ID
	}

	ok, err := s.db.IsFullProblemViewer(context.Background(), problem.ID, userID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	return ok
}

func (s *BaseAPI) IsContestEditor(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if s.IsAdmin(user) {
		return true
	}
	if contest == nil {
		return false
	}

	for _, editor := range contest.Editors {
		if editor.ID == user.ID {
			return true
		}
	}
	return false
}

// Tester = Testers + Editors + Admins
func (s *BaseAPI) IsContestTester(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if s.IsAdmin(user) {
		return true
	}
	if contest == nil {
		return false
	}

	for _, editor := range contest.Editors {
		if editor.ID == user.ID {
			return true
		}
	}
	for _, tester := range contest.Testers {
		if tester.ID == user.ID {
			return true
		}
	}
	return false
}

// TODO: This only accounts for editors/testers
// This should also account for those that are registered and the contest is running
func (s *BaseAPI) IsContestVisible(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if s.IsAdmin(user) {
		return true
	}
	if contest == nil {
		return false
	}
	userID := 0
	if user != nil {
		userID = user.ID
	}

	ok, err := s.db.IsContestViewer(context.Background(), contest.ID, userID)
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	return ok
}

func (s *BaseAPI) FilterVisibleProblems(user *kilonova.UserBrief, pbs []*kilonova.ScoredProblem) []*kilonova.ScoredProblem {
	vpbs := make([]*kilonova.ScoredProblem, 0, len(pbs))
	for _, pb := range pbs {
		if s.IsProblemVisible(user, &pb.Problem) {
			vpbs = append(vpbs, pb)
		}
	}
	return vpbs
}

func (s *BaseAPI) IsSubmissionEditor(sub *kilonova.Submission, user *kilonova.UserBrief) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if sub == nil {
		return false
	}
	return s.IsAdmin(user) || user.ID == sub.UserID
}

func (s *BaseAPI) IsPasteEditor(paste *kilonova.SubmissionPaste, user *kilonova.UserBrief) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if paste == nil {
		return false
	}
	return s.IsSubmissionEditor(paste.Submission, user) || user.ID == paste.Author.ID
}
