package util

import (
	"database/sql"
	"net/http"

	"github.com/KiloProjects/kilonova"
)

// this file stores stuff to both the server and web parts

// CONVENTION: IsR* is shorthand for getting the required stuff from request and passing it to its non-R counterpart

func IsAuthed(user *kilonova.UserBrief) bool {
	return user != nil && user.ID != 0
}

func IsAdmin(user *kilonova.UserBrief) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin
}

func IsProposer(user *kilonova.UserBrief) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin || user.Proposer
}

func IsProblemEditor(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if !IsAuthed(user) {
		return false
	}
	if IsAdmin(user) {
		return true
	}
	if problem == nil {
		return false
	}
	return user.ID == problem.AuthorID
}

func IsProblemVisible(user *kilonova.UserBrief, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if problem.Visible {
		return true
	}
	return IsProblemEditor(user, problem)
}

func IsSubmissionEditor(sub *kilonova.Submission, user *kilonova.UserBrief) bool {
	if !IsAuthed(user) {
		return false
	}
	if sub == nil {
		return false
	}
	return IsAdmin(user) || user.ID == sub.UserID
}

func IsSubmissionVisible(sub *kilonova.Submission, user *kilonova.UserBrief) bool {
	if sub == nil {
		return false
	}
	if IsSubmissionEditor(sub, user) {
		return true
	}

	if !IsAuthed(user) {
		return false
	}

	return false
}

// FilterSubmission controls what happens to a submission when it is viewed by a user
func FilterSubmission(sub *kilonova.Submission, user *kilonova.UserBrief) {
	if sub != nil && !IsSubmissionVisible(sub, user) {
		sub.Code = ""
		sub.CompileMessage = sql.NullString{}
	}
}

func IsRAuthed(r *http.Request) bool {
	return IsAuthed(UserBrief(r))
}

func IsRAdmin(r *http.Request) bool {
	return IsAdmin(UserBrief(r))
}

func IsRProposer(r *http.Request) bool {
	return IsProposer(UserBrief(r))
}

func IsRProblemEditor(r *http.Request) bool {
	return IsProblemEditor(UserBrief(r), Problem(r))
}

func IsRProblemVisible(r *http.Request) bool {
	return IsProblemVisible(UserBrief(r), Problem(r))
}

func FilterVisible(user *kilonova.UserBrief, pbs []*kilonova.Problem) []*kilonova.Problem {
	vpbs := make([]*kilonova.Problem, 0, len(pbs))
	for _, pb := range pbs {
		if IsProblemVisible(user, pb) {
			vpbs = append(vpbs, pb)
		}
	}
	return vpbs
}
