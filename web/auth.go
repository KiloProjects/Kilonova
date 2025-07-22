package web

import (
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/views/authviews"
	"github.com/KiloProjects/kilonova/web/views/utilviews"
	"github.com/zitadel/oidc/v3/pkg/op"
)

func (rt *Web) getLogin(w http.ResponseWriter, r *http.Request) {
	back := rt.hostURL.JoinPath(r.URL.Query().Get("back")).String()
	oidcID := r.URL.Query().Get("id")
	if util.UserBrief(r) == nil {
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "auth.login"),
			Head:    utilviews.CanonicalURL("/login"),
			Content: authviews.LoginPage(oidcID, back),
		})
		return
	}

	if oidcID == "" {
		// authed, no openid flow, just redirect back
		http.Redirect(w, r, back, http.StatusFound)
		return
	}

	request, err := rt.base.GetAuthRequest(r.Context(), oidcID)
	if err != nil {
		rt.statusPage(w, r, http.StatusInternalServerError, "Invalid auth request")
		return
	}

	client, err := rt.base.GetOAuthClient(r.Context(), request.ApplicationID)
	if err != nil {
		rt.statusPage(w, r, http.StatusInternalServerError, "Invalid auth request")
		return
	}

	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.oauth_grant"),
		Head:    utilviews.CanonicalURL("/login"),
		Content: authviews.OAuthGrant(request, client),
	})

}

func (rt *Web) postLogin(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("authRequestID")

	if err := rt.base.ApproveAuthRequest(r.Context(), id, util.UserBrief(r).ID); err != nil {
		slog.ErrorContext(r.Context(), "Failed to approve auth request", slog.Any("error", err))
		rt.statusPage(w, r, http.StatusInternalServerError, "Invalid auth request")
		return
	}

	http.Redirect(w, r, op.AuthCallbackURL(rt.base.OIDCProvider())(r.Context(), id), http.StatusFound)
}
