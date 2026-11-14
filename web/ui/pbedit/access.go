package pbedit

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type AccessParams struct {
	Problem      *kilonova.Problem
	Topbar       *problems.TopbarParams
	Editors      []*kilonova.UserBrief
	Viewers      []*kilonova.UserBrief
	AuthedUserID int
}

// Access is the problem's access control page.
func Access(ctx context.Context, p AccessParams) Node {
	return Group{page(ctx, p.Topbar,
		Div(Class("page-content-wrapper"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "header.edit.access_control")),
				Div(ID("access_lists"),
					ui.UserList(ctx, ui.UserListParams{ID: "editors_list", Title: ui.T(ctx, "editors"), Users: p.Editors, Open: true, Action: ui.StripAccessAction(ctx, p.AuthedUserID)}),
					ui.UserList(ctx, ui.UserListParams{ID: "viewers_list", Title: ui.T(ctx, "viewers"), Users: p.Viewers, Open: true, Action: ui.StripAccessAction(ctx, p.AuthedUserID)}),
				),
			),
		),
		Aside(Class("page-sidebar"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "add_user")),
				Label(Class("block"),
					Span(Class("form-label"), ui.T(ctx, "username"), Text(":")),
					Text(" "),
					Input(Class("form-input my-1 block"), Type("text"), ID("userToAdd"), Placeholder("AlexVasiluta"), AutoComplete("off")),
				),
				Div(Class("mt-2 mb-2 inline-flex"),
					Button(Type("button"), Class("btn-blue font-bold py-2 px-4 rounded-l-sm"), Attr("onclick", "addEditor()"), ui.T(ctx, "add_editor")),
					Text(" "),
					Button(Type("button"), Class("btn-blue font-bold py-2 px-4 rounded-r-sm"), Attr("onclick", "addViewer()"), ui.T(ctx, "add_viewer")),
				),
			),
		),
	), script("access.js", map[string]any{"PBID": p.Problem.ID})}
}
