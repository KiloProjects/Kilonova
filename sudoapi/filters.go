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

// NOTE: This must be in sync with the visible_posts PSQL function
func (s *BaseAPI) IsBlogPostVisible(user *kilonova.UserBrief, post *kilonova.BlogPost) bool {
	if post == nil {
		return false
	}
	if post.Visible {
		return true
	}
	if s.IsAdmin(user) {
		return true
	}
	userID := 0
	if user != nil {
		userID = user.ID
	}

	return userID == post.AuthorID
}

func (s *BaseAPI) IsBlogPostEditor(user *kilonova.UserBrief, post *kilonova.BlogPost) bool {
	if !s.IsAuthed(user) {
		return false
	}
	if s.IsAdmin(user) {
		return true
	}
	if post == nil {
		return false
	}
	return post.AuthorID == user.ID
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

// Full visibility is currently used for problem statistics and submission code visibility
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
