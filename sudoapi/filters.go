package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

// NOTE: This must be in sync with the visible_posts PSQL function
// TODO: Refactor into method of *kilonova.BlogPost
func (s *BaseAPI) IsBlogPostVisible(user *kilonova.UserBrief, post *kilonova.BlogPost) bool {
	if post == nil {
		return false
	}
	if post.Visible {
		return true
	}
	if user.IsAdmin() {
		return true
	}
	userID := 0
	if user != nil {
		userID = user.ID
	}

	return userID == post.AuthorID
}

func (s *BaseAPI) IsBlogPostEditor(user *kilonova.UserBrief, post *kilonova.BlogPost) bool {
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
		return true
	}
	if post == nil {
		return false
	}
	return post.AuthorID == user.ID
}

func (s *BaseAPI) IsProblemEditor(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
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

// Full visibility is currently used for:
//   - problem statistics;
//   - submission code visibility;
//   - problem archive availability (however, some stuff like private attachments or tests depend on further privileges).
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

func (s *BaseAPI) CanViewTests(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if !user.IsAuthed() {
		return false
	}

	if problem.VisibleTests {
		return true
	}

	return s.IsProblemEditor(user, problem)
}

// TODO: Refactor into method of *kilonova.Contest
func (s *BaseAPI) IsContestEditor(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
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
// TODO: Refactor into method of *kilonova.Contest
func (s *BaseAPI) IsContestTester(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
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
	if user.IsAdmin() {
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

// TODO: Refactor into method of *kilonova.Submission
func (s *BaseAPI) IsSubmissionEditor(sub *kilonova.Submission, user *kilonova.UserBrief) bool {
	if !user.IsAuthed() {
		return false
	}
	if sub == nil {
		return false
	}
	return user.IsAdmin() || user.ID == sub.UserID
}

// TODO: Refactor into method of *kilonova.SubmissionPaste
func (s *BaseAPI) IsPasteEditor(paste *kilonova.SubmissionPaste, user *kilonova.UserBrief) bool {
	if !user.IsAuthed() {
		return false
	}
	if paste == nil {
		return false
	}
	return s.IsSubmissionEditor(paste.Submission, user) || user.ID == paste.Author.ID
}

// The leaderboards are not frozen if:
//   - No freeze time is set
//   - Current moment is before freeze time
//   - User is contest editor
//
// Otherwise, leaderboard should be frozen
// Also, in case the editor wants to see the frozen leaderboard, an option is provided
func (s *BaseAPI) UserContestFreezeTime(user *kilonova.UserBrief, contest *kilonova.Contest, showFrozen bool) *time.Time {
	if contest.LeaderboardFreeze == nil {
		return nil
	}
	if time.Now().Before(*contest.LeaderboardFreeze) {
		return nil
	}
	if s.IsContestEditor(user, contest) {
		if showFrozen {
			return contest.LeaderboardFreeze
		}
		return nil
	}
	return contest.LeaderboardFreeze
}
