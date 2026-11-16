package api

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"uuid"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
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
			userFull, scopes, err := s.bearerUser(ctx.Context(), h[7:])
			if err != nil {
				// Clients tell an expired token from a missing permission by this 401, and refresh on it
				slog.DebugContext(ctx.Context(), "Rejected access token", slog.Any("err", err))
				ctx.SetHeader("WWW-Authenticate", `Bearer error="invalid_token"`)
				huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired access token")
				return
			}

			trace.SpanFromContext(ctx.Context()).SetAttributes(attribute.Int("user.id", userFull.ID), attribute.String("user.name", userFull.Name))

			next(huma.WithValue(
				huma.WithValue(huma.WithValue(ctx, util.ScopesKey, scopes),
					user.AuthedUserKey, userFull),
				util.AuthMethodKey, "oauth",
			))
			return
		}

		r, _ := humachi.Unwrap(ctx)
		sessionUser, err := s.base.SessionUser(ctx.Context(), cmp.Or(h, s.cookieSession(r)), r)
		if err != nil || sessionUser == nil {
			next(huma.WithValue(ctx, util.AuthMethodKey, "none"))
			return
		}
		trace.SpanFromContext(ctx.Context()).SetAttributes(attribute.Int("user.id", sessionUser.ID), attribute.String("user.name", sessionUser.Name))
		// TODO: Scopes
		next(huma.WithValue(
			huma.WithValue(ctx, user.AuthedUserKey, sessionUser),
			util.AuthMethodKey, "session",
		))
	}
}

// bearerUser resolves an OAuth access token to its user and granted scopes
func (s *API) bearerUser(ctx context.Context, rawToken string) (*kilonova.UserFull, []string, error) {
	// The signature and expiry are checked here; revocation is the row lookup below
	claims, err := op.VerifyAccessToken[*oidc.AccessTokenClaims](ctx, rawToken, s.base.OIDCProvider().AccessTokenVerifier(ctx))
	if err != nil {
		return nil, nil, err
	}
	userID, err := strconv.Atoi(claims.GetSubject())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid subject: %w", err)
	}
	tokenID, err := uuid.Parse(claims.JWTID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid jwt id: %w", err)
	}
	// Revoking a token deletes its row, so a missing row means revoked
	token, err := s.base.GetAccessToken(ctx, tokenID)
	if err != nil {
		return nil, nil, fmt.Errorf("token not found: %w", err)
	}
	if token.UserID == nil || userID != *token.UserID {
		return nil, nil, errors.New("token subject does not match its user")
	}
	userFull, err := s.base.UserFull(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return userFull, token.Scopes, nil
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

		if !s.base.IsProblemVisible(user.UserBriefContext(ctx.Context()), problem) {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "You are not allowed to access this problem")
			return
		}
		next(huma.WithValue(ctx, util.ProblemKey, problem))
	}
}

func (s *API) validateContestIDv2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		contestID, err := strconv.Atoi(ctx.Param("contestID"))
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "Invalid contest ID")
			return
		}
		contest, err := s.base.Contest(ctx.Context(), contestID)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "Contest does not exist")
			return
		}

		if !s.base.IsContestVisible(user.UserBriefContext(ctx.Context()), contest) {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "You are not allowed to access this contest")
			return
		}

		next(huma.WithValue(ctx, util.ContestKey, contest))
	}
}

// MustBeAuthedV2 is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthedV2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if !user.UserBriefContext(ctx.Context()).IsAuthed() {
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "You must be authenticated to do this")
			return
		}
		next(ctx)
	}
}

// MustBeAdminV2 is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdminV2(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		s.MustBeAuthedV2(api)(ctx, func(ctx huma.Context) {
			if !user.UserBriefContext(ctx.Context()).IsAdmin() {
				huma.WriteErr(api, ctx, http.StatusForbidden, "You must be an admin to do this")
				return
			}
			next(ctx)
		})
	}
}
