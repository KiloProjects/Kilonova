package util

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
)

// KNContextType is the string type for all context values
type KNContextType string

const (
	// UserKey is the key to be used for adding user objects to context
	UserKey = KNContextType("user")
	// ProblemKey is the key to be used for adding problems to context
	ProblemKey = KNContextType("problem")
	// SubKey is the key to be used for adding submissions to context
	SubKey = KNContextType("submission")
	// TestKey is the key to be used for adding tests to context
	TestKey = KNContextType("test")
	// SubTaskKey is the key to be used for adding subtasks to context
	SubTaskKey = KNContextType("test")
	// ProblemListKey is the key to be used for adding problem lists to context
	ProblemListKey = KNContextType("problemList")
)

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

func ProblemList(r *http.Request) *kilonova.ProblemList {
	switch v := r.Context().Value(ProblemListKey).(type) {
	case kilonova.ProblemList:
		return &v
	case *kilonova.ProblemList:
		return v
	default:
		return nil
	}
}

func SubTask(r *http.Request) *kilonova.SubTask {
	switch v := r.Context().Value(SubTaskKey).(type) {
	case kilonova.SubTask:
		return &v
	case *kilonova.SubTask:
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
