package authviews

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type SignupParams struct {
	// Enabled is false when sign ups are turned off on the platform.
	Enabled bool
	// CaptchaID is empty when CAPTCHAs are disabled.
	CaptchaID string
	// ShowCaptcha shows the CAPTCHA up front instead of after the first failed attempt.
	ShowCaptcha bool
}

func SignupPage(ctx context.Context, params SignupParams) Node {
	if !params.Enabled {
		return H1(ui.T(ctx, "disabled_signup"))
	}
	lang := util.LanguageContext(ctx)
	field := func(key string, input Node) Node {
		return Label(Class("block mb-2"), Span(ui.T(ctx, key)), Text(" "), input)
	}
	return Group{
		ui.CenteredLayout(
			Form(Class("segment-panel"), ID("signup_form"),
				H1(Class("mb-4 text-center"), ui.T(ctx, "auth.signup")),
				field("email", Input(Class("form-input w-full"), Type("email"), ID("email"), Name("email"), AutoComplete("email"), Required())),
				field("username", Input(Class("form-input w-full"), Type("text"), ID("name"), Name("username"), MinLength("3"), MaxLength("24"), AutoComplete("username"), Required())),
				field("password", Input(Class("form-input w-full"), Type("password"), ID("pwd"), Name("password"), MinLength("6"), MaxLength("72"), AutoComplete("new-password"), Required())),
				field("passwordCheck", Input(Class("form-input w-full"), Type("password"), ID("pwd_check"), Name("pwd_check"), MinLength("6"), MaxLength("72"), AutoComplete("new-password"), Required())),
				field("displayLang", Select(ID("langSelect"), Name("langSelect"), Class("form-select w-full"),
					Option(Value("ro"), If(lang == "ro", Selected()), Text("🇷🇴 Română")),
					Option(Value("en"), If(lang == "en", Selected()), Text("🇬🇧 English")),
				)),
				captcha(ctx, params.CaptchaID, !params.ShowCaptcha),
				ui.SubmitButton(Class("block mb-3"), ui.T(ctx, "auth.signup")),
				P(Class("text-gray-500 dark:text-gray-300"), Text(" "), ui.TRaw(ctx, "loginReminder")),
			),
		),
		Script(Raw(signupJS)),
	}
}

func ForgotPasswordPage(ctx context.Context, captchaID string) Node {
	return Group{
		ui.CenteredLayout(
			Form(Class("segment-panel"), ID("pwd_forgot_form"),
				H1(Class("mb-4 text-center"), ui.T(ctx, "auth.forgot_pwd")),
				Label(Class("block mb-2"),
					Span(Class("form-label"), ui.T(ctx, "email")), Text(" "),
					Input(Class("form-input w-full"), Type("email"), ID("pwd_email"), Name("email"), Required()),
				),
				captcha(ctx, captchaID, false),
				ui.SubmitButton(Class("block"), ui.T(ctx, "recover")),
			),
		),
		Script(Raw(forgotPwdJS)),
	}
}

func ResetPasswordPage(ctx context.Context, u *kilonova.UserFull, requestID string) Node {
	pwd := func(key, id string) Node {
		return Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, key)), Text(" "),
			Input(Class("form-input w-full"), Type("password"), ID(id), AutoComplete("off"), Name("password"), Required()),
		)
	}
	return Group{
		ui.CenteredLayout(
			Form(Class("segment-panel"), ID("pwd_reset_form"), AutoComplete("off"),
				H1(Class("mb-4 text-center"), ui.T(ctx, "resetTitle", u.Name)),
				Input(Type("hidden"), ID("pwd_reset_reqid"), Name("req_id"), Value(requestID)),
				Input(Type("email"), Class("hidden"), Value(u.Email)),
				pwd("newPwd", "pwd_reset_pwd"),
				pwd("newPwdConfirm", "pwd_reset_pwd1"),
				ui.SubmitButton(Class("block"), ui.T(ctx, "reset")),
			),
		),
		Script(Raw(resetPwdJS)),
	}
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

// captcha renders the CAPTCHA box, or a stub defining the same JS functions when id is empty.
func captcha(ctx context.Context, id string, hidden bool) Node {
	if id == "" {
		return Script(Raw(noCaptchaJS))
	}
	return Group{
		Div(ID("captcha_container"), Class("segment-panel"), If(hidden, Style("display: none;")),
			P(
				ui.T(ctx, "auth.captcha.title"), Text(":"), Br(),
				Img(ID("captcha_img"), Class("rounded-sm"), Src("/api/auth/captcha/"+id+".png"), Alt("Captcha image"), Width("240"), Height("80"), Style("background-color: white;")),
			),
			Input(Type("hidden"), ID("captcha_id"), Value(id)),
			Label(Class("block my-2"),
				Span(ui.T(ctx, "response")), Text(" "),
				Input(Type("text"), ID("captcha_response"), Class("form-input w-full"), Pattern("[0-9]*"), Value(""), AutoComplete("off")),
			),
			ui.Button(ID("captcha_reload_btn"), I(Class("fas fa-fw fa-rotate")), Text(" "), ui.T(ctx, "reload")),
		),
		Script(Raw(captchaJS)),
	}
}

