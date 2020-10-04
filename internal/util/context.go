package util

import (
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/db"
)

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
