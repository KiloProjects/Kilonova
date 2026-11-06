package pbedit

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Access is the problem's access control page. The editor and viewer lists are loaded client-side.
func Access(ctx context.Context, pb *kilonova.Problem, topbar *problems.TopbarParams) Node {
	loading := func(id string) Node {
		return Div(ID(id), Class("list-group mb-2"), Div(Class("list-group-head"), ui.T(ctx, "loading")))
	}
	return Group{page(ctx, topbar,
		Div(Class("page-content-wrapper"),
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "header.edit.access_control")),
				H3(ui.T(ctx, "editors"), Text(":")),
				loading("editors_list"),
				H3(ui.T(ctx, "viewers"), Text(":")),
				loading("viewers_list"),
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
	), script("access.js", map[string]any{"PBID": pb.ID})}
}
