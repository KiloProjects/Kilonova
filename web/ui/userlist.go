package ui

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova"
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

type UserListParams struct {
	ID    string
	Title Node
	Users []*kilonova.UserBrief
	Open  bool
	// Action renders the actions cell for u; a nil result leaves it empty. A nil Action hides the column.
	Action func(u *kilonova.UserBrief) Node
}

// UserList is a collapsible table of users, each with an optional actions cell.
func UserList(ctx context.Context, p UserListParams) Node {
	return h.Details(h.ID(p.ID), h.Class("mb-2"), If(p.Open, h.Open()),
		h.Summary(h.Class("text-xl my-2 cursor-pointer"), p.Title, Text(" ("+strconv.Itoa(len(p.Users))+")")),
		If(len(p.Users) == 0, h.P(h.Class("text-muted"), T(ctx, "no_users"))),
		If(len(p.Users) > 0, h.Table(h.Class("kn-table"),
			h.THead(h.Tr(
				h.Th(h.Class("kn-table-cell w-1/12"), h.Scope("col"), T(ctx, "id")),
				h.Th(h.Class("kn-table-cell"), h.Scope("col"), T(ctx, "username")),
				If(p.Action != nil, h.Th(h.Class("kn-table-cell w-1/12"), h.Scope("col"), T(ctx, "actions"))),
			)),
			h.TBody(Map(p.Users, func(u *kilonova.UserBrief) Node {
				return h.Tr(h.Class("kn-table-row"),
					h.Td(h.Class("kn-table-cell w-1/12"), Text(strconv.Itoa(u.ID))),
					h.Td(h.Class("kn-table-cell"),
						h.A(h.Href("/profile/"+u.Name), h.Class("inline-flex align-middle items-center"),
							h.Img(h.Width("32"), h.Height("32"), h.Class("flex-none mr-2 rounded-sm"), h.Src("/profile/"+u.Name+"/avatar?s=32"), h.Alt("Avatar"), h.Loading("lazy")),
							Text(u.Name),
							If(u.DisplayName != "", Text(" ("+u.DisplayName+")")),
						),
					),
					Iff(p.Action != nil, func() Node { return h.Td(h.Class("kn-table-cell w-1/12"), p.Action(u)) }),
				)
			})),
		)),
	)
}

// RemoveUserButton is a danger button that runs onclick to remove a user.
func RemoveUserButton(ctx context.Context, onclick string) Node {
	return DangerButton(Attr("onclick", onclick), T(ctx, "button.remove"))
}

// StripAccessAction is a UserList action calling the page's stripAccess(id); nobody can strip their own access.
func StripAccessAction(ctx context.Context, authedUserID int) func(u *kilonova.UserBrief) Node {
	return func(u *kilonova.UserBrief) Node {
		if u.ID == authedUserID {
			return nil
		}
		return RemoveUserButton(ctx, "stripAccess("+strconv.Itoa(u.ID)+")")
	}
}
