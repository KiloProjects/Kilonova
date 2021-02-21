package util

import (
	"log"
	"net/http"
	"reflect"

	"github.com/KiloProjects/kilonova"
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

// ID returns the ID with that specific key from context
func ID(r *http.Request, tp KNContextType) int {
	switch v := r.Context().Value(tp).(type) {
	case int:
		return v
	case uint:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case nil:
		return 0
	default:
		log.Println("ID with type:", reflect.TypeOf(v))
		return -1
	}
}

// User returns the user from request context
func User(r *http.Request) *kilonova.User {
	switch v := r.Context().Value(UserKey).(type) {
	case kilonova.User:
		return &v
	case *kilonova.User:
		return v
	default:
		return nil
	}
}

// Problem returns the problem from request context
func Problem(r *http.Request) *kilonova.Problem {
	switch v := r.Context().Value(ProblemKey).(type) {
	case kilonova.Problem:
		return &v
	case *kilonova.Problem:
		return v
	default:
		return nil
	}
}

// Submission returns the submission from request context
func Submission(r *http.Request) *kilonova.Submission {
	switch v := r.Context().Value(SubKey).(type) {
	case kilonova.Submission:
		return &v
	case *kilonova.Submission:
		return v
	default:
		return nil
	}
}

// Test returns the test from request context
func Test(r *http.Request) *kilonova.Test {
	switch v := r.Context().Value(TestKey).(type) {
	case kilonova.Test:
		return &v
	case *kilonova.Test:
		return v
	default:
		return nil
	}
}
