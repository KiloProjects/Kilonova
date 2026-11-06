package contests

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// InviteParams is a contest invitation landing page.
type InviteParams struct {
	Contest           *kilonova.Contest
	Invite            *kilonova.ContestInvitation
	Inviter           *kilonova.UserBrief // nil when unknown
	Registration      *kilonova.ContestRegistration
	AlreadyRegistered bool
}

// Invite shows the contest an invitation is for and lets the viewer accept it.
func Invite(ctx context.Context, p InviteParams) Node {
	c := p.Contest
	var body Node
	switch {
	case p.Invite.Expired:
		body = P(ui.T(ctx, "expiredInvite"))
	case p.AlreadyRegistered:
		body = P(ui.T(ctx, "alreadyRegistered"), Text(". "),
			A(Href(fmt.Sprintf("/contests/%d", c.ID)), ui.T(ctx, "alreadyRegistered.jump")), Text("."))
	case p.Invite.Invalid():
		body = P(ui.T(ctx, "inviteManyRedeems"))
	default:
		body = Group{
			Div(Class("segment-panel"),
				H2(Class("font-normal"), ui.T(ctx, "contestInfo")),
				Div(Class("px-2"),
					P(ui.T(ctx, "startTime"), Text(": "), ui.ServerTimestamp(c.StartTime)),
					P(ui.T(ctx, "duration"), Text(": "+c.EndTime.Sub(c.StartTime).Round(time.Minute).String())),
					P(ui.T(ctx, "status"), Text(": "), status(ctx, c, p.Registration)),
					If(usacoStyle(c), P(ui.T(ctx, "contest_user_time"), Text(": "+usacoDuration(c)))),
				),
			),
			Button(ID("acceptButton"), Class("my-2 btn btn-blue"), ui.T(ctx, "accept")),
			Script(Raw(fmt.Sprintf(inviteScript, ui.JS(p.Invite.ID), c.ID))),
		}
	}
	return Div(Class("segment-panel"),
		H1(Class("mb-2"),
			ui.TRaw(ctx, "header.invitation", html.EscapeString(c.Name)), Text(" "),
			Iff(p.Inviter != nil, func() Node {
				return ui.TRaw(ctx, "header.invitation.from", html.EscapeString(p.Inviter.Name))
			}),
		),
		body,
	)
}

const inviteScript = `
        document.getElementById("acceptButton").addEventListener("click", async () => {
            const res = await bundled.postCall("/contest/acceptInvitation", {invite_id: %s})
            if(res.status === "error") {
                bundled.apiToast(res)
                return
            }
            window.location.assign("/contests/%d")
        })
`
