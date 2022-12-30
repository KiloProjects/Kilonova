package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
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
	for _, uid := range problem.Editors {
		if uid == user.ID {
			return true
		}
	}

	// May be contest editor though
	// TODO: Maybe require valid context?
	contests, err := s.ProblemContests(context.Background(), problem.ID)
	if err == nil {
		for _, contest := range contests {
			if s.IsContestEditor(user, contest) {
				return true
			}
		}
	}
	return false
}

func (s *BaseAPI) IsProblemVisible(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if problem.Visible {
		return true
	}
	if user != nil {
		for _, uid := range problem.Viewers {
			if uid == user.ID {
				return true
			}
		}
	}

	// May be contest editor though
	// TODO: Maybe require valid context?
	contests, err := s.ProblemContests(context.Background(), problem.ID)
	if err == nil {
		for _, contest := range contests {
			if s.IsContestVisible(user, contest) {
				return true
			}
		}
	}

	return s.IsProblemEditor(user, problem)
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

// TODO: This only accounts for editors/testers
// This should also account for those that are registered and the contest is running
func (s *BaseAPI) IsContestVisible(user *kilonova.UserBrief, contest *kilonova.Contest) bool {
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

func (s *BaseAPI) FilterVisibleProblems(user *kilonova.UserBrief, pbs []*kilonova.Problem) []*kilonova.Problem {
	vpbs := make([]*kilonova.Problem, 0, len(pbs))
	for _, pb := range pbs {
		if s.IsProblemVisible(user, pb) {
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
