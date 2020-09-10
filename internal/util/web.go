package util

import (
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/models"
)

// this file stores stuff to both the server and web parts

// KNContextType is the string type for all context values
type KNContextType string

const (
	UserID = KNContextType("userID")
	// UserKey is the key to be used for adding user objects to context
	UserKey = KNContextType("user")
	// PbID is the key to be used for adding problem IDs to context
	PbID = KNContextType("pbID")
	// ProblemKey is the key to be used for adding problems to context
	ProblemKey = KNContextType("problem")
	// SubID  is the key to be used for adding submission IDs to context
	SubID = KNContextType("subID")
	// SubKey is the key to be used for adding submissions to context
	SubKey = KNContextType("submission")
	// SubEditorKey is the key to be used for adding the submission editor bool to context
	SubEditorKey = KNContextType("subEditor")
	// TestKey is the key to be used for adding test IDs to context
	TestID = KNContextType("testID")
	// TestKey is the key to be used for adding tests to context
	TestKey = KNContextType("test")
)

// RetData should be the way data is sent between the API and the Client
type RetData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func IDFromContext(r *http.Request, tp KNContextType) uint {
	switch v := r.Context().Value(tp).(type) {
	case uint:
		return v
	default:
		return 99999
	}
}

// UserFromContext returns the user from request context
func UserFromContext(r *http.Request) models.User {
	switch v := r.Context().Value(UserKey).(type) {
	case models.User:
		return v
	case *models.User:
		return *v
	default:
		return models.User{}
	}
}

// ProblemFromContext returns the problem from request context
func ProblemFromContext(r *http.Request) models.Problem {
	switch v := r.Context().Value(ProblemKey).(type) {
	case models.Problem:
		return v
	case *models.Problem:
		return *v
	default:
		return models.Problem{}
	}
}

// SubmissionFromContext returns the submission from request context
func SubmissionFromContext(r *http.Request) models.Submission {
	switch v := r.Context().Value(SubKey).(type) {
	case models.Submission:
		return v
	case *models.Submission:
		return *v
	default:
		return models.Submission{}
	}
}

// TestFromContext returns the test from request context
func TestFromContext(r *http.Request) models.Test {
	switch v := r.Context().Value(TestKey).(type) {
	case models.Test:
		return v
	case *models.Test:
		return *v
	default:
		return models.Test{}
	}
}

// CONVENTION: IsR* is shorthand for getting the required stuff from request and passing it to its non-R counterpart

func IsAuthed(user models.User) bool {
	return user.ID != 0
}

func IsAdmin(user models.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin
}

func IsProposer(user models.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return user.Admin || user.Proposer
}

func IsProblemEditor(user models.User, problem models.Problem) bool {
	if !IsAuthed(user) {
		return false
	}
	if IsAdmin(user) {
		return true
	}
	return user.ID == problem.UserID
}

func IsProblemVisible(user models.User, problem models.Problem) bool {
	if problem.Visible {
		return true
	}
	return IsProblemEditor(user, problem)
}

func IsSubmissionEditor(sub models.Submission, user models.User) bool {
	if !IsAuthed(user) {
		return false
	}
	return IsAdmin(user) || user.ID == sub.UserID
}

func IsSubmissionVisible(sub models.Submission, user models.User) bool {
	if sub.Visible {
		return true
	}
	return IsSubmissionEditor(sub, user)
}

func IsRAuthed(r *http.Request) bool {
	return IsAuthed(UserFromContext(r))
}

func IsRAdmin(r *http.Request) bool {
	return IsAdmin(UserFromContext(r))
}

func IsRProposer(r *http.Request) bool {
	return IsProposer(UserFromContext(r))
}

func IsRProblemEditor(r *http.Request) bool {
	return IsProblemEditor(UserFromContext(r), ProblemFromContext(r))
}

func IsRProblemVisible(r *http.Request) bool {
	return IsProblemVisible(UserFromContext(r), ProblemFromContext(r))
}

func IsRSubmissionEditor(r *http.Request) bool {
	return IsSubmissionEditor(SubmissionFromContext(r), UserFromContext(r))
}

func IsRSubmissionVisible(r *http.Request) bool {
	return IsSubmissionVisible(SubmissionFromContext(r), UserFromContext(r))
}

func FilterVisible(problems []models.Problem, user models.User) []models.Problem {
	var showedProblems []models.Problem
	for _, pb := range problems {
		if IsProblemVisible(user, pb) {
			showedProblems = append(showedProblems, pb)
		}
	}
	return showedProblems
}
