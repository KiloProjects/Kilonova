package user

import (
	"context"
	"net/http"
	"net/netip"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/ctxt"
)

type userContextType string

const (
	// AuthedUserKey is the key to be used for adding (AUTHENTICATED) user objects to context
	AuthedUserKey = userContextType("authed")
	// ContentUserKey is the key to be used for adding CONTENT user objects to context
	ContentUserKey = userContextType("user")

	// IPKey is the key to be used for adding the user's IP address to context
	IPKey = userContextType("ip")
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

// IPContext returns the IP from context
// TODO: IPContext and IPKey would probably be better suited for another package
// But for agility reasons, they are kept here for now
func IPContext(ctx context.Context) *netip.Addr {
	return ctxt.Value[netip.Addr, userContextType](ctx, IPKey)
}
