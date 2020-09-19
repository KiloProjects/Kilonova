package util

import (
	"context"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/db"
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

func IDFromContext(r *http.Request, tp KNContextType) int64 {
	switch v := r.Context().Value(tp).(type) {
	case int:
		return int64(v)
	case uint:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return -1
	}
}

// UserFromContext returns the user from request context
func UserFromContext(r *http.Request) db.User {
	switch v := r.Context().Value(UserKey).(type) {
	case db.User:
		return v
	case *db.User:
		return *v
	default:
		return db.User{}
	}
}

// ProblemFromContext returns the problem from request context
func ProblemFromContext(r *http.Request) db.Problem {
	switch v := r.Context().Value(ProblemKey).(type) {
	case db.Problem:
		return v
	case *db.Problem:
		return *v
	default:
		return db.Problem{}
	}
}

// SubmissionFromContext returns the submission from request context
func SubmissionFromContext(r *http.Request) db.Submission {
	switch v := r.Context().Value(SubKey).(type) {
	case db.Submission:
		return v
	case *db.Submission:
		return *v
	default:
		return db.Submission{}
	}
}

// TestFromContext returns the test from request context
func TestFromContext(r *http.Request) db.Test {
	switch v := r.Context().Value(TestKey).(type) {
	case db.Test:
		return v
	case *db.Test:
		return *v
	default:
		return db.Test{}
	}
}

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

func GetVisible(kdb *db.Queries, ctx context.Context, user db.User) ([]db.Problem, error) {
	if user.Admin {
		return kdb.Problems(ctx)
	}
	return kdb.VisibleProblems(ctx, user.ID)
}
