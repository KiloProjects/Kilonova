package authviews

import (
	"cmp"
	"context"
	"net/url"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SignupParams struct {
	// Enabled is false when sign ups are turned off on the platform.
	Enabled bool
	// CaptchaID is empty when no CAPTCHA has to be solved.
	CaptchaID string
	// Email, Username and Language refill the form after a failed attempt.
	Email, Username, Language string
	// Back is the path to return to after signing up.
	Back         string
	ErrorMessage string
}

func SignupPage(ctx context.Context, params SignupParams) Node {
	if !params.Enabled {
		return H1(ui.T(ctx, "disabled_signup"))
	}
	lang := cmp.Or(params.Language, util.LanguageContext(ctx))
	field := func(key string, input Node) Node {
		return Label(Class("block mb-2"), Span(ui.T(ctx, key)), Text(" "), input)
	}
	return ui.CenteredLayout(
		Form(Class("segment-panel"), ID("signup_form"), Method("POST"), Action("/signup"),
			H1(Class("mb-4 text-center"), ui.T(ctx, "auth.signup")),
			If(params.ErrorMessage != "", P(Class("text-red-500"), Text(params.ErrorMessage))),
			If(params.Back != "", Input(Type("hidden"), Name("back"), Value(params.Back))),
			field("email", Input(Class("form-input w-full"), Type("email"), ID("email"), Name("email"), Value(params.Email), AutoComplete("email"), Required())),
			field("username", Input(Class("form-input w-full"), Type("text"), ID("name"), Name("username"), Value(params.Username), MinLength("3"), MaxLength("24"), AutoComplete("username"), Required())),
			field("password", Input(Class("form-input w-full"), Type("password"), ID("pwd"), Name("password"), MinLength("6"), MaxLength("72"), AutoComplete("new-password"), Required())),
			field("passwordCheck", Input(Class("form-input w-full"), Type("password"), ID("pwd_check"), Name("password_confirm"), MinLength("6"), MaxLength("72"), AutoComplete("new-password"), Required())),
			field("displayLang", Select(ID("langSelect"), Name("language"), Class("form-select w-full"),
				Option(Value("ro"), If(lang == "ro", Selected()), Text("🇷🇴 Română")),
				Option(Value("en"), If(lang == "en", Selected()), Text("🇬🇧 English")),
			)),
			Iff(params.CaptchaID != "", func() Node { return captcha(ctx, params.CaptchaID) }),
			ui.SubmitButton(Class("block mb-3"), ui.T(ctx, "auth.signup")),
			P(Class("text-gray-500 dark:text-gray-300"), Text(" "), ui.TRaw(ctx, "loginReminder")),
		),
		Script(Raw(signupJS)),
	)
}

// signupJS colours the password fields as they are typed; the server checks the match either way.
const signupJS = `
{
	const pwd = document.querySelector("#signup_form #pwd")
	const pwdCheck = document.querySelector("#signup_form #pwd_check")
	function updateCheck() {
		const filled = pwd.value != "" && pwdCheck.value != ""
		const same = pwd.value === pwdCheck.value
		for (const el of [pwd, pwdCheck]) {
			el.classList.toggle("is-valid", filled && same)
			el.classList.toggle("is-invalid", filled && !same)
		}
	}
	pwd.addEventListener("focusout", updateCheck)
	pwdCheck.addEventListener("keyup", updateCheck)
}
`

// ForgotPasswordPage is the form asking for a password reset email. sent replaces the form with the confirmation.
func ForgotPasswordPage(ctx context.Context, captchaID, email, errorMessage string, sent bool) Node {
	if sent {
		return ui.CenteredLayout(Div(Class("segment-panel"),
			H1(Class("mb-4 text-center"), ui.T(ctx, "auth.forgot_pwd")),
			P(ui.T(ctx, "auth.forgot_pwd.sent")),
		))
	}
	return ui.CenteredLayout(
		Form(Class("segment-panel"), ID("pwd_forgot_form"), Method("POST"), Action("/forgot_pwd"),
			H1(Class("mb-4 text-center"), ui.T(ctx, "auth.forgot_pwd")),
			If(errorMessage != "", P(Class("text-red-500"), Text(errorMessage))),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "email")), Text(" "),
				Input(Class("form-input w-full"), Type("email"), ID("pwd_email"), Name("email"), Value(email), Required()),
			),
			Iff(captchaID != "", func() Node { return captcha(ctx, captchaID) }),
			ui.SubmitButton(Class("block"), ui.T(ctx, "recover")),
		),
	)
}

