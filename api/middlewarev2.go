package api

import (
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *API) SetupSessionV2(ctx huma.Context, next func(huma.Context)) {
	h := ctx.Header("Authorization")
	if h == "guest" {
		h = ""
	}
	r, _ := humachi.Unwrap(ctx)
	user, err := s.base.SessionUser(ctx.Context(), h, r)
	if err != nil || user == nil {
		next(ctx)
		return
	}
	trace.SpanFromContext(ctx.Context()).SetAttributes(attribute.Int("user.id", user.ID), attribute.String("user.name", user.Name))
	// TODO: Scopes
	next(huma.WithValue(ctx, util.AuthedUserKey, user))
}
