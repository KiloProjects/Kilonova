// Package proposer renders the forms proposers use to create contests and problem lists.
package proposer

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// CreateContestForm lets proposers create a virtual contest; only admins may pick official.
func CreateContestForm(ctx context.Context) Node {
	admin := user.UserBriefContext(ctx).IsAdmin()
	return Group{
		H2(ui.T(ctx, "createContest")),
		Form(ID("contest_form"),
			field(ctx, "name", Input(ID("contest_title"), Type("text"), Class("form-input"), Required())),
			Label(Class("block my-2"),
				Span(Classes{"form-label": true, "text-muted": !admin}, ui.T(ctx, "contest_type"), Text(":")),
				Select(ID("contest_type"), Class("form-select"), Required(), AutoComplete("off"),
					Option(Value("official"), If(!admin, Disabled()), ui.T(ctx, "official")),
					Option(Value("virtual"), Selected(), ui.T(ctx, "virtual")),
				),
				P(Class("block text-muted text-sm"), ui.T(ctx, "contest_type_warn")),
			),
			ui.SubmitButton(ui.T(ctx, "button.create")),
		),
		// TODO: Remove and use htmx instead
		Script(Raw(`async function createContest(e) {
	e.preventDefault();
	let name = document.getElementById("contest_title").value;
	let type = document.getElementById("contest_type").value;
	let res = await bundled.postCall("/contest/create", {name, type})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	window.location.assign(` + "`/contests/${res.data}`" + `)
}
document.getElementById("contest_form").addEventListener("submit", createContest)`)),
	}
}

// CreateProblemList creates a list, nested under parentList when it isn't nil.
func CreateProblemList(ctx context.Context, parentList *kilonova.ProblemList) Node {
	var parent Node
	if parentList != nil {
		parent = Input(ID("cr_pblist_parent_id"), Type("hidden"), Value(strconv.Itoa(parentList.ID)))
	} else {
		parent = field(ctx, "list_parent_id", Input(ID("cr_pblist_parent_id"), Type("number"), Class("form-input")))
	}
	return Group{
		Form(ID("cr_pblist_form"),
			field(ctx, "name", Input(ID("cr_pblist_title"), Type("text"), Class("form-input"), Required())),
			field(ctx, "desc", Textarea(ID("cr_pblist_desc_tarea"), Class("hidden"))),
			field(ctx, "sublist_label", Input(Type("text"), ID("cr_pblist_sublists"), Class("form-input"), Pattern("([0-9]+,?)+"))),
			field(ctx, "pbs", Input(ID("cr_pblist_pbs"), Type("text"), Class("form-input"), Pattern("([0-9]+,?)+"))),
			parent,
			Div(Class("block my-2"),
				Label(Class("inline-flex items-center text-lg"),
					Input(Class("form-checkbox"), Name("sidebar_hidable"), ID("cr_sidebar_hidable"), Type("checkbox")),
					Span(Class("ml-2"), ui.T(ctx, "sidebar_hidable")),
				),
			),
			ui.SubmitButton(ui.T(ctx, "button.create")),
		),
		Script(Raw(`var cr_pblist_cm = new vendored.KNEditor({
	textArea: document.getElementById("cr_pblist_desc_tarea"),
	language: "md",
	dynamicSize: true,
})

async function createProblemList(e) {
	e.preventDefault()

	const parentID = parseInt(document.getElementById("cr_pblist_parent_id").value);

	let data = {
		title: document.getElementById("cr_pblist_title").value,
		description: cr_pblist_cm.getText(),
		ids: bundled.stringIntToNumber(document.getElementById("cr_pblist_pbs").value.split(',')),
		sublists: bundled.stringIntToNumber(document.getElementById("cr_pblist_sublists").value.split(',')),
		sidebar_hidable: document.getElementById("cr_sidebar_hidable").checked,
	};

	if(!isNaN(parentID)) {
		data.parent_id = parentID
	}

	let res = await bundled.bodyCall("/problemList/create", data)
	if(res.status === "success") {
		if(res.data.nested) {
			// go to parent instead
			window.location.assign(` + "`/problem_lists/${parentID}`" + `);
			return
		}
		window.location.assign(` + "`/problem_lists/${res.data.id}`" + `);
		return
	}
	bundled.apiToast(res)
}

document.getElementById("cr_pblist_form").addEventListener("submit", createProblemList)`)),
	}
}

// field is a labelled form row: "<label>: <input>".
func field(ctx context.Context, labelKey string, input Node) Node {
	return Label(Class("block my-2"),
		Span(Class("form-label"), ui.T(ctx, labelKey), Text(":")),
		input,
	)
}
