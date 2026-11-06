package contests

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// EditParams is the contest management page.
type EditParams struct {
	Topbar      *problems.TopbarParams
	Contest     *kilonova.Contest
	Problems    []*kilonova.ScoredProblem
	Invitations []*kilonova.ContestInvitation
	// Inviters maps invitation creator IDs to users; missing entries render as "-".
	Inviters   map[int]*kilonova.UserBrief
	MOSS       []*kilonova.MOSSSubmission
	IsAdmin    bool
	IsProposer bool
}

// Edit is the contest management page.
func Edit(ctx context.Context, p EditParams) Node {
	c := p.Contest
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content-full-wrapper"),
				settingsForm(ctx, p),
				problemsForm(ctx, p.Problems),
				accessControl(ctx),
				descriptionForm(ctx, c),
				invitations(ctx, p.Invitations, p.Inviters),
				If(p.IsProposer, pblistForm(ctx)),
				If(p.IsAdmin, forceRegisterForm(ctx, c.ID)),
				If(c.Type == kilonova.ContestTypeOfficial, mossResults(ctx, p.MOSS, p.Problems)),
			),
		),
		editScripts(c),
	}
}

func problemsForm(ctx context.Context, pbs []*kilonova.ScoredProblem) Node {
	ids := make([]string, len(pbs))
	for i, pb := range pbs {
		ids[i] = strconv.Itoa(pb.ID)
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "header.contest.edit_problems")),
		Form(Class("mb-4"), ID("contest_problems_form"), AutoComplete("off"),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "pbs"), Text(":")),
				Text(" "),
				Input(Type("text"), ID("contest_list"), Class("form-input"), Pattern("([0-9]+,?)+"), Value(strings.Join(ids, ","))),
			),
			ui.SubmitButton(ui.T(ctx, "button.update")),
		),
	)
}

// accessControl lists editors and testers; the lists are filled in by editAccessScript.
func accessControl(ctx context.Context) Node {
	loading := func(id string) Node {
		return Div(ID(id), Class("list-group mb-2"), Div(Class("list-group-head"), ui.T(ctx, "loading")))
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "header.contest.access_control")),
		H3(ui.T(ctx, "editors"), Text(":")),
		loading("editors_list"),
		H3(ui.T(ctx, "testers"), Text(":")),
		loading("testers_list"),
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
				Button(Type("button"), Class("btn-blue font-bold py-2 px-4 rounded-r-sm"), Attr("onclick", "addTester()"), ui.T(ctx, "add_tester")),
			),
		),
	)
}

func descriptionForm(ctx context.Context, c *kilonova.Contest) Node {
	return Form(Class("segment-panel"), ID("contest_description_form"), AutoComplete("off"),
		H2(Class("inline-block mb-2"), ui.T(ctx, "header.edit.description")),
		Div(Class("mb-2"), Textarea(ID("contestDesc"), Class("form-textarea"), Text(c.Description))),
		Div(Class("mb-2"), ui.SubmitButton(ui.T(ctx, "button.update"))),
	)
}

func pblistForm(ctx context.Context) Node {
	return Form(Class("segment-panel"), ID("contest_pblist_form"),
		H2(Class("inline-block mb-2"), ui.T(ctx, "header.contest.create_pblist")),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "list_parent_id"), Text(":")),
			Text(" "),
			Input(ID("cr_pblist_parent_id"), Type("number"), Class("form-input")),
		),
		ui.SubmitButton(ui.T(ctx, "button.create")),
	)
}

func forceRegisterForm(ctx context.Context, contestID int) Node {
	return Group{
		Form(Class("segment-panel"), ID("force_register_form"), AutoComplete("off"),
			H2(Class("inline-block mb-2"), ui.T(ctx, "header.contest.force_register")),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "username"), Text(":")),
				Text(" "),
				Input(ID("force_register_uname"), Type("text"), Class("form-input"), Required()),
			),
			ui.SubmitButton(ui.T(ctx, "button.add")),
		),
		Script(Raw(`
            document.getElementById("force_register_form").addEventListener("submit", async (e) => {
                e.preventDefault();
                let res = await bundled.postCall("/contest/` + strconv.Itoa(contestID) + `/forceRegister", {
                    name: document.getElementById("force_register_uname").value,
                })
                bundled.apiToast(res)
            })
`)),
	}
}

// editScripts wires up the page. Only the contest ID and its times are interpolated.
func editScripts(c *kilonova.Contest) Node {
	id := strconv.Itoa(c.ID)
	datetimes := "setDatetime(\"start_time\", " + jsTime(c.StartTime) + ");\n" +
		"setDatetime(\"end_time\", " + jsTime(c.EndTime) + ");\n"
	if c.LeaderboardFreeze != nil {
		datetimes += "setDatetime(\"freeze_time\", " + jsTime(*c.LeaderboardFreeze) + ");\n"
	} else {
		datetimes += "document.getElementById(\"freeze_time\").value = \"\";\n"
	}
	return Group{
		Script(Raw(strings.ReplaceAll(editInviteScript, "{{ID}}", id))),
		Script(Raw(editDatetimeScript + datetimes)),
		Script(Raw(editToggleScript)),
		Script(Raw("\nconst contest_id = " + id + ";" + editUpdateScript)),
		Script(Raw(editCloneScript)),
		Script(Raw(editDeleteScript)),
		Script(Raw(editProblemsScript)),
		Script(Raw(editAccessScript)),
	}
}

// jsTime is t as a JavaScript string literal.
func jsTime(t time.Time) string {
	b, _ := json.Marshal(t) // time.Time marshals as an RFC 3339 string
	return string(b)
}
