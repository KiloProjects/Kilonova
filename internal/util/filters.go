package util

import (
	"context"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/db"
)

// this file stores stuff to both the server and web parts

// CONVENTION: IsR* is shorthand for getting the required stuff from request and passing it to its non-R counterpart

func IsAuthed(user db.User) bool {
	return user.ID != 0
}

func IsAdmin(user db.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin
}

func IsProposer(user db.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin || user.Proposer
}

func IsProblemEditor(user db.User, problem db.Problem) bool {
	if !IsAuthed(user) {
		return false
	}
	if IsAdmin(user) {
		return true
	}
	return user.ID == problem.AuthorID
}

func IsProblemVisible(user db.User, problem db.Problem) bool {
	if problem.Visible {
		return true
	}
	return IsProblemEditor(user, problem)
}

func IsSubmissionEditor(sub db.Submission, user db.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return IsAdmin(user) || user.ID == sub.UserID
}

func IsSubmissionVisible(sub db.Submission, user db.User) bool {
	if sub.Visible {
		return true
	}
	return IsSubmissionEditor(sub, user)
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

func IsRSubmissionVisible(r *http.Request) bool {
	return IsSubmissionVisible(Submission(r), User(r))
}

func Visible(kdb *db.Queries, ctx context.Context, user db.User) ([]db.Problem, error) {
	if user.Admin {
		return kdb.Problems(ctx)
	}
	return kdb.VisibleProblems(ctx, user.ID)
}
