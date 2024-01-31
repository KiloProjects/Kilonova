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
	// AuthedUserKey is the key to be used for adding (AUTHENTICATED) user objects to context
	AuthedUserKey = KNContextType("authed")
	// AuthedUserKey is the key to be used for adding CONTENT user objects to context
	ContentUserKey = KNContextType("user")
	// ProblemKey is the key to be used for adding problems to context
	ProblemKey = KNContextType("problem")
	// ProblemKey is the key to be used for adding blog posts to context
	BlogPostKey = KNContextType("blogPost")
	// SubKey is the key to be used for adding submissions to context
	SubKey = KNContextType("submission")
	// PasteKey is the key to be used for adding pastes to context
	PasteKey = KNContextType("paste")
	// TestKey is the key to be used for adding tests to context
	TestKey = KNContextType("test")
	// SubTaskKey is the key to be used for adding subtasks to context
	SubTaskKey = KNContextType("subtask")
	// ProblemListKey is the key to be used for adding problem lists to context
	ProblemListKey = KNContextType("problemList")
	// AttachmentKey is the key to be used for adding attachments to context
	AttachmentKey = KNContextType("attachment")
	// TagKey is the key to be used for adding tags to context
	TagKey = KNContextType("tag")
	// ContestKey is the key to be used for adding contests to context
	ContestKey = KNContextType("contest")
	// LangKey is the key to be used for adding the user language to context
	LangKey = KNContextType("language")
	// ThemeKey is the key to be used for adding the user's preferred theme to context
	ThemeKey = KNContextType("theme")
)

func UserBriefContext(ctx context.Context) *kilonova.UserBrief {
	b := getValueContext[kilonova.UserFull](ctx, AuthedUserKey).Brief()
	if b == nil {
		b = getValueContext[kilonova.UserBrief](ctx, AuthedUserKey)
	}
	return b
}

func UserBrief(r *http.Request) *kilonova.UserBrief {
	return UserBriefContext(r.Context())
}

func UserFullContext(ctx context.Context) *kilonova.UserFull {
	return getValueContext[kilonova.UserFull](ctx, AuthedUserKey)
}

func UserFull(r *http.Request) *kilonova.UserFull {
	return UserFullContext(r.Context())
}

func ContentUserContext(ctx context.Context) *kilonova.UserFull {
	return getValueContext[kilonova.UserFull](ctx, ContentUserKey)
}

func ContentUser(r *http.Request) *kilonova.UserFull {
	return ContentUserContext(r.Context())
}

func ProblemContext(ctx context.Context) *kilonova.Problem {
	return getValueContext[kilonova.Problem](ctx, ProblemKey)
}

func Problem(r *http.Request) *kilonova.Problem {
	return ProblemContext(r.Context())
}

func BlogPostContext(ctx context.Context) *kilonova.BlogPost {
	return getValueContext[kilonova.BlogPost](ctx, BlogPostKey)
}

func BlogPost(r *http.Request) *kilonova.BlogPost {
	return BlogPostContext(r.Context())
}

func SubmissionContext(ctx context.Context) *kilonova.FullSubmission {
	return getValueContext[kilonova.FullSubmission](ctx, SubKey)
}
func Submission(r *http.Request) *kilonova.FullSubmission {
	return SubmissionContext(r.Context())
}

func ContestContext(ctx context.Context) *kilonova.Contest {
	return getValueContext[kilonova.Contest](ctx, ContestKey)
}

func Contest(r *http.Request) *kilonova.Contest {
	return ContestContext(r.Context())
}

func Paste(r *http.Request) *kilonova.SubmissionPaste {
	return getValueContext[kilonova.SubmissionPaste](r.Context(), PasteKey)
}

func Test(r *http.Request) *kilonova.Test {
	return TestContext(r.Context())
}

func TestContext(ctx context.Context) *kilonova.Test {
	return getValueContext[kilonova.Test](ctx, TestKey)
}

func SubTask(r *http.Request) *kilonova.SubTask {
	return getValueContext[kilonova.SubTask](r.Context(), SubTaskKey)
}

func ProblemList(r *http.Request) *kilonova.ProblemList {
	return ProblemListContext(r.Context())
}

func ProblemListContext(ctx context.Context) *kilonova.ProblemList {
	return getValueContext[kilonova.ProblemList](ctx, ProblemListKey)
}

func Attachment(r *http.Request) *kilonova.Attachment {
	return AttachmentContext(r.Context())
}

func AttachmentContext(ctx context.Context) *kilonova.Attachment {
	return getValueContext[kilonova.Attachment](ctx, AttachmentKey)
}

func Tag(r *http.Request) *kilonova.Tag {
	return getValueContext[kilonova.Tag](r.Context(), TagKey)
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

func Theme(r *http.Request) kilonova.PreferredTheme {
	switch v := r.Context().Value(ThemeKey).(type) {
	case string:
		return kilonova.PreferredTheme(v)
	case *string:
		return kilonova.PreferredTheme(*v)
	case kilonova.PreferredTheme:
		return v
	case *kilonova.PreferredTheme:
		return *v
	default:
		return kilonova.PreferredThemeDark
	}
}

func getValueContext[T any](ctx context.Context, key KNContextType) *T {
	switch v := ctx.Value(key).(type) {
	case *T:
		return v
	case T:
		return &v
	default:
		return nil
	}
}
