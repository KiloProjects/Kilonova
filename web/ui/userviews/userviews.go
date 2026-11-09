// Package userviews renders the user profile, settings and Discord link pages.
package userviews

import (
	"context"
	"html"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/bwmarrin/discordgo"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Topbar is the profile header with the overview/submissions/settings tabs. page is
// "overview", "submissions" or "settings".
func Topbar(ctx context.Context, u *kilonova.UserFull, page string) Node {
	authed := user.UserBriefContext(ctx)
	admin := authed.IsAdmin()
	self := authed.IsAuthed() && authed.ID == u.ID
	name := Text(u.Name)
	if u.DisplayName != "" {
		name = Text(u.DisplayName + " (" + u.Name + ")")
	}
	return Div(Class("segment-panel"),
		Img(Class("rounded-lg float-right"), Width("128"), Height("128"), Src("/profile/"+u.Name+"/avatar?s=128"), Alt(ui.TS(ctx, "profilePic")+" "+u.Name)),
		H1(
			If(admin, Group{Span(Class("tag rounded-lg font-semibold badge-blue"), Text("#"+strconv.Itoa(u.ID))), Text(" ")}),
			name,
		),
		P(ui.T(ctx, "created_at"), Text(": "), ui.ServerTimestamp(u.CreatedAt)),
		If(u.Generated, P(Text("!!!"), ui.T(ctx, "generated_acc"))),
		If(admin || self, P(
			ui.T(ctx, "email"), Text(": "+u.Email+" ("), ui.T(ctx, "email.verified"), Text("? "),
			If(u.VerifiedEmail, ui.T(ctx, "yes")),
			If(!u.VerifiedEmail, Group{ui.T(ctx, "no"), Text(" | "), A(Href("#"), Attr("onclick", "resendEmail()"), ui.T(ctx, "resend"))}),
			Text(")"),
		)),
		Script(Raw(`
async function resendEmail() {
	bundled.apiToast(await bundled.postCall('/user/resendEmail'))
}
`)),
		Div(Style("clear: both;")),
		Nav(Class("mt-1"), ui.Tabs(page,
			ui.Tab{Key: "overview", Href: "/profile/" + u.Name, Label: ui.T(ctx, "profile_nav")},
			ui.Tab{Key: "submissions", Href: "/profile/" + u.Name + "/submissions", Label: ui.T(ctx, "submissions")},
			ui.Tab{Key: "settings", Href: "/settings", Label: ui.T(ctx, "settings"), Hide: !self},
		)),
	)
}

type ProfileParams struct {
	User *kilonova.UserFull
	// BioHTML is the rendered markdown of User.Bio.
	BioHTML string

	SolvedProblems    []*sudoapi.FullProblem
	SolvedCount       int
	AttemptedProblems []*sudoapi.FullProblem
	AttemptedCount    int

	ChangeHistory []*kilonova.UsernameChange
}

func Profile(ctx context.Context, params ProfileParams) Node {
	u := params.User
	authed := user.UserBriefContext(ctx)
	admin := authed.IsAdmin()
	self := authed.IsAuthed() && authed.ID == u.ID

	return Group{
		Topbar(ctx, u, "overview"),
		If(admin, manageUser(ctx, u)),
		If(u.Bio != "" || self, Div(Class("segment-panel reset-list statement-content enhance-tables"),
			H2(ui.T(ctx, "about_me_bio"),
				If(self && u.Bio != "", Group{Text(" "), A(Href("/settings#bio"), I(Class("fas fa-edit")), Text(" "), ui.T(ctx, "editBio"))}),
			),
			If(self && u.Bio == "", A(Href("/settings#bio"), I(Class("fas fa-edit")), Text(" "), ui.T(ctx, "createBio"))),
			Raw(params.BioHTML),
		)),
		Script(Rawf(profileJS, u.ID, u.ID)),
		If(params.SolvedCount > 0, Div(Class("segment-panel"),
			H2(countHeading(ctx, admin || self, params.SolvedCount, "oneSolvedProblem", "manySolvedProblems", "solvedProblems")),
			El("kn-pb-solvedby", Attr("enc", ui.EncodeJSON(params.SolvedProblems)), Attr("count", strconv.Itoa(params.SolvedCount)), Attr("userid", strconv.Itoa(u.ID))),
		)),
		If(params.AttemptedCount > 0, Div(Class("segment-panel"),
			H2(countHeading(ctx, admin || self, params.AttemptedCount, "oneAttemptedProblem", "manyAttemptedProblems", "attemptedProblems")),
			El("kn-pb-attemptedby", Attr("enc", ui.EncodeJSON(params.AttemptedProblems)), Attr("count", strconv.Itoa(params.AttemptedCount)), Attr("userid", strconv.Itoa(u.ID))),
		)),
		If(len(params.ChangeHistory) > 1, Div(Class("segment-panel reset-list"),
			H2(ui.T(ctx, "usernameChangeHistory")),
			Ul(Map(params.ChangeHistory, func(c *kilonova.UsernameChange) Node {
				return Li(Code(Text(c.Name)), Text(" ("), ui.ServerTimestamp(c.ChangedAt), Text(")"))
			})),
		)),
	}
}

func countHeading(ctx context.Context, exact bool, cnt int, oneKey, manyKey, plainKey string) Node {
	switch {
	case !exact:
		return ui.T(ctx, plainKey)
	case cnt == 1:
		return ui.T(ctx, oneKey)
	default:
		return ui.T(ctx, manyKey, cnt)
	}
}

func manageUser(ctx context.Context, u *kilonova.UserFull) Node {
	check := func(id string, checked bool) Node {
		return Div(Class("block my-2"),
			Label(Class("inline-flex items-center text-lg"),
				Input(Class("form-checkbox"), Name(id), ID(id), Type("checkbox"), If(checked, Checked())),
				Text(" "),
				Span(Class("ml-2"), ui.T(ctx, id)),
			),
		)
	}
	return Group{
		Form(ID("managerUserForm"), Class("segment-panel"), AutoComplete("off"),
			H2(ui.T(ctx, "manageUser")),
			P(A(Class("btn btn-blue"), Href("/profile/"+u.Name+"/sessions"), ui.T(ctx, "manage_sessions_btn"))),
			check("can_login", !u.LockedLogin),
			check("must_change_uname", u.NameChangeForced),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "change_name"), Text(":")), Text(" "),
				Input(Type("text"), ID("manage_change_name"), Style("filter: none"), Class("form-input"), Value(u.Name), AutoComplete("off")),
			),
			ui.SubmitButton(Class("mr-2"), ui.T(ctx, "button.update")),
			Text(" "),
			If(!u.Admin, Group{ui.DangerButton(Class("mr-2"), Attr("onclick", "deleteAccount()"), ui.T(ctx, "deleteAccount")), Text(" ")}),
			ui.DangerButton(Class("mr-2"), Attr("onclick", "pruneBio()"), ui.T(ctx, "purgeBio")),
			Text(" "),
			ui.DangerButton(Class("mr-2"), Attr("onclick", "deauthAll()"), ui.T(ctx, "deauthAll")),
		),
		Script(Rawf(manageUserJS, u.ID, u.ID)),
	}
}

