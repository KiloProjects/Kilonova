package util

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
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
	// AttachmentKey is the key to be used for adding attachments to context
	AttachmentKey = KNContextType("attachment")
	// Contest is the key to be used for adding contests to context
	ContestKey = KNContextType("contest")
	// LangKey is the key to be used for adding the user language to context
	LangKey = KNContextType("language")
)

func UserBriefContext(ctx context.Context) *kilonova.UserBrief {
	return getValueContext[kilonova.UserFull](ctx, UserKey).Brief()
}

func UserBrief(r *http.Request) *kilonova.UserBrief {
	return UserBriefContext(r.Context())
}

func UserFull(r *http.Request) *kilonova.UserFull {
	return getValueContext[kilonova.UserFull](r.Context(), UserKey)
}

func ProblemContext(ctx context.Context) *kilonova.Problem {
	return getValueContext[kilonova.Problem](ctx, ProblemKey)
}

func Problem(r *http.Request) *kilonova.Problem {
	return ProblemContext(r.Context())
}

func Submission(r *http.Request) *kilonova.Submission {
	return getValueContext[kilonova.Submission](r.Context(), SubKey)
}

func Test(r *http.Request) *kilonova.Test {
	return getValueContext[kilonova.Test](r.Context(), TestKey)
}

func SubTask(r *http.Request) *kilonova.SubTask {
	return getValueContext[kilonova.SubTask](r.Context(), SubTaskKey)
}

func ProblemList(r *http.Request) *kilonova.ProblemList {
	return getValueContext[kilonova.ProblemList](r.Context(), ProblemListKey)
}

func Attachment(r *http.Request) *kilonova.Attachment {
	return getValueContext[kilonova.Attachment](r.Context(), AttachmentKey)
}

func Contest(r *http.Request) *kilonova.Contest {
	return getValueContext[kilonova.Contest](r.Context(), ContestKey)
}

func Language(r *http.Request) string {
	switch v := r.Context().Value(LangKey).(type) {
	case string:
		return v
	case *string:
		return *v
	default:
		return config.Common.DefaultLang
	}
}

func getValueContext[T any](ctx context.Context, key KNContextType) *T {
	switch v := ctx.Value(key).(type) {
	case T:
		return &v
	case *T:
		return v
	default:
		return nil
	}
}
