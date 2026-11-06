package proposer

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Panel is the proposer panel: create or import problems, create lists and contests.
func Panel(ctx context.Context) Node {
	return Group{
		H1(ui.T(ctx, "panel.proposer")),
		createProblem(ctx),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "createPbList")),
			CreateProblemList(ctx, nil),
		),
		Div(Class("segment-panel"), CreateContestForm(ctx)),
		importProblem(ctx),
	}
}

func statementOptions(ctx context.Context, defaultLang string) Node {
	opt := func(val, label string) Node {
		return Option(Value(val), If(val == defaultLang, Selected()), Text(label))
	}
	return Group{
		Option(Value(""), If(defaultLang == "", Selected()), ui.T(ctx, "dontCreate")),
		opt("ro", "🇷🇴 Română"),
		opt("en", "🇬🇧 English"),
		opt("de", "🇩🇪 Deutsch"),
	}
}

func createProblem(ctx context.Context) Node {
	var userID int
	if u := user.UserBriefContext(ctx); u != nil {
		userID = u.ID
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "createPb")),
		Form(ID("problem_create_form"), AutoComplete("off"),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "problemName")), Text(" "),
				Input(Type("text"), Class("form-input w-full"), Placeholder(ui.TS(ctx, "name")+"..."), ID("pbname"), Name("pbname"), Required()),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "createDefaultStatement")), Text(" "),
				Select(Class("form-select w-full"), ID("pbstatement"), Name("pbstatement"), statementOptions(ctx, "ro")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "addToPblist")), Text(" "),
				Input(Type("number"), Class("form-input w-full"), Placeholder(ui.TS(ctx, "pblist_id")+"..."), ID("pb_pblist"), Name("pb_pblist"), Value("")),
			),
			Div(Class("block mb-3"),
				Label(Class("inline-flex items-center text-lg"),
					Input(Class("form-checkbox"), Type("checkbox"), ID("cInputSwitch")), Text(" "),
					Span(Class("ml-2"), ui.T(ctx, "consoleInput")),
				),
			),
			Button(Class("btn btn-blue mb-2"), ui.T(ctx, "button.create")),
			Div(Class("reset-list segment-panel"),
				H3(Text("NEW (2023-07-08): "), A(Href("https://kilonova.ro/posts/problem-guide"), Text("The definitive problem guide")), Text(" was published.")),
				P(Text("It merges all the older guides into a single, comprehensive document, along with new information such as the Kilonova style guide. "), B(Text("Everyone")), Text(" should take a look at it (including veteran proposers), most importantly the style guide!")),
			),
			A(Class("btn btn-blue"), Href("/problems?editor_user="+strconv.Itoa(userID)), ui.T(ctx, "created_problems")),
		),
		Script(Raw(`
	async function createProblem(e) {
		e.preventDefault();
        let data = {
            title: document.getElementById("pbname").value,
            consoleInput: document.getElementById("cInputSwitch").checked,
            statementLang: document.getElementById("pbstatement").value,
        }
        const pblistID = parseInt(document.getElementById("pb_pblist").value);
        if(!isNaN(pblistID)) {
            data.pblistID = pblistID
        }
		const res = await bundled.postCall('/problem/create', data)
		if(res.status == "error") {
			bundled.apiToast(res)
			return
		}
		window.location.assign(`+"`/problems/${res.data}`"+`)
	}
	document.getElementById("problem_create_form").addEventListener("submit", createProblem)
`)),
	)
}

func importProblem(ctx context.Context) Node {
	noFiles, _ := json.Marshal(ui.TS(ctx, "noFiles"))
	return Div(ID("problem_import"), Class("segment-panel"),
		H2(ui.T(ctx, "importProblem")),
		Form(ID("problem_import_form"), AutoComplete("off"),
			Label(Class("block mb-2"),
				Span(Class("mr-2 text-xl"), ui.T(ctx, "file"), Text(": ")), Text(" "),
				Input(Class("form-input"), ID("importArchiveFile"), Type("file"), Accept(".zip"), Required(), Multiple()),
			),
			Label(Class("block my-2"),
				Span(Class("mr-2 text-xl"), Text(" "), ui.T(ctx, "scoreParams"), Text(":")), Text(" "),
				Input(ID("importScoreParams"), Type("text"), Class("form-input"), AutoComplete("off")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "createDefaultStatement"), Text(":")), Text(" "),
				Select(Class("form-select"), ID("statementImport"), statementOptions(ctx, "")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "addToPblist"), Text(":")), Text(" "),
				Input(Type("number"), Class("form-input"), Placeholder(ui.TS(ctx, "pblist_id")+"..."), ID("pb_import_pblist"), Value("")),
			),
			Button(Class("btn btn-blue mb-2"), ui.T(ctx, "button.upload")),
			P(Class("text-muted text-sm"), ui.T(ctx, "importProblemWarn")),
		),
		Script(Raw(`
async function importProblem(e) {
	e.preventDefault()
	var files = document.getElementById("importArchiveFile").files;
	if(files === null || files.length === 0) {
		bundled.createToast({status: "error", title: `+string(noFiles)+`})
		return
	}
	for(const file of files) {
		var form = new FormData();
		form.append("testArchive", file);
	    form.append("statementLang", document.getElementById("statementImport").value);
	    const scoreParams = document.getElementById("importScoreParams").value;
	    if(scoreParams.length > 0) {
	        form.append("scoreParameters", scoreParams);
	    }
	    const pblistID = parseInt(document.getElementById("pb_import_pblist").value);
	    if(!isNaN(pblistID)) {
	        form.append("pblistID", pblistID);
	    }

		let res = await bundled.multipartProgressCall("/problem/import", form)
		if(res.status === "success" && files.length === 1) {
			window.location.assign(`+"`/problems/${res.data}`"+`);
			return
		}
		bundled.apiToast(res)
	}
}

document.getElementById("problem_import_form").addEventListener("submit", importProblem)
`)),
	)
}
