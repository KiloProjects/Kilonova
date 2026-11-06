package problempages

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/contests"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type SubmitFormParams struct {
	// Topbar.Contest, Topbar.CanSubmitInContest, Topbar.FullyVisible and Topbar.Page are read.
	Topbar    *problems.TopbarParams
	Languages []language.Lang
	// RunningContests the user can submit to, offered when not viewing through a contest.
	RunningContests []*kilonova.Contest
	// FilenameExplanationHTML is the rendered "filename_explanation" markdown.
	FilenameExplanationHTML string
}

// SubmitForm is the code submission form with its client-side logic.
func SubmitForm(ctx context.Context, p SubmitFormParams) Node {
	contestPicker, initiallyContest := submitContestPicker(ctx, p)
	return Group{
		Form(Class("segment-panel"), ID("sendSubForm"), AutoComplete("off"),
			contestPicker,
			Span(ID("submit_attempts_label"), Classes{"block mb-2 form-label": true, "hidden": !initiallyContest},
				ui.T(ctx, "remainingAttempts"), Text(": "), Span(ID("submit_attempts"), ui.T(ctx, "loading")),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "language"), Text(":")),
				Select(ID("sub_language"), Class("form-select"),
					Map(p.Languages, func(l language.Lang) Node {
						return Option(Value(l.InternalName()), If(l.InternalName() == "cpp17", Selected()), Text(l.PrintableName()))
					}),
				),
			),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "submit_style"), Text(":")),
				Select(ID("submit_style"), Class("form-select"), AutoComplete("off"),
					Option(Value("code"), ui.T(ctx, "code_insert")),
					Option(Value("file"), ui.T(ctx, "file")),
				),
			),
			Div(ID("cm_label"), Class("block mb-2"),
				Label(ID("filename_label"), Class("block mb-2 hidden"),
					Span(Class("form-label"), ui.T(ctx, "codeFilename")),
					Input(Class("form-input"), ID("filename"), Type("text"), Value("Main.java"), AutoComplete("off")),
					Span(Class("block text-sm text-muted"), Raw(p.FilenameExplanationHTML)),
				),
				Textarea(ID("SubArea"), Style("display: none;"), AutoComplete("off"), Aria("hidden", "true")),
			),
			Label(ID("file_label"), Class("block mb-2 hidden"),
				Span(Class("form-label"), ui.T(ctx, "upload_file"), Text(":")),
				Input(Class("form-input"), ID("submit_file"), Type("file"), AutoComplete("off")),
			),
			Button(Type("submit"), Class("btn btn-blue my-2"), ui.T(ctx, "send")),
		),
		Script(Raw(fmt.Sprintf(submitFormJS, p.Topbar.Problem.ID, ui.JS(p.Topbar.Page), initialAttemptsCall(initiallyContest)))),
	}
}

// submitContestPicker picks the contest a submission goes to; initiallyContest reports
// whether a contest is preselected, so the remaining attempts are shown from the start.
func submitContestPicker(ctx context.Context, p SubmitFormParams) (Node, bool) {
	hidden := func(v string) Node { return Input(Type("hidden"), ID("sub_contestid"), Value(v)) }
	if c := p.Topbar.Contest; c != nil {
		if p.Topbar.CanSubmitInContest {
			return Group{H1(Class("mt-2"), ui.T(ctx, "uploadContestSub")), hidden(fmt.Sprint(c.ID))}, true
		}
		return Group{H1(Class("mt-2"), ui.T(ctx, "upsolveSub")), hidden("-1")}, false
	}
	title := H1(Class("mt-2"), ui.T(ctx, "uploadSub"))
	cs := p.RunningContests
	if len(cs) == 0 {
		return Group{title, hidden("-1")}, false
	}
	option := func(c *kilonova.Contest, selected bool) Node {
		return Option(Value(fmt.Sprint(c.ID)), If(selected, Selected()), Text(c.Name))
	}
	var options Node
	switch {
	case p.Topbar.FullyVisible:
		options = Group{Option(Value("-1"), Selected(), Text("-")), Map(cs, func(c *kilonova.Contest) Node { return option(c, false) })}
	case len(cs) == 1:
		options = option(cs[0], true)
	default:
		options = Group{Option(Value(""), Disabled(), Selected(), ui.T(ctx, "selectContest")), Map(cs, func(c *kilonova.Contest) Node { return option(c, false) })}
	}
	return Group{title, Label(Class("block mb-2"),
		Span(Class("form-label"), ui.T(ctx, "contest"), Text(": ")),
		Select(ID("sub_contestid"), Class("form-select"), Required(), options),
	)}, !p.Topbar.FullyVisible
}

func initialAttemptsCall(initiallyContest bool) string {
	if initiallyContest {
		return "reloadRemainingAttempts()"
	}
	return ""
}

type SubmitPageParams struct {
	Form           SubmitFormParams
	ContestSidebar *contests.SidebarParams // set when viewing through a contest
	SubViewer      Node                    // may be nil
}

// SubmitPage is the full-page submission form followed by the user's past submissions.
func SubmitPage(ctx context.Context, p SubmitPageParams) Node {
	wrapper := "page-content-full-wrapper min-h-0"
	if p.Form.Topbar.Contest != nil {
		wrapper = "page-content-wrapper min-h-0"
	}
	return Group{
		problems.Topbar(ctx, p.Form.Topbar),
		Div(Class("page-holder mb-0"),
			Div(Class(wrapper), SubmitForm(ctx, p.Form)),
			Iff(p.ContestSidebar != nil, func() Node {
				return Aside(Class("page-sidebar min-h-0"), contests.Sidebar(ctx, *p.ContestSidebar))
			}),
			Div(Class("page-content-full-wrapper pt-0"), p.SubViewer),
		),
	}
}

