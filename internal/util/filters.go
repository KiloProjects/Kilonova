package util

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
)

// this file stores stuff to both the server and web parts

// CONVENTION: IsR* is shorthand for getting the required stuff from request and passing it to its non-R counterpart

func IsAuthed(user *kilonova.User) bool {
	return user != nil && user.ID != 0
}

func IsAdmin(user *kilonova.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin
}

func IsProposer(user *kilonova.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin || user.Proposer
}

func IsProblemEditor(user *kilonova.User, problem *kilonova.Problem) bool {
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

func IsProblemVisible(user *kilonova.User, problem *kilonova.Problem) bool {
	if problem == nil {
		return false
	}
	if problem.Visible {
		return true
	}
	return IsProblemEditor(user, problem)
}

func IsSubmissionEditor(sub *kilonova.Submission, user *kilonova.User) bool {
	if !IsAuthed(user) {
		return false
	}
	if sub == nil {
		return false
	}
	return IsAdmin(user) || user.ID == sub.UserID
}

func IsSubmissionVisible(sub *kilonova.Submission, user *kilonova.User, sserv kilonova.SubmissionService) bool {
	if sub == nil {
		return false
	}
	if sub.Visible || IsSubmissionEditor(sub, user) {
		return true
	}

	if !IsAuthed(user) {
		return false
	}
	score := sserv.MaxScore(context.Background(), user.ID, sub.ProblemID)
	if score == 100 {
		return true
	}

	return false
}

func IsRAuthed(r *http.Request) bool {
	return IsAuthed(User(r))
}

func IsRAdmin(r *http.Request) bool {
	return IsAdmin(User(r))
}

func IsRProposer(r *http.Request) bool {
	return IsProposer(User(r))
}

func IsRProblemEditor(r *http.Request) bool {
	return IsProblemEditor(User(r), Problem(r))
}

func IsRProblemVisible(r *http.Request) bool {
	return IsProblemVisible(User(r), Problem(r))
}

func IsRSubmissionEditor(r *http.Request) bool {
	return IsSubmissionEditor(Submission(r), User(r))
}

func IsRSubmissionVisible(r *http.Request, sserv kilonova.SubmissionService) bool {
	return IsSubmissionVisible(Submission(r), User(r), sserv)
}
