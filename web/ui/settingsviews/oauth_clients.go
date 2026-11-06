// Package settingsviews renders OAuth client management and authorized-app pages.
package settingsviews

import (
	"context"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type OAuthClientsParams struct {
	Clients []*auth.Client
	// Authors is only set on the admin overview, where the author column is shown
	Authors map[int]*kilonova.UserBrief
	// ShowCreate enables the creation form (settings page)
	ShowCreate bool

	// Set right after creation, the secret is shown only once
	CreatedID     string
	CreatedSecret string
	Error         string
}

func appTypeName(t auth.ApplicationType) string {
	switch t {
	case auth.ApplicationTypeWeb:
		return "Web"
	case auth.ApplicationTypeNative:
		return "Native"
	case auth.ApplicationTypeUserAgent:
		return "User Agent (SPA)"
	default:
		return "?"
	}
}

func OAuthClients(ctx context.Context, params OAuthClientsParams) Node {
	admin := params.Authors != nil
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "oauth_clients")),
		If(params.Error != "", Div(Class("segment-panel text-red-600"), Text(params.Error))),
		If(params.CreatedSecret != "", Div(Class("segment-panel"),
			H2(ui.T(ctx, "oauth_clients.created")),
			P(ui.T(ctx, "oauth_clients.secret_warning")),
			P(Text("Client ID: "), Code(Text(params.CreatedID))),
			P(Text("Client Secret: "), Code(Text(params.CreatedSecret))),
		)),
		If(params.ShowCreate, createForm(ctx)),
		If(len(params.Clients) > 0, Table(Class("kn-table"),
			THead(Tr(
				th(ui.T(ctx, "name")),
				th(Text("Client ID")),
				th(ui.T(ctx, "oauth_clients.app_type")),
				th(ui.T(ctx, "oauth_clients.redirects")),
				If(admin, th(ui.T(ctx, "author"))),
				th(ui.T(ctx, "created_at")),
				th(),
			)),
			TBody(Map(params.Clients, func(c *auth.Client) Node {
				return Tr(Class("kn-table-row"),
					td(Text(c.Name), If(c.DeveloperMode, Group{Text(" "), Span(Class("badge-lite text-sm"), Text("dev"))})),
					td(Code(Text(c.ID.String()))),
					td(Text(appTypeName(c.AppType))),
					td(Text(strings.Join(c.AllowedRedirects, ", "))),
					If(admin, td(author(c, params.Authors))),
					td(ui.ServerTimestamp(c.CreatedAt)),
					td(confirmForm(fmt.Sprintf("/settings/oauth/%s/delete", c.ID), ui.TS(ctx, "oauth_clients.delete_confirm"),
						If(admin, Input(Type("hidden"), Name("back"), Value("admin"))),
						ui.DangerSubmitButton(ui.T(ctx, "button.delete")),
					)),
				)
			})),
		)),
		If(len(params.Clients) == 0, Div(Class("text-2xl text-center my-6"), ui.T(ctx, "oauth_clients.none"))),
	)
}

func createForm(ctx context.Context) Node {
	field := func(key string, input Node) Node {
		return Label(Class("block my-2"), Span(Class("form-label"), ui.T(ctx, key), Text(":")), Text(" "), input)
	}
	return Form(Class("segment-panel"), Method("POST"), Action("/settings/oauth"), AutoComplete("off"),
		H2(ui.T(ctx, "oauth_clients.create")),
		field("name", Input(Class("form-input block w-full"), Type("text"), Name("name"), Required())),
		field("oauth_clients.app_type", Select(Class("form-select block"), Name("app_type"),
			Option(Value("web"), Selected(), Text(appTypeName(auth.ApplicationTypeWeb))),
			Option(Value("native"), Text(appTypeName(auth.ApplicationTypeNative))),
			Option(Value("userAgent"), Text(appTypeName(auth.ApplicationTypeUserAgent))),
		)),
		field("oauth_clients.redirects", Textarea(Class("form-textarea block w-full"), Name("redirects"), Rows("3"), Required(), Placeholder("https://example.com/callback"))),
		field("oauth_clients.post_logout_redirects", Textarea(Class("form-textarea block w-full"), Name("post_logout_redirects"), Rows("2"))),
		Label(Class("inline-flex items-center my-2"),
			Input(Class("form-checkbox"), Type("checkbox"), Name("dev_mode"), Value("true")), Text(" "),
			Span(Class("ml-2"), ui.T(ctx, "oauth_clients.dev_mode")),
		),
		Div(ui.SubmitButton(ui.T(ctx, "button.create"))),
	)
}

func author(c *auth.Client, authors map[int]*kilonova.UserBrief) Node {
	if c.AuthorID == nil || authors[*c.AuthorID] == nil {
		return Text("-")
	}
	name := authors[*c.AuthorID].Name
	return A(Href("/profile/"+name), Text(name))
}

func AuthorizedApps(ctx context.Context, clients []*auth.Client) Node {
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "authorized_apps")),
		If(len(clients) > 0, Table(Class("kn-table"),
			THead(Tr(th(ui.T(ctx, "name")), th())),
			TBody(Map(clients, func(c *auth.Client) Node {
				return Tr(Class("kn-table-row"),
					td(Text(c.Name)),
					td(confirmForm(fmt.Sprintf("/settings/integrations/%s/revoke", c.ID), ui.TS(ctx, "authorized_apps.revoke_confirm"),
						ui.DangerSubmitButton(ui.T(ctx, "button.revoke")),
					)),
				)
			})),
		)),
		If(len(clients) == 0, Div(Class("text-2xl text-center my-6"), ui.T(ctx, "authorized_apps.none"))),
	)
}

// confirmForm POSTs to action after the browser confirms message.
func confirmForm(action, message string, children ...Node) Node {
	return Form(Method("POST"), Action(action), Data("confirm", message), Attr("onsubmit", "return confirm(this.dataset.confirm)"), Group(children))
}

func th(children ...Node) Node { return Th(Class("kn-table-cell"), Scope("col"), Group(children)) }
func td(children ...Node) Node { return Td(Class("kn-table-cell"), Group(children)) }
