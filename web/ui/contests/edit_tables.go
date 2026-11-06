package contests

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func tableHead(ctx context.Context, keys ...string) Node {
	return THead(Tr(Map(keys, func(k string) Node {
		return Th(Class("kn-table-cell"), Scope("col"), Iff(k != "", func() Node { return ui.T(ctx, k) }))
	})))
}

func invitations(ctx context.Context, invs []*kilonova.ContestInvitation, inviters map[int]*kilonova.UserBrief) Node {
	table := P(ui.T(ctx, "noInvitations"))
	if len(invs) > 0 {
		table = Table(Class("kn-table"),
			tableHead(ctx, "id", "created_at", "author", "inviteUses", "expired"),
			TBody(ID("invitationsTableBody"), Map(invs, func(inv *kilonova.ContestInvitation) Node {
				return invitationRow(ctx, inv, inviters)
			})),
		)
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "header.contest.invitations")),
		table,
		Button(Attr("onclick", "createInvite(-1)"), Class("my-2 btn btn-blue"), ui.T(ctx, "createInvitation")),
		Text(" "),
		Button(Attr("onclick", "createInvite(1)"), Class("my-2 btn btn-blue"), ui.T(ctx, "createSingleUseInvitation")),
	)
}

func invitationRow(ctx context.Context, inv *kilonova.ContestInvitation, inviters map[int]*kilonova.UserBrief) Node {
	id := ui.JS(inv.ID)
	author := Text("-")
	if inv.CreatorID != nil {
		if u := inviters[*inv.CreatorID]; u != nil {
			author = A(Href("/profile/"+u.Name),
				Img(Class("rounded-sm inline-block mr-2"), Src("/profile/"+u.Name+"/avatar?s=32"), Width("32"), Height("32"), Alt("Avatar")),
				Text(" "+u.Name),
			)
		}
	}
	maxUses := "-"
	if inv.MaxCount != nil {
		maxUses = strconv.Itoa(*inv.MaxCount)
	}
	invalid := inv.Invalid()
	idText := Text(inv.ID)
	if invalid {
		idText = S(idText)
	}
	return Tr(Class("kn-table-row"),
		Th(Class("kn-table-cell"), Scope("row"),
			A(Href("/contests/invite/"+inv.ID), idText),
			Text(" "),
			Button(Attr("onclick", "copyInvitationURL("+id+")"), Class("ml-2 tag tag-other"), I(Class("fas fa-copy"))),
			Text(" "),
			Button(Attr("onclick", "bundled.inviteQRCode("+id+")"), Class("ml-1 tag tag-author"), I(Class("fas fa-qrcode"))),
		),
		Td(Class("kn-table-cell"), ui.ServerTimestamp(inv.CreatedAt)),
		Td(Class("kn-table-cell"), author),
		Td(Class("kn-table-cell"), Text(fmt.Sprintf("%d / %s", inv.RedeemCount, maxUses))),
		Td(Class("kn-table-cell"),
			Text(strconv.FormatBool(invalid)),
			If(!invalid, Group{
				Text(" "),
				Button(Attr("onclick", "invalidateInvite("+id+")"), Class("btn btn-red ml-2"), ui.T(ctx, "invalidate")),
			}),
		),
	)
}

func mossResults(ctx context.Context, subs []*kilonova.MOSSSubmission, pbs []*kilonova.ScoredProblem) Node {
	names := make(map[int]string, len(pbs))
	for _, pb := range pbs {
		names[pb.ID] = pb.Name
	}
	table := P(ui.T(ctx, "noMOSSHistory"))
	if len(subs) > 0 {
		table = Table(Class("kn-table"),
			tableHead(ctx, "id", "created_at", "problemSingle", "subcount", ""),
			// Duplicate id kept from the template.
			TBody(ID("invitationsTableBody"), Map(subs, func(s *kilonova.MOSSSubmission) Node {
				name, ok := names[s.ProblemID]
				if !ok {
					name = "???"
				}
				var link Node = Group{I(Class("fas fa-spinner animate-spin")), Text(" "), ui.T(ctx, "waiting")}
				if s.URL != "" {
					link = A(Class("btn btn-blue"), Href(s.URL), ui.T(ctx, "view"))
				}
				return Tr(Class("kn-table-row"),
					Th(Class("kn-table-cell"), Scope("row"), Text(strconv.Itoa(s.ID))),
					Td(Class("kn-table-cell"), ui.ServerTimestamp(s.CreatedAt)),
					Td(Class("kn-table-cell"),
						A(Href(fmt.Sprintf("/problems/%d", s.ProblemID)), Text(name)),
						Text(" - "+s.Language),
					),
					Td(Class("kn-table-cell"), Text(strconv.Itoa(s.SubCount))),
					Td(Class("kn-table-cell"), link),
				)
			})),
		)
	}
	return Div(Class("segment-panel"),
		H2(Class("inline-block mb-2"), ui.T(ctx, "header.contest.moss")),
		table,
		Button(Attr("onclick", "createMOSS()"), Class("my-2 btn btn-blue"), ui.T(ctx, "createMOSS")),
	)
}