const captchaJS = `
function reloadCaptcha() {
	document.getElementById("captcha_img").src = ` + "`/api/auth/captcha/${document.getElementById(\"captcha_id\").value}.png?reload=${bundled.dayjs().valueOf()}`" + `
}
document.getElementById("captcha_reload_btn").addEventListener("click", reloadCaptcha)
function resetCaptcha(newID) {
	document.getElementById("captcha_container").style.display = "block"
	document.getElementById("captcha_id").value = newID
	document.getElementById("captcha_response").value = ""
	reloadCaptcha()
}
function getCaptchaData() {
	return {captcha_id: document.getElementById("captcha_id").value, captcha_response: document.getElementById("captcha_response").value}
}
`

const noCaptchaJS = `
function resetCaptcha() {
	// CAPTCHA module not loaded in, prompt a page reload
	window.location.reload();
}
function getCaptchaData() {
	return {captcha_id: "", captcha_response: ""}
}
`

// handleCaptchaErrorJS is shared by the forms that may answer with a CAPTCHA challenge.
const handleCaptchaErrorJS = `
	if(res.status == "error") {
		if(typeof res.data === "object") {
			resetCaptcha(res.data.captcha_id)
			if(res.data.hasOwnProperty("translation_key")) {
				bundled.apiToast({"status": "error", "data": bundled.getText(res.data.translation_key)})
			} else {
				bundled.apiToast({"status": "error", "data": res.data.text})
			}
		} else {
			bundled.apiToast(res)
		}
		return
	}
`

const signupJS = `
let pwd = document.querySelector("#signup_form #pwd")
let pwd_check = document.querySelector("#signup_form #pwd_check")
function updateCheck() {
	let pval = pwd.value;
	let cval = pwd_check.value;
	if(cval == "" || pval == "") {
		pwd_check.classList.remove("is-valid", "is-invalid")
	} else {
		pwd_check.classList.toggle("is-valid", pval === cval)
		pwd_check.classList.toggle("is-invalid", pval !== cval)
		pwd.classList.toggle("is-valid", pval === cval)
		pwd.classList.toggle("is-invalid", pval !== cval)
	}
}
pwd.addEventListener("focusout", updateCheck)
pwd_check.addEventListener("keyup", updateCheck)

async function signup(e) {
	e.preventDefault()
	let data = {
		email: document.querySelector("#signup_form #email").value,
		username: document.querySelector("#signup_form #name").value,
		password: pwd.value,
		language: document.querySelector("#signup_form #langSelect").value,
		...getCaptchaData()
	}

	if(pwd.value !== pwd_check.value) {
		bundled.createToast({
			status: "error",
			title: bundled.getText("differentPwds")
		})
		return
	}

	let res = await bundled.postCall("/auth/signup", data)
` + handleCaptchaErrorJS + `
	bundled.clearLanguageCookie()
	bundled.setSession(res.data)
	bundled.navigateBack()
}
document.getElementById("signup_form").addEventListener("submit", signup)
`

const forgotPwdJS = `
document.getElementById("pwd_forgot_form").addEventListener("submit", sendRequest)

async function sendRequest(e) {
	e.preventDefault();
	let email = document.getElementById("pwd_email").value;
	let res = await bundled.postCall("/auth/forgotPassword", {
		email,
		...getCaptchaData(),
	})
` + handleCaptchaErrorJS + `
	bundled.apiToast(res)
}
`

const resetPwdJS = `
async function resetPassword(e) {
	e.preventDefault()
	let request_id = document.getElementById("pwd_reset_reqid").value;
	let pwd = document.getElementById("pwd_reset_pwd").value,
		pwd1= document.getElementById("pwd_reset_pwd1").value;
	if(pwd != pwd1) {
		bundled.createToast({title: bundled.getText("differentPwds"), status: 'error'});
		return
	}
	let res = await bundled.postCall("/auth/resetPassword", {req_id: request_id, password: pwd})
	bundled.apiToast(res)
}

document.getElementById("pwd_reset_form").addEventListener("submit", resetPassword)
`
