// Package tags renders the tag index and tag pages.
package tags

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Types are the tag types, in display order.
var Types = []kilonova.TagType{kilonova.TagTypeAuthor, kilonova.TagTypeContest, kilonova.TagTypeMethod, kilonova.TagTypeOther}

// IndexParams holds the tags of each type. CanCreate (proposers) shows the create form.
type IndexParams struct {
	Tags      map[kilonova.TagType][]*kilonova.Tag
	CanCreate bool
}

// Index lists every tag by type, with a client-side search box.
func Index(ctx context.Context, p IndexParams) Node {
	return Group{
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "title.tags")),
			Input(AutoComplete("off"), Attr("autocapitalize", "off"), ID("tag_search_input"), Type("text"), Class("form-input"), Placeholder(ui.TS(ctx, "search_tag"))),
			Div(Class("grid lg:grid-cols-2 xl:grid-cols-4 gap-2"),
				Map(Types, func(t kilonova.TagType) Node { return tagList(ctx, t, p.Tags[t]) }),
			),
		),
		Script(Raw(searchScript)),
		If(p.CanCreate, Group{
			Div(Class("segment-panel"),
				H2(ui.T(ctx, "create_tag")),
				Form(ID("tagCreateForm"), AutoComplete("off"),
					Label(Class("block my-2"),
						Span(Class("form-label"), ui.T(ctx, "name"), Text(":")),
						Input(ID("tagTitle"), Type("text"), Class("form-input"), Required()),
					),
					Label(Class("block my-2"),
						Span(Class("form-label"), ui.T(ctx, "tag_type"), Text(": ")),
						Select(ID("tagType"), Class("form-select"), Required(),
							Option(Value(""), Selected(), Text("-")),
							Map(Types, func(t kilonova.TagType) Node {
								return Option(Value(string(t)), ui.T(ctx, "tag_names."+string(t)))
							}),
						),
					),
					Button(Type("submit"), Class("btn btn-blue mb-2"), ui.T(ctx, "button.create")),
				),
			),
			Script(Raw(createScript)),
		}),
	}
}

func tagList(ctx context.Context, t kilonova.TagType, tags []*kilonova.Tag) Node {
	return Details(Class("segment-panel"), Open(),
		Summary(H2(Class("inline-block mb-2"), ui.T(ctx, "tag_names."+string(t)))),
		Div(Class("mt-1 mb-2"),
			If(len(tags) == 0, ui.T(ctx, "no_tags")),
			Map(tags, func(tag *kilonova.Tag) Node {
				return El("kn-tag", Attr("enc", ui.EncodeJSON(tag)), Data("name", tag.Name))
			}),
		),
	)
}

const searchScript = `
document.getElementById("tag_search_input").addEventListener("input", e => {
	const query = bundled.cleanupTagSearchTerm(e.currentTarget.value);
	document.querySelectorAll("kn-tag").forEach(el => {
		const name = bundled.cleanupTagSearchTerm(el?.dataset?.name ?? "");
		if(name !== "") {
			el.classList.toggle("hidden", !name.includes(query))
		}
	})
})
`

const createScript = `
async function createTag(e) {
	e.preventDefault()

	const data = {
		name: document.getElementById("tagTitle").value,
		type: document.getElementById("tagType").value,
	}
	let res = await bundled.postCall("/tags/create", data)
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}
	window.location.reload()
}

document.getElementById("tagCreateForm").addEventListener("submit", createTag)
`

// TagParams is a tag's page. CanEdit (proposers) shows the update form, CanDelete (admins) the delete button.
type TagParams struct {
	Tag          *kilonova.Tag
	RelevantTags []*kilonova.Tag
	Problems     []*sudoapi.FullProblem
	ProblemCount int
	Authed       bool
	CanEdit      bool
	CanDelete    bool
}

// Tag is a tag's page: its problems, related tags and, for proposers, the edit form.
func Tag(ctx context.Context, p TagParams) Node {
	t := p.Tag
	id := strconv.Itoa(t.ID)
	return Group{
		A(Class("block mb-1"), Href("/tags/"), I(Class("fas fa-arrow-left")), Text(" "), ui.T(ctx, "tags")),
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "tag_header"), Text(" "), El("kn-tag", Attr("enc", ui.EncodeJSON(t)), Attr("link", "false"))),
		),
		If(p.CanEdit, editForm(ctx, p)),
		If(len(p.RelevantTags) > 0, Div(Class("segment-panel"),
			H2(ui.T(ctx, "relevant_tags")),
			Map(p.RelevantTags, func(rt *kilonova.Tag) Node { return El("kn-tag", Attr("enc", ui.EncodeJSON(rt))) }),
		)),
		Div(Class("segment-panel"),
			H2(Textf("%d ", p.ProblemCount), ui.T(ctx, "tag_pbs")),
			El("kn-tag-pbs", Attr("enc", ui.EncodeJSON(p.Problems)), Attr("count", strconv.Itoa(p.ProblemCount)), Attr("tagid", id)),
			If(p.ProblemCount > 2, Div(Class("block my-2"),
				A(Class("btn btn-blue mr-2"), Href("/problems/random?tag_id="+id), ui.T(ctx, "random_problem")),
				If(p.Authed, Group{Text(" "), A(Class("btn btn-blue"), Href("/problems/random?tag_id="+id+"&unsolved=true"), ui.T(ctx, "random_unsolved_problem"))}),
			)),
		),
	}
}

func editForm(ctx context.Context, p TagParams) Node {
	t := p.Tag
	tagJSON, _ := json.Marshal(t) // json escapes <, > and &, so this is safe in a script
	return Div(Class("segment-panel"),
		Details(
			Summary(H2(Class("inline-block"), ui.T(ctx, "update_tag"))),
			Form(ID("tag_update_form"), AutoComplete("off"),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "name"), Text(":")),
					Input(ID("editTagTitle"), Type("text"), Class("form-input"), Value(t.Name), Required()),
				),
				Label(Class("block mb-2"),
					Span(Class("form-label"), ui.T(ctx, "tag_type"), Text(": ")),
					Select(ID("editTagType"), Class("form-select"),
						Map(Types, func(tt kilonova.TagType) Node {
							return Option(Value(string(tt)), If(t.Type == tt, Selected()), ui.T(ctx, "tag_names."+string(tt)))
						}),
					),
				),
				Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
				If(p.CanDelete, Group{
					Text(" "),
					Button(ID("tag_del_btn"), Type("submit"), Class("btn btn-red"), ui.T(ctx, "button.delete")),
					Script(Raw(deleteScript)),
				}),
			),
		),
		Script(Raw("\nconst tag = "+string(tagJSON)+";"+updateScript)),
	)
}

const deleteScript = `
async function deleteTag(e) {
	e.preventDefault()
	if(!(await bundled.confirm(bundled.getText("tagDeleteConfirm")))) {
		return
	}
	let res = await bundled.postCall("/tags/delete", {id: tag.id})
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}

	window.history.back()
}
document.getElementById("tag_del_btn")?.addEventListener("click", deleteTag)
`

const updateScript = `
async function updateTag(e) {
	e.preventDefault()
	const data = {
		id: tag.id,
		name: document.getElementById("editTagTitle").value,
		type: document.getElementById("editTagType").value,
	}

	let res = await bundled.postCall("/tags/update", data)
	if(res.status === "error") {
		bundled.apiToast(res)
		return
	}

	window.location.reload()
}

document.getElementById("tag_update_form")?.addEventListener("submit", updateTag)
`
