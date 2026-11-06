package contests

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// CommunicationParams is the announcements and questions page.
type CommunicationParams struct {
	PageParams
	Announcements []*kilonova.ContestAnnouncement
	AllQuestions  []*kilonova.ContestQuestion // editors only
	CanSubmit     bool
	UserQuestions []*kilonova.ContestQuestion // the viewer's own, when CanSubmit
}

// Communication shows announcements, lets editors post them and answer questions, and
// lets contestants ask questions.
func Communication(ctx context.Context, p CommunicationParams) Node {
	id := strconv.Itoa(p.Contest.ID)
	return Group{
		problems.Topbar(ctx, p.Topbar),
		Div(Class("page-holder"),
			Div(Class("page-content-full-wrapper"),
				Div(Class("segment-panel"),
					El("kn-announcements", Attr("contestid", id), Attr("encoded", ui.EncodeJSON(p.Announcements)),
						Attr("canedit", strconv.FormatBool(p.IsEditor && !p.Contest.Ended()))),
				),
				If(p.IsEditor, Group{
					Form(ID("announcement_create_form"), Class("segment-panel"), AutoComplete("off"),
						H2(ui.T(ctx, "create_announcement")),
						Label(
							Span(Class("form-label text-base"), ui.T(ctx, "contestAnnouncement"), Text(":")),
							Textarea(ID("announcement_area"), Class("form-textarea w-full my-2")),
						),
						ui.SubmitButton(ui.T(ctx, "create")),
					),
					Div(Class("segment-panel"),
						H2(ui.T(ctx, "received_questions")),
						El("kn-question-mgr", Attr("contestid", id), Attr("encoded", ui.EncodeJSON(p.AllQuestions))),
					),
				}),
				If(p.CanSubmit, Group{
					Form(ID("question_submit_form"), Class("segment-panel"), AutoComplete("off"),
						H2(ui.T(ctx, "ask_question")),
						Label(
							Span(Class("form-label text-base"), ui.T(ctx, "question_text"), Text(":")),
							Textarea(ID("question_area"), Class("form-textarea w-full my-2")),
						),
						ui.SubmitButton(ui.T(ctx, "button.add")),
					),
					El("kn-questions", Attr("contestid", id), Attr("encoded", ui.EncodeJSON(p.UserQuestions))),
				}),
			),
		),
		Script(Raw(fmt.Sprintf(communicationScript, p.Contest.ID, p.Contest.ID))),
	}
}

const communicationScript = `
    async function createAnnouncement(e) {
        e.preventDefault()
        const data = {text: document.getElementById("announcement_area").value};
        let res = await bundled.postCall("/contest/%d/createAnnouncement", data)
        bundled.apiToast(res);
        if(res.status === "success") {
            bundled.reloadAnnouncements();
            return
        }
    }

    async function askQuestion(e) {
        e.preventDefault()
        const data = {text: document.getElementById("question_area").value};
        let res = await bundled.postCall("/contest/%d/askQuestion", data)
        bundled.apiToast(res);
        if(res.status === "success") {
            bundled.reloadQuestions();
            return
        }
    }

    document.getElementById("announcement_create_form")?.addEventListener("submit", createAnnouncement)
    document.getElementById("question_submit_form")?.addEventListener("submit", askQuestion)
`
