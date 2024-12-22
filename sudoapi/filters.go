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
//   - tag visibility;
//   - ability to add problems to contests;
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

	if problem.VisibleTests && s.IsProblemFullyVisible(user, problem) {
		return true
	}

	return s.IsProblemEditor(user, problem)
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

	if contest.IsEditor(user) {
		if showFrozen {
			return contest.LeaderboardFreeze
		}
		return nil
	}
	return contest.LeaderboardFreeze
}