// manageUserJS takes the user ID twice.
const manageUserJS = `
async function updateUserStatus(e) {
	e.preventDefault()
	const data = {
		lockout: !document.getElementById("can_login").checked,
		new_name: document.getElementById("manage_change_name").value,
		force_username_change: document.getElementById("must_change_uname").checked,
	}

	const rez = await bundled.postCall("/user/byID/%d/moderation/manage", data)
	if(rez.status == "error") {
		bundled.apiToast(rez)
		return
	}
	window.location.assign(` + "`/profile/${data.new_name}`" + `);
}

async function deauthAll() {
	const rez = await bundled.postCall("/user/byID/%d/deauthAll", {})
	if(rez.status == "error") {
		bundled.apiToast(rez)
		return
	}
	bundled.apiToast(rez)
}

document.getElementById("managerUserForm").addEventListener("submit", updateUserStatus);
`

// profileJS takes the user ID twice.
const profileJS = `
async function deleteAccount() {
	if(!(await bundled.confirm(bundled.getText("delAccConfirmation")))) {
		return
	}

	let res = await bundled.postCall("/user/byID/%d/moderation/deleteUser", {});
	if(res.status === "success") {
		window.history.back();
	}
	bundled.apiToast(res);
}

async function pruneBio() {
	if(!(await bundled.confirm(bundled.getText("bioPruneConfirmation")))) {
		return
	}

	let res = await bundled.postCall("/user/byID/%d/setBio", {bio: "" });
	if(res.status === "success") {
		window.location.reload();
	}
	bundled.apiToast(res);
}
`

