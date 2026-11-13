// Package authviews renders the login, logout and OAuth consent pages.
package authviews

import (
	"context"
	"html"

	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func LoginPage(ctx context.Context, oidcID, back, errorMessage string) Node {
	return ui.CenteredLayout(
		Form(Class("segment-panel"), ID("login_form"), Method("POST"),
			H1(Class("mb-4 text-center"), ui.T(ctx, "authenticate")),
			If(errorMessage != "", P(Class("text-red-500"), Text(errorMessage))),
			If(oidcID != "", Input(Type("hidden"), Name("oidcID"), Value(oidcID))),
			If(back != "", Input(Type("hidden"), Name("back"), Value(back))),
			Input(Type("hidden"), Name("form_type"), Value("login")),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "username_email")), Text(" "),
				Input(Class("form-input w-full"), Type("text"), ID("login_uname"), Name("username")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "password")), Text(" "),
				Input(Class("form-input w-full"), Type("password"), ID("login_upwd"), Name("password")),
			),
			ui.SubmitButton(Class("block mb-3"), ui.T(ctx, "auth.login")),
			If(flags.SignupEnabled.Get(ctx), P(Class("text-gray-600 dark:text-gray-300"), ui.TRaw(ctx, "signupReminder"))),
			P(Class("text-gray-600 dark:text-gray-300"), ui.TRaw(ctx, "forgot_pwd_announcement")),
		),
	)
}

func OAuthGrant(ctx context.Context, authRequest *auth.Request, client *auth.Client) Node {
	return ui.CenteredLayout(
		Form(Class("segment-panel"), ID("login_form"), Method("POST"),
			H1(Class("mb-4 text-center"), ui.T(ctx, "authorize_application")),
			P(ui.TRaw(ctx, "authorize_application.description", html.EscapeString(client.Name))),
			P(Class("reset-list"),
				Strong(ui.T(ctx, "authorize_application.scopes")),
				Ul(Map(authRequest.Scopes, func(scope string) Node { return Li(Text(scope)) })),
			),
			Input(Type("hidden"), Name("form_type"), Value("oauth_grant")),
			Input(Type("hidden"), Name("authRequestID"), Value(authRequest.ID.String())),
			ui.SubmitButton(ui.T(ctx, "button.authorize")),
			Text(" "),
			A(Href("/"), Class("btn"), ui.T(ctx, "button.cancel")),
		),
	)
}

func ConfirmLogout(ctx context.Context, back string) Node {
	return ui.CenteredLayout(
		Form(Class("segment-panel"), Method("POST"), Action("/logout"),
			H1(Class("mb-4"), ui.T(ctx, "auth.confirm_logout")),
			Input(Type("hidden"), Name("back"), Value(back)),
			ui.SubmitButton(ui.T(ctx, "auth.logout")),
		),
	)
}
