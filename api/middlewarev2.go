package api

import (
	"cmp"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/google/uuid"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *API) SetupSessionV2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		h := ctx.Header("Authorization")
		if h == "guest" {
			h = ""
		}
		h = cmp.Or(h, ctx.Header("X-Api-Key"))

		// Handle OAuth token
		if strings.HasPrefix(h, "Bearer ") {
			h = h[7:]
			// TODO: It's not good, it does not handle revoked tokens
			claims, err := op.VerifyAccessToken[*oidc.AccessTokenClaims](ctx.Context(), h, s.base.OIDCProvider().AccessTokenVerifier(ctx.Context()))
			if err != nil {
				next(ctx)
				return
			}
			userID, err := strconv.Atoi(claims.GetSubject())
			if err != nil {
				slog.ErrorContext(ctx.Context(), "Received access token with invalid subject", slog.Any("err", err))
				next(ctx)
				return
			}

			tokenID, err := uuid.Parse(claims.JWTID)
			if err != nil {
				slog.ErrorContext(ctx.Context(), "Received access token with invalid jwt id", slog.Any("err", err))
				next(ctx)
				return
			}

			token, err := s.base.GetAccessToken(ctx.Context(), tokenID)
			if err != nil {
				// If err is nil, the token is probably revoked
				next(ctx)
				return
			}
			if token.UserID == nil || userID != *token.UserID {
				slog.ErrorContext(ctx.Context(), "Received access token with invalid user id", slog.Int("user_id", userID), slog.Int("token_user_id", *token.UserID))
				next(ctx)
				return
			}

			user, err := s.base.UserFull(ctx.Context(), userID)
			if err != nil {
				next(ctx)
				return
			}

			trace.SpanFromContext(ctx.Context()).SetAttributes(attribute.Int("user.id", user.ID), attribute.String("user.name", user.Name))

			next(huma.WithValue(
				huma.WithValue(huma.WithValue(ctx, util.ScopesKey, token.Scopes),
					util.AuthedUserKey, user),
				util.AuthMethodKey, "oauth",
			))
			return
		}

		r, _ := humachi.Unwrap(ctx)
		user, err := s.base.SessionUser(ctx.Context(), h, r)
		if err != nil || user == nil {
			next(huma.WithValue(ctx, util.AuthMethodKey, "none"))
			return
		}
		trace.SpanFromContext(ctx.Context()).SetAttributes(attribute.Int("user.id", user.ID), attribute.String("user.name", user.Name))
		// TODO: Scopes
		next(huma.WithValue(
			huma.WithValue(ctx, util.AuthedUserKey, user),
			util.AuthMethodKey, "session",
		))
	}
}

func (s *API) CheckScopes(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authMethod, ok := ctx.Context().Value(util.AuthMethodKey).(string)
		if !ok || authMethod == "" {
			huma.WriteErr(api, ctx, http.StatusInternalServerError, "Unknown auth method")
			return
		}

		if authMethod == "oauth" {
			var anyOfNeededScopes []string
			authorizationRequired := false
			for _, opScheme := range ctx.Operation().Security {
				var ok bool
				if anyOfNeededScopes, ok = opScheme[authMethod]; ok {
					authorizationRequired = true
					break
				}
			}

			if !authorizationRequired {
				next(ctx)
				return
			}

			scopes, ok := ctx.Context().Value(util.ScopesKey).([]string)
			if !ok || len(scopes) == 0 {
				huma.WriteErr(api, ctx, http.StatusForbidden, "Missing scopes")
				return
			}

			if !slices.Contains(scopes, "api") {
				huma.WriteErr(api, ctx, http.StatusForbidden, "Missing `api` scope")
				return
			}

			for _, scope := range anyOfNeededScopes {
				if slices.Contains(scopes, scope) {
					next(ctx)
					return
				}
			}
			huma.WriteErr(api, ctx, http.StatusForbidden, "Missing required scope")
		} else {
			next(ctx)
		}
	}
}

// validateProblemID pre-emptively returns if there isn't a valid problem ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) validateProblemIDv2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		problemID, err := strconv.Atoi(ctx.Param("problemID"))
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "Invalid problem ID", err)
			return
		}

		problem, err := s.base.Problem(ctx.Context(), problemID)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "Problem does not exist")
			return
		}

		if !s.base.IsProblemVisible(util.UserBriefContext(ctx.Context()), problem) {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "You are not allowed to access this problem")
			return
		}
		next(huma.WithValue(ctx, util.ProblemKey, problem))
	}
}

// MustBeAuthedV2 is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthedV2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if !util.UserBriefContext(ctx.Context()).IsAuthed() {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "You must be authenticated to do this")
			return
		}
		next(ctx)
	}
}