type SettingsParams struct {
	User *kilonova.UserFull
	// AvatarHelpHTML is the rendered markdown explaining the avatar choice
	// (changeAvatarWithDiscord or changeAvatarNoDiscord).
	AvatarHelpHTML string
}

func Settings(ctx context.Context, params SettingsParams) Node {
	u := params.User
	label := func(key string, input Node) Node {
		return Label(Class("block mb-2"), Span(Class("form-label"), ui.T(ctx, key), Text(": ")), Text(" "), input)
	}
	return Group{
		Topbar(ctx, u, "settings"),
		Div(Class("segment-panel"),
			H1(Text(" "), ui.T(ctx, "settingsHeader"), Text(" ")),
			P(
				A(Href("/settings/integrations"), ui.T(ctx, "authorized_apps")),
				If(user.UserBriefContext(ctx).IsAdmin(), Group{Text(" | "), A(Href("/settings/oauth"), ui.T(ctx, "oauth_clients"))}),
			),
			Div(Class("segment-panel"),
				H2(Text(" "), ui.T(ctx, "updateBio"), Text(" ")),
				Form(ID("bio_form"),
					Textarea(ID("newBio"), Class("form-textarea w-full mb-1"), AutoCapitalize("sentences"), AutoComplete("off"), Placeholder("Bio..."), Text(html.UnescapeString(u.Bio))),
					P(ID("bio_len_notice"), Class("text-sm text-muted mb-2"), Span(ID("bio_len"), Text(strconv.Itoa(len(u.Bio)))), Text(" / 10000")),
					Button(Class("btn btn-blue"), Attr("onclick", "updateBio()"), ui.T(ctx, "button.update")),
				),
				Script(Raw(bioLenJS)),
			),
			Div(Class("segment-panel"),
				H2(Text(" "), ui.T(ctx, "updateLang"), Text(" ")),
				Form(ID("lang_form"), AutoComplete("off"),
					Select(ID("langSelect"), Class("form-select block my-2"),
						Option(Value("ro"), If(u.PreferredLanguage == "ro", Selected()), Text("🇷🇴 Română")),
						Option(Value("en"), If(u.PreferredLanguage == "en", Selected()), Text("🇬🇧 English")),
					),
					ui.SubmitButton(ui.T(ctx, "button.update")),
				),
			),
			Form(Class("segment-panel"), ID("name_change_form"), AutoComplete("off"),
				H2(ui.T(ctx, "changeName")),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "newUsername"), Text(":")), Text(" "),
					Input(Type("text"), ID("name_change_name"), Class("form-input"), Required()),
				),
				label("pwdConfirmation", Input(Class("form-input"), Type("password"), ID("name_change_pwd"))),
				Button(Class("btn btn-blue"), ui.T(ctx, "button.update")),
			),
			Form(Class("segment-panel"), ID("avatar_change_form"), AutoComplete("off"),
				H2(ui.T(ctx, "changeAvatar")),
				Select(ID("avatarSelect"), Class("form-select block my-2"),
					Option(Value("gravatar"), If(u.AvatarType == "gravatar", Selected()), Text("Gravatar")),
					Option(Value("discord"), If(u.AvatarType == "discord", Selected()), Text("Discord")),
				),
				Raw(params.AvatarHelpHTML),
				Button(Class("btn btn-blue"), ui.T(ctx, "button.update")),
			),
			Form(Class("segment-panel"), ID("pwd_change_form"),
				H2(Text(" "), ui.T(ctx, "updatePwd"), Text(" ")),
				label("password", Input(Class("form-input"), Type("password"), ID("pwd_change_org"))),
				label("newPwd", Input(Class("form-input"), Type("password"), ID("pwd_change_pwd"), AutoComplete("off"))),
				label("newPwdConfirm", Input(Class("form-input"), Type("password"), ID("pwd_change_pwd1"), AutoComplete("off"))),
				ui.SubmitButton(ui.T(ctx, "button.update")),
			),
			Form(Class("segment-panel"), ID("email_change_form"),
				H2(Text(" "), ui.T(ctx, "updateEmail"), Text(" ")),
				label("newEmail", Input(Class("form-input"), Type("email"), ID("em_change_email"), AutoComplete("off"))),
				label("pwdConfirmation", Input(Class("form-input"), Type("password"), ID("em_change_pwd"))),
				ui.SubmitButton(ui.T(ctx, "button.update")),
			),
			Script(Raw(settingsJS)),
		),
	}
}

