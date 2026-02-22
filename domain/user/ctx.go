package user

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/ctxt"
)

type userContextType string

const (
	// AuthedUserKey is the key to be used for adding (AUTHENTICATED) user objects to context
	AuthedUserKey = userContextType("authed")
	// ContentUserKey is the key to be used for adding CONTENT user objects to context
	ContentUserKey = userContextType("user")
)

func userBrief(ctx context.Context, key userContextType) *kilonova.UserBrief {
	b := ctxt.Value[kilonova.UserFull, userContextType](ctx, key).Brief()
	if b == nil {
		b = ctxt.Value[kilonova.UserBrief, userContextType](ctx, key)
	}
	return b
}

func UserBriefContext(ctx context.Context) *kilonova.UserBrief {
	return userBrief(ctx, AuthedUserKey)
}

func UserBrief(r *http.Request) *kilonova.UserBrief {
	return UserBriefContext(r.Context())
}

func UserFullContext(ctx context.Context) *kilonova.UserFull {
	return ctxt.Value[kilonova.UserFull, userContextType](ctx, AuthedUserKey)
}

func UserFull(r *http.Request) *kilonova.UserFull {
	return UserFullContext(r.Context())
}

func ContentUserFullContext(ctx context.Context) *kilonova.UserFull {
	return ctxt.Value[kilonova.UserFull, userContextType](ctx, ContentUserKey)
}

func ContentUserFull(r *http.Request) *kilonova.UserFull {
	return ContentUserFullContext(r.Context())
}

func ContentUserBriefContext(ctx context.Context) *kilonova.UserBrief {
	return userBrief(ctx, ContentUserKey)
}

func ContentUserBrief(r *http.Request) *kilonova.UserBrief {
	return ContentUserBriefContext(r.Context())
}
