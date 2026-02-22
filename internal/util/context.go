package util

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/ctxt"
)

type knContextType string

const (
	// AuthedUserKey is the key to be used for adding (AUTHENTICATED) user objects to context
	// Deprecated: use [user.AuthedUserKey] instead
	AuthedUserKey = user.AuthedUserKey
	// ContentUserKey is the key to be used for adding CONTENT user objects to context
	// Deprecated: use [user.ContentUserKey] instead
	ContentUserKey = user.ContentUserKey
	// ProblemKey is the key to be used for adding problems to context
	ProblemKey = knContextType("problem")
	// BlogPostKey is the key to be used for adding blog posts to context
	BlogPostKey = knContextType("blogPost")
	// SubKey is the key to be used for adding submissions to context
	SubKey = knContextType("submission")
	// PasteKey is the key to be used for adding pastes to context
	PasteKey = knContextType("paste")
	// TestKey is the key to be used for adding tests to context
	TestKey = knContextType("test")
	// SubTaskKey is the key to be used for adding subtasks to context
	SubTaskKey = knContextType("subtask")
	// ProblemListKey is the key to be used for adding problem lists to context
	ProblemListKey = knContextType("problemList")
	// ExternalResourceKey is the key to be used for adding external resources to context
	ExternalResourceKey = knContextType("externalResource")
	// AttachmentKey is the key to be used for adding attachments to context
	AttachmentKey = knContextType("attachment")
	// TagKey is the key to be used for adding tags to context
	TagKey = knContextType("tag")
	// ContestKey is the key to be used for adding contests to context
	ContestKey = knContextType("contest")
	// LangKey is the key to be used for adding the user language to context
	LangKey = knContextType("language")
	// BucketKey is the key to be used for adding the requested bucket to context
	BucketKey = knContextType("bucket")
	// ThemeKey is the key to be used for adding the user's preferred theme to context
	ThemeKey = knContextType("theme")

	// ScopesKey is the key to be used for adding the scopes to context
	ScopesKey = knContextType("scopes")

	// AuthMethodKey is the key to be used for adding the auth method to context
	AuthMethodKey = knContextType("authMethod")

	// IPKey is the key to be used for adding the IP address to context
	IPKey = knContextType("ip")
)

// Deprecated: use [user.UserBrief] instead
func UserBrief(r *http.Request) *kilonova.UserBrief {
	return user.UserBrief(r)
}

// Deprecated: use [user.UserFullContext] instead
func UserFullContext(ctx context.Context) *kilonova.UserFull {
	return user.UserFullContext(ctx)
}

// Deprecated: use [user.UserFull] instead
func UserFull(r *http.Request) *kilonova.UserFull {
	return user.UserFull(r)
}

// Deprecated: use [user.ContentUserFull] instead
func ContentUserFull(r *http.Request) *kilonova.UserFull {
	return user.ContentUserFull(r)
}

// Deprecated: use [user.ContentUserBriefContext] instead
func ContentUserBriefContext(ctx context.Context) *kilonova.UserBrief {
	return user.ContentUserBriefContext(ctx)
}

// Deprecated: use [user.ContentUserBrief] instead
func ContentUserBrief(r *http.Request) *kilonova.UserBrief {
	return user.ContentUserBrief(r)
}

func ProblemContext(ctx context.Context) *kilonova.Problem {
	return ctxt.Value[kilonova.Problem, knContextType](ctx, ProblemKey)
}

func Problem(r *http.Request) *kilonova.Problem {
	return ProblemContext(r.Context())
}

func BlogPostContext(ctx context.Context) *kilonova.BlogPost {
	return ctxt.Value[kilonova.BlogPost, knContextType](ctx, BlogPostKey)
}

func BlogPost(r *http.Request) *kilonova.BlogPost {
	return BlogPostContext(r.Context())
}

func SubmissionContext(ctx context.Context) *kilonova.FullSubmission {
	return ctxt.Value[kilonova.FullSubmission, knContextType](ctx, SubKey)
}
func Submission(r *http.Request) *kilonova.FullSubmission {
	return SubmissionContext(r.Context())
}

func ContestContext(ctx context.Context) *kilonova.Contest {
	return ctxt.Value[kilonova.Contest, knContextType](ctx, ContestKey)
}

func Contest(r *http.Request) *kilonova.Contest {
	return ContestContext(r.Context())
}

func Paste(r *http.Request) *kilonova.SubmissionPaste {
	return ctxt.Value[kilonova.SubmissionPaste, knContextType](r.Context(), PasteKey)
}

func ExternalResource(r *http.Request) *kilonova.ExternalResource {
	return ctxt.Value[kilonova.ExternalResource, knContextType](r.Context(), ExternalResourceKey)
}

func Test(r *http.Request) *kilonova.Test {
	return TestContext(r.Context())
}

func TestContext(ctx context.Context) *kilonova.Test {
	return ctxt.Value[kilonova.Test, knContextType](ctx, TestKey)
}

func SubTask(r *http.Request) *kilonova.SubTask {
	return ctxt.Value[kilonova.SubTask, knContextType](r.Context(), SubTaskKey)
}

func ProblemList(r *http.Request) *kilonova.ProblemList {
	return ProblemListContext(r.Context())
}

func ProblemListContext(ctx context.Context) *kilonova.ProblemList {
	return ctxt.Value[kilonova.ProblemList, knContextType](ctx, ProblemListKey)
}

func Attachment(r *http.Request) *kilonova.Attachment {
	return AttachmentContext(r.Context())
}

func AttachmentContext(ctx context.Context) *kilonova.Attachment {
	return ctxt.Value[kilonova.Attachment, knContextType](ctx, AttachmentKey)
}

func BucketContext(ctx context.Context) datastore.Bucket {
	switch v := ctx.Value(BucketKey).(type) {
	case *datastore.Bucket:
		return *v
	case datastore.Bucket:
		return v
	default:
		return nil
	}
}

func Tag(r *http.Request) *kilonova.Tag {
	return ctxt.Value[kilonova.Tag, knContextType](r.Context(), TagKey)
}

func Language(r *http.Request) string {
	return LanguageContext(r.Context())
}

func IPContext(ctx context.Context) *netip.Addr {
	return ctxt.Value[netip.Addr, knContextType](ctx, IPKey)
}

func LanguageContext(ctx context.Context) string {
	switch v := ctx.Value(LangKey).(type) {
	case string:
		return v
	case *string:
		return *v
	default:
		return config.Common.DefaultLang
	}
}

func Theme(ctx context.Context) kilonova.PreferredTheme {
	switch v := ctx.Value(ThemeKey).(type) {
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