const bioLenJS = `
const maxSize = 10000
document.getElementById("newBio").addEventListener("input", e => {
	document.getElementById("bio_len").innerText = e.currentTarget.value.length
	document.getElementById("bio_len_notice").classList.toggle("text-muted", !(e.currentTarget.value.length > maxSize))
	document.getElementById("bio_len_notice").classList.toggle("text-red-600", e.currentTarget.value.length > maxSize)
	document.getElementById("bio_len_notice").classList.toggle("dark:text-red-400", e.currentTarget.value.length > maxSize)
})
`

const settingsJS = `
async function updateBio(e) {
	e?.preventDefault()
	var bio = document.getElementById("newBio").value;

	let res = await bundled.postCall("/user/self/setBio", {bio})
	bundled.apiToast(res)
}
async function updateLanguage(e) {
	e.preventDefault()
	var language = document.getElementById("langSelect").value;
	bundled.clearLanguageCookie()

	let res = await bundled.postCall("/user/self/setPreferredLanguage", {language})
	if(res.status === "success") {
		window.location.reload();
		return
	}
	bundled.apiToast(res)
}
async function updateName(e) {
	e.preventDefault()
	const data = {
		newName: document.getElementById("name_change_name").value,
		password: document.getElementById("name_change_pwd").value,
	}
	const rez = await bundled.postCall("/user/updateName", data)
	if(rez.status == "error") {
		bundled.apiToast(rez)
		return
	}
	window.location.reload()
}
async function updateAvatar(e) {
	e.preventDefault()
	const data = {
		avatarType: document.getElementById("avatarSelect").value
	}
	const rez = await bundled.postCall("/user/self/setAvatarType", data)
	bundled.apiToast(rez)
}
async function updatePassword(e) {
	e.preventDefault()
	let oldpwd = document.getElementById("pwd_change_org").value,
		pwd = document.getElementById("pwd_change_pwd").value,
		pwd1 = document.getElementById("pwd_change_pwd1").value;
	if(pwd != pwd1) {
		bundled.createToast({title: bundled.getText("differentPwds"), status: 'error'});
		return
	}
	let res = await bundled.postCall("/user/changePassword", {password:pwd, old_password:oldpwd})
	bundled.apiToast(res)
}
async function updateEmail(e) {
	e.preventDefault()
	let req = {
		password: document.getElementById("em_change_pwd").value,
		email: document.getElementById("em_change_email").value,
	};
	let res = await bundled.postCall("/user/changeEmail", req)
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	window.location.reload();
}
document.getElementById("bio_form").addEventListener("submit", updateBio)
document.getElementById("lang_form").addEventListener("submit", updateLanguage)
document.getElementById("name_change_form").addEventListener("submit", updateName)
document.getElementById("avatar_change_form").addEventListener("submit", updateAvatar)
document.getElementById("pwd_change_form").addEventListener("submit", updatePassword)
document.getElementById("email_change_form").addEventListener("submit", updateEmail)
`

// DiscordLink shows which Discord account, if any, u is linked to. discordUser is nil when unlinked.
func DiscordLink(ctx context.Context, u *kilonova.UserFull, discordUser *discordgo.User) Node {
	return Div(Class("segment-panel"),
		If(discordUser == nil, P(ui.T(ctx, "noDiscordLink"))),
		Iff(discordUser != nil, func() Node {
			return P(
				A(Href("/profile/"+u.Name),
					Img(Class("rounded-sm inline-block mr-2"), Src("/profile/"+u.Name+"/avatar?s=32"), Width("32"), Height("32"), Alt("Avatar")),
					Text(" "+u.Name),
				),
				Text(" "), ui.T(ctx, "discordLinkStatus"), Text(": "),
				Img(Class("rounded-sm inline-block mr-2"), Src("/profile/"+u.Name+"/discordAvatar?s=32"), Width("32"), Height("32"), Alt("Avatar")),
				Text(" "+discordUser.String()),
				If(user.UserBriefContext(ctx).IsAdmin(), Group{Text(" ("), Code(Text(discordUser.ID)), Text(")")}),
				Text("."),
			)
		}),
		A(Class("btn btn-blue my-2"), Href("/profile/"+u.Name), ui.T(ctx, "backToProfile")),
	)
}
