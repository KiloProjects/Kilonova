package web

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/views/settingsviews"
	"github.com/google/uuid"
)

func (rt *Web) renderOAuthClients(w http.ResponseWriter, r *http.Request, params settingsviews.OAuthClientsParams) {
	authorID := user.UserBrief(r).ID
	clients, err := rt.base.OAuthClients(r.Context(), &authorID)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not get OAuth clients", slog.Any("err", err))
		rt.statusPage(w, r, 500, "Couldn't fetch applications")
		return
	}
	params.Clients = clients
	params.ShowCreate = true
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "oauth_clients"),
		Content: settingsviews.OAuthClients(params),
	})
}

func (rt *Web) oauthClientsSettings(w http.ResponseWriter, r *http.Request) {
	rt.renderOAuthClients(w, r, settingsviews.OAuthClientsParams{})
}

// parseRedirects splits a textarea into URIs, requiring each to be an absolute URI
func parseRedirects(s string) ([]string, error) {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Native apps may use custom schemes (com.example.app:/callback), so only require a scheme
		if u, err := url.Parse(line); err != nil || u.Scheme == "" || u.Fragment != "" {
			return nil, kilonova.Statusf(400, "Invalid redirect URI: %q", line)
		}
		out = append(out, line)
	}
	return out, nil
}

func (rt *Web) createOAuthClient(w http.ResponseWriter, r *http.Request) {
	fail := func(msg string) {
		w.WriteHeader(400)
		rt.renderOAuthClients(w, r, settingsviews.OAuthClientsParams{Error: msg})
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		fail("Name is required")
		return
	}

	var appType auth.ApplicationType
	switch r.FormValue("app_type") {
	case "web":
		appType = auth.ApplicationTypeWeb
	case "native":
		appType = auth.ApplicationTypeNative
	case "userAgent":
		appType = auth.ApplicationTypeUserAgent
	default:
		fail("Invalid application type")
		return
	}

	redirects, err := parseRedirects(r.FormValue("redirects"))
	if err != nil {
		fail(err.Error())
		return
	}
	if len(redirects) == 0 {
		fail("At least one redirect URI is required")
		return
	}
	postLogout, err := parseRedirects(r.FormValue("post_logout_redirects"))
	if err != nil {
		fail(err.Error())
		return
	}

	id, secret, err := rt.base.CreateClient(r.Context(), name, appType, user.UserBrief(r).ID, r.FormValue("dev_mode") == "true", redirects, postLogout)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not create OAuth client", slog.Any("err", err))
		fail("Couldn't create application")
		return
	}

	rt.renderOAuthClients(w, r, settingsviews.OAuthClientsParams{
		CreatedID:     id.String(),
		CreatedSecret: secret,
	})
}

func (rt *Web) deleteOAuthClient(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("clientID"))
	if err != nil {
		rt.statusPage(w, r, 400, "Invalid client ID")
		return
	}
	client, err := rt.base.GetOAuthClient(r.Context(), id)
	if err != nil {
		rt.statusPage(w, r, 404, "Application not found")
		return
	}
	if !rt.base.IsOAuthClientEditor(user.UserBrief(r), client) {
		rt.statusPage(w, r, 403, "You can't delete this application")
		return
	}
	if err := rt.base.DeleteOAuthClient(r.Context(), id); err != nil {
		slog.WarnContext(r.Context(), "Could not delete OAuth client", slog.Any("err", err))
		rt.statusPage(w, r, 500, "Couldn't delete application")
		return
	}
	back := "/settings/oauth"
	if r.FormValue("back") == "admin" {
		back = "/admin/oauth"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (rt *Web) adminOAuthClients(w http.ResponseWriter, r *http.Request) {
	clients, err := rt.base.OAuthClients(r.Context(), nil)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not get OAuth clients", slog.Any("err", err))
		rt.statusPage(w, r, 500, "Couldn't fetch applications")
		return
	}
	authors := make(map[int]*kilonova.UserBrief)
	for _, c := range clients {
		if c.AuthorID == nil {
			continue
		}
		if _, ok := authors[*c.AuthorID]; !ok {
			// ponytail: one query per distinct author, batch it if the app count grows
			authors[*c.AuthorID], _ = rt.base.UserBrief(r.Context(), *c.AuthorID)
		}
	}
	rt.runLayout(w, r, &LayoutParams{
		Title: kilonova.GetText(util.Language(r), "oauth_clients"),
		Content: settingsviews.OAuthClients(settingsviews.OAuthClientsParams{
			Clients: clients,
			Authors: authors,
		}),
	})
}

func (rt *Web) authorizedApps(w http.ResponseWriter, r *http.Request) {
	clients, err := rt.base.AuthorizedOAuthClients(r.Context(), user.UserBrief(r).ID)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not get authorized apps", slog.Any("err", err))
		rt.statusPage(w, r, 500, "Couldn't fetch applications")
		return
	}
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "authorized_apps"),
		Content: settingsviews.AuthorizedApps(clients),
	})
}

func (rt *Web) revokeAuthorizedApp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("clientID"))
	if err != nil {
		rt.statusPage(w, r, 400, "Invalid client ID")
		return
	}
	if err := rt.base.RevokeOAuthClient(r.Context(), user.UserBrief(r).ID, id); err != nil {
		slog.WarnContext(r.Context(), "Could not revoke app", slog.Any("err", err))
		rt.statusPage(w, r, 500, "Couldn't revoke application")
		return
	}
	http.Redirect(w, r, "/settings/integrations", http.StatusSeeOther)
}