// Placeholders: problem ID, page name (JS string), initial remaining-attempts call.
const submitFormJS = `
const problemID = %[1]d;
const pageName = %[2]s;

function isOutputOnly() {
	const languages = Object.values(document.getElementById("sub_language").options).map(x => x.value);
	return languages.length === 1 && (languages[0] === "outputOnly" || languages[0] === "ai")
}

function isJava() {
	return document.getElementById("sub_language")?.value === "java";
}

let cmConfig = {
	textArea: document.getElementById("SubArea"),
	language: bundled.languages[bundled.getCodeLangPreference()],
}

if(pageName === "pb_submit") {
	cmConfig.dynamicSize = true;
	cmConfig.autoFocus = true;
}

let cm = new vendored.KNEditor(cmConfig)

document.getElementById("submit_style").addEventListener("change", (e) => {
	const val = e.currentTarget.value;
	if(!isOutputOnly()) {
		bundled.setSubmitStyle(val);
	}
	document.getElementById("file_label")?.classList.toggle("hidden", val === "code");
	document.getElementById("cm_label")?.classList.toggle("hidden", val !== "code");
	document.getElementById("filename_label")?.classList.toggle("hidden", !isJava());
})

document.addEventListener("DOMContentLoaded", () => {
	let val = bundled.getSubmitStyle();
	if(isOutputOnly()) val = "file";

	document.getElementById("submit_style").value = val;
	document.getElementById("file_label")?.classList.toggle("hidden", val === "code");
	document.getElementById("cm_label")?.classList.toggle("hidden", val !== "code");

	var pref = bundled.getCodeLangPreference();
	if(Object.values(document.getElementById("sub_language").options).map(x => x.value).includes(pref)) {
		document.getElementById("sub_language").value = pref
	}
	document.getElementById("filename_label")?.classList.toggle("hidden", !isJava());
})

document.getElementById("sub_language").addEventListener("change", (e) => {
	bundled.setCodeLangPreference(e.target.value)
	let lang = bundled.languages[e.target.value]
	if (lang !== null) {
		cm.setLanguage(lang)
	}
	console.log(isJava());
	document.getElementById("filename_label")?.classList.toggle("hidden", !isJava());
})

async function sendSub() {
	let form = new FormData();
	form.set("problem_id", problemID);
	form.set("language", document.getElementById("sub_language").value)

	if(document.getElementById("submit_style").value === "code") {
		const code = cm.getText().trim()
		if(code.length === 0) {
			bundled.apiToast({status: "error", data: bundled.getText("no_code")})
			return;
		}
		let filename = "code";
		if(isJava()) {
			filename = document.getElementById("filename").value;
			if(filename.length === 0 || filename.endsWith(".java") === false) {
				bundled.apiToast({status: "error", data: bundled.getText("no_filename")})
				return;
			}
		}
		form.set("code", new File([code], filename, {type: "text/plain;charset=utf-8"}));
	} else {
		const fInput = document.getElementById("submit_file");
		if(fInput.files.length > 1) {
			bundled.apiToast({status: "error", data: bundled.getText("invalid_file")})
			return
		} else if(fInput.files.length == 0) {
			bundled.apiToast({status: "error", data: bundled.getText("no_code")})
			return
		}
		form.set("code", fInput.files[0]);
	}

	if(document.getElementById("sub_contestid").value !== "-1") {
		const val = parseInt(document.getElementById("sub_contestid").value);
		if(isNaN(val)) {
			console.warn("Submission form's contest ID is not a number")
		} else {
			form.set("contest_id", val);
		}
	}

	let res = await bundled.multipartCall("/submissions/submit", form)
	if (res.status == "error") {
		bundled.apiToast(res)
		return
	}
	bundled.createToast({ title: bundled.getText("uploaded"), description: ` + "`" + `<a href="/submissions/${res.data}">${bundled.getText("view")}</btn>` + "`" + ` })
	console.log(res.data, bundled.makeSubWaiter(res.data));
	document.dispatchEvent(new CustomEvent("kn-poll"));
	htmx.trigger("#older_subs", "kn-poll", {})
}

let oldContestID = "";

function reloadRemainingAttempts() {
	let contestID = document.getElementById("sub_contestid").value
	let attemptsLabel = document.getElementById("submit_attempts_label");
	let attemptsText = document.getElementById("submit_attempts");
	if(oldContestID != contestID) {
		attemptsText.innerText = bundled.getText("loading")
	}
	attemptsLabel.classList.toggle("hidden", contestID == "-1");
	if( contestID!= "-1") {
		bundled.getCall(` + "`" + `/contest/${contestID}/problemRemainingCount` + "`" + `).then(rez => {
			if(rez.status == "error") {
				bundled.apiToast(rez)
				return
			}
			if(problemID in rez.data.counts) {
				let pb = rez.data.counts[problemID];
				if(!pb.limited) {
					attemptsText.innerText = bundled.getText("unlimitedAttempts")
				} else {
					attemptsText.innerText = Math.max(pb.remaining, 0);
				}
			} else {
				attemptsText.innerText = "N/A"
			}
			oldContestID = contestID
		})
	} else {
		oldContestID = contestID
	}
}

%[3]s

document.getElementById("sub_contestid").addEventListener("change", (e) => {
	reloadRemainingAttempts();
})
document.addEventListener("kn-poll", () => {
	reloadRemainingAttempts();
})

const debounced = bundled.debounce(() => sendSub().catch(console.error), 400, {leading: true, trailing: false})

// send submission on ctrl + enter
document.body.addEventListener('keydown', (event) => {
	if(event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
		debounced()
	}
});

document.getElementById("sendSubForm").addEventListener("submit", (e) => {
	e.preventDefault();
	debounced()
})
`
