// Package adminviews renders admin-only pages.
package adminviews

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type UsersListParams struct {
	Users      []*kilonova.UserBrief
	Name       string
	Pagination Node // nil when there's a single page
}

func UsersList(ctx context.Context, params UsersListParams) Node {
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "users")),
		Form(Class("block my-2"), Method("GET"), Attr("hx-boost", "true"),
			Label(
				Span(Class("form-label"), ui.T(ctx, "name"), Text(":")),
				Input(Class("form-input my-1 block"), Type("text"), Name("name"), Placeholder("AlexVasiluta"), AutoComplete("off"), Value(params.Name)),
			),
			ui.SubmitButton(ui.T(ctx, "search_user")),
		),
		params.Pagination,
		If(len(params.Users) > 0, Table(Class("kn-table"),
			THead(Tr(
				Th(Class("kn-table-cell w-1/12"), Scope("col"), ui.T(ctx, "id")),
				Th(Class("kn-table-cell"), Scope("col"), Text("User")),
				Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "admin")),
				Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "proposer")),
			)),
			TBody(Map(params.Users, userRow)),
		)),
		If(len(params.Users) == 0, Div(Class("text-4xl mx-auto my-auto w-full mt-10 mb-10 text-center"), ui.T(ctx, "noUserFound"))),
	)
}

func userRow(u *kilonova.UserBrief) Node {
	return Tr(Class("kn-table-row"),
		Td(Class("kn-table-cell"), Text(strconv.Itoa(u.ID))),
		Td(Class("kn-table-cell"),
			A(Href("/profile/"+u.Name), Class("inline-flex align-middle items-center"),
				Img(Width("32"), Height("32"), Class("flex-none mr-2 rounded-sm"), Src(fmt.Sprintf("/profile/%s/avatar?s=32", u.Name)), Loading("lazy")),
				Text(" "+u.Name),
				If(u.DisplayName != "", Text(" ("+u.DisplayName+")")),
			),
		),
		Td(Class("kn-table-cell"), yesNo(u.Admin)),
		Td(Class("kn-table-cell"), yesNo(u.Proposer)),
	)
}

func yesNo(b bool) Node {
	if b {
		return Span(Class("text-lg"), Text("✅"))
	}
	return Span(Class("text-lg"), Text("❌"))
}