func ResetPasswordPage(ctx context.Context, u *kilonova.UserFull, requestID, errorMessage string) Node {
	pwd := func(key, name string) Node {
		return Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, key)), Text(" "),
			Input(Class("form-input w-full"), Type("password"), AutoComplete("new-password"), Name(name), MinLength("6"), MaxLength("72"), Required()),
		)
	}
	return ui.CenteredLayout(
		Form(Class("segment-panel"), ID("pwd_reset_form"), Method("POST"), Action("/resetPassword/"+url.PathEscape(requestID)),
			H1(Class("mb-4 text-center"), ui.T(ctx, "resetTitle", u.Name)),
			If(errorMessage != "", P(Class("text-red-500"), Text(errorMessage))),
			// Lets password managers match the new password to the account
			Input(Type("email"), Class("hidden"), AutoComplete("username"), Value(u.Email)),
			pwd("newPwd", "password"),
			pwd("newPwdConfirm", "password_confirm"),
			ui.SubmitButton(Class("block"), ui.T(ctx, "reset")),
		),
	)
}

func PasswordResetDone(ctx context.Context) Node {
	return ui.CenteredLayout(Div(Class("segment-panel"),
		H1(Class("mb-4 text-center"), ui.T(ctx, "reset_pwd")),
		P(Class("mb-2"), ui.T(ctx, "auth.reset_pwd.done")),
		A(Href("/login"), Class("btn btn-blue"), ui.T(ctx, "auth.login")),
	))
}

func EmailResent(ctx context.Context) Node {
	return Group{
		H1(Text(" "), ui.T(ctx, "email.resent"), Text(" ")),
		A(Href("/"), ui.T(ctx, "frontPage")),
	}
}

func EmailVerified(ctx context.Context, name string) Node {
	return Group{
		H1(ui.T(ctx, "confirmedEmail", name)),
		H3(ui.T(ctx, "closeTab")),
	}
}

// captcha renders the CAPTCHA box; the script only reloads the image.
func captcha(ctx context.Context, id string) Node {
	return Group{
		Div(ID("captcha_container"), Class("segment-panel"),
			P(
				ui.T(ctx, "auth.captcha.title"), Text(":"), Br(),
				Img(ID("captcha_img"), Class("rounded-sm"), Src("/api/auth/captcha/"+id+".png"), Alt("Captcha image"), Width("240"), Height("80"), Style("background-color: white;")),
			),
			Input(Type("hidden"), ID("captcha_id"), Name("captcha_id"), Value(id)),
			Label(Class("block my-2"),
				Span(ui.T(ctx, "response")), Text(" "),
				Input(Type("text"), ID("captcha_response"), Name("captcha_response"), Class("form-input w-full"), Pattern("[0-9]*"), Value(""), AutoComplete("off")),
			),
			ui.Button(ID("captcha_reload_btn"), I(Class("fas fa-fw fa-rotate")), Text(" "), ui.T(ctx, "reload")),
		),
		Script(Raw(captchaJS)),
	}
}

const captchaJS = `
document.getElementById("captcha_reload_btn").addEventListener("click", () => {
	document.getElementById("captcha_img").src = "/api/auth/captcha/" + document.getElementById("captcha_id").value + ".png?reload=" + Date.now()
})
`
