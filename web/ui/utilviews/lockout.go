// Package utilviews holds one-off full pages.
package utilviews

import (
	"context"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UserLockout forces the logged-in user to pick a new username before using the site.
func UserLockout(ctx context.Context) Node {
	name := user.UserBriefContext(ctx).Name
	return Group{
		ui.CenteredLayout(
			Form(ID("logoutForm"), Method("POST"), Action("/logout")),
			Form(ID("forceUpdateForm"), Class("segment-panel"), AutoComplete("off"),
				H1(ui.T(ctx, "usernameChangeForced")),
				P(Class("my-2"), ui.T(ctx, "usernameChangeDetails")),
				P(ui.T(ctx, "usernameChangeUniqueRemark")),
				Div(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "currentUsername"), Text(": ")),
					Strong(Class("text-lg"), Text(name)),
				),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "newUsername"), Text(": ")),
					Input(ID("forcedNewUsername"), Type("text"), Class("form-input"), Required()),
				),
				// Only warn users whose current name is valid, so old-style usernames aren't scared off.
				If(user.ValidUsername(name) == nil, P(Class("text-sm text-muted mb-2"), ui.T(ctx, "usernameChangeWarning"))),
				ui.SubmitButton(Class("mr-2"), ui.T(ctx, "button.update")),
				Button(Type("submit"), FormAttr("logoutForm"), Class("btn"), ui.T(ctx, "auth.logout")),
			),
		),
		// TODO: don't use this JS for updating usernames
		Script(Raw(`async function updateUsername(e) {
	e.preventDefault();
	const data = {
		newName: document.getElementById("forcedNewUsername").value,
	};
	const rez = await bundled.postCall("/user/updateName", data);
	if (rez.status == "error") {
		bundled.apiToast(rez);
		return;
	}
	window.location.reload();
}

document.getElementById("forceUpdateForm").addEventListener("submit", updateUsername);`)),
	}
}
