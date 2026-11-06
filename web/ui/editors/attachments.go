package editors

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// AttachmentEditorParams drives the attachment manager. Exactly one of Problem and BlogPost is set.
type AttachmentEditorParams struct {
	Attachments []*kilonova.Attachment
	Problem     *kilonova.Problem
	BlogPost    *kilonova.BlogPost

	// Updaters maps Attachment.LastUpdatedBy to the user; missing users render as "-".
	Updaters map[int]*kilonova.UserBrief

	APIPrefix string // e.g. /problem/{id} or /blogPosts/{id}, no trailing slash
}

// AttachmentEditor renders the attachment manager.
func AttachmentEditor(ctx context.Context, p AttachmentEditorParams) Node {
	// attName is the <kn-att-name> web component, which links to the attachment.
	attName := func(a *kilonova.Attachment, editable string) Node {
		var owner Node
		switch {
		case p.Problem != nil:
			owner = Attr("pbid", strconv.Itoa(p.Problem.ID))
		case p.BlogPost != nil:
			owner = Group{Attr("postid", strconv.Itoa(p.BlogPost.ID)), Attr("postslug", p.BlogPost.Slug)}
		}
		return El("kn-att-name", owner, Attr("attid", strconv.Itoa(a.ID)), Attr("orgname", a.Name), Attr("editable", editable))
	}
	th := func(class string, children ...Node) Node {
		return Th(Attr("scope", "col"), Class(class), Group(children))
	}
	td := func(children ...Node) Node { return Td(Class("kn-table-cell"), Group(children)) }
	checkbox := func(id string, children ...Node) Node {
		return Input(Class("form-checkbox"), ID(id), Type("checkbox"), Group(children))
	}
	checkLabel := func(labelID, inputID, key string) Node {
		return Label(If(labelID != "", ID(labelID)), Class("block my-2"),
			checkbox(inputID), Text(" "), Span(Class("ml-2 text-xl"), ui.T(ctx, key)))
	}
	fieldLabel := func(key string) Node {
		return Group{Span(Class("mr-2 text-xl"), ui.T(ctx, key), Text(": ")), Text(" ")}
	}
	hasAtts := len(p.Attachments) > 0

	return Group{
		If(p.Problem == nil && p.BlogPost == nil, Div(Class("segment-panel"), Text("ERROR"))),
		script("att_flags", "const apiPrefix = "+js(p.APIPrefix)+";"),
		Div(Class("page-holder"),
			Div(Class("page-content-wrapper"),
				Div(Class("segment-panel"),
					H2(ui.T(ctx, "header.edit.attachments")),
					If(hasAtts, Group{
						Table(Class("kn-table my-2"),
							THead(Tr(
								th("w-1/12", checkbox("selectAllBox", AutoComplete("off"))),
								th("w-1/2", ui.T(ctx, "name")),
								th("w-1/12", ui.T(ctx, "size")),
								th("w-1/12", ui.T(ctx, "visible")),
								th("w-1/12", ui.T(ctx, "private")),
								th("w-1/12", ui.T(ctx, "exec")),
								th("w-1/12"),
							)),
							TBody(ID("att-body"), Map(p.Attachments, func(a *kilonova.Attachment) Node {
								id := strconv.Itoa(a.ID)
								return Tr(ID("row-att-"+id), Class("kn-table-row"),
									td(checkbox("pb-att-"+id, AutoComplete("off"))),
									Td(Class("kn-table-cell"), ID("name-att-"+id), attName(a, "true")),
									Td(Class("kn-table-cell"), ID("size-att-"+id), Text(strconv.Itoa(a.Size))),
									td(checkbox("visible-att-"+id, If(a.Visible, Checked()))),
									td(checkbox("private-att-"+id, If(a.Private, Checked()))),
									td(checkbox("exec-att-"+id, If(a.Exec, Checked()))),
									td(
										Button(Class("btn btn-blue"), Attr("onclick", "toggleEdit("+id+")"), I(Class("fas fa-edit"))),
										If(p.Problem != nil && (a.Name == "statement-ro.pdf" || a.Name == "statement-en.pdf"), Group{
											Text(" "),
											Button(Class("btn btn-blue"), Attr("onclick", "transcribeStatement("+js(a.Name)+")"), Text("Transcribe")),
										}),
									),
								)
							})),
						),
						Div(Class("my-2"),
							Button(Class("btn btn-red mr-2"), Attr("onclick", "deleteAttachments()"), ui.T(ctx, "deleteAttachments")),
							Text(" "),
							Button(Class("btn btn-blue mr-2"), Attr("onclick", "updateAttachments()"), ui.T(ctx, "updateAttachments")),
						),
					}),
					If(!hasAtts, ui.T(ctx, "noAttachments")),
				),
				script("att_table"),
				Div(Class("segment-panel hidden"), ID("editAttView"),
					H2(Class("inline-block mb-2"), ui.T(ctx, "edit_attachment"), Text(" "), Code(ID("ediAttNameView"))),
					Form(ID("editAttForm"), AutoComplete("off"),
						Input(Type("hidden"), ID("editAttID")),
						Input(Type("hidden"), ID("editAttName")),
						Label(Class("block my-2"),
							fieldLabel("contents"),
							Textarea(ID("attEditContent"), Class("hidden")),
						),
						Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
					),
				),
				Details(Class("segment-panel"), ID("addDetails"),
					Summary(H2(Class("inline-block mb-2"), ui.T(ctx, "create_attachment"))),
					Form(ID("addAttForm"), AutoComplete("off"),
						Label(Class("mb-2"),
							fieldLabel("name"),
							Input(Type("text"), Class("form-input w-full"), Placeholder(ui.TS(ctx, "name")+"..."), ID("attCreateName"), Required()),
						),
						Label(Class("block my-2"),
							fieldLabel("contents"),
							Textarea(ID("attCreateContent"), Class("hidden")),
						),
						checkLabel("", "attCreatePrivate", "private"),
						checkLabel("visibleCreateLabel", "attCreateVisible", "visible"),
						checkLabel("", "attCreateExec", "exec"),
						Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.add")),
					),
				),
				script("att_edit"),
				script("att_add"),
				Div(Class("segment-panel"),
					H2(ui.T(ctx, "header.edit.attachment_history")),
					If(hasAtts, Table(Class("kn-table my-2"),
						THead(Tr(
							th("w-1/2", ui.T(ctx, "name")),
							th("w-1/4", ui.T(ctx, "last_updated_at")),
							th("w-1/4", ui.T(ctx, "last_updated_by")),
						)),
						TBody(ID("att-body"), Map(p.Attachments, func(a *kilonova.Attachment) Node {
							return Tr(Class("kn-table-row"),
								td(attName(a, "false")),
								td(ui.ServerTimestamp(a.LastUpdatedAt)),
								td(updater(p.Updaters, a.LastUpdatedBy)),
							)
						})),
					)),
					If(!hasAtts, ui.T(ctx, "noAttachments")),
				),
			),
			Aside(Class("page-sidebar"),
				Form(Class("segment-panel"), ID("attCreate"), AutoComplete("off"),
					H2(ui.T(ctx, "header.edit.attachment_create")),
					Label(Class("block my-2"),
						fieldLabel("file"),
						Input(Class("form-input"), ID("attFile"), Type("file"), Required(), Multiple()),
					),
					Label(ID("attUploadNameLabel"), Class("block my-2"),
						fieldLabel("name"),
						Input(Class("form-input"), ID("attUploadName"), Type("text")),
					),
					checkLabel("", "attUploadPrivate", "private"),
					checkLabel("visibleUploadLabel", "attUploadVisible", "visible"),
					checkLabel("", "attUploadExec", "exec"),
					Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.upload")),
				),
				script("att_upload"),
			),
		),
	}
}

func updater(users map[int]*kilonova.UserBrief, id *int) Node {
	if id == nil || users[*id] == nil {
		return Text("-")
	}
	u := users[*id]
	return A(Href("/profile/"+u.Name),
		Img(Class("rounded-sm inline-block mr-2"), Src("/profile/"+u.Name+"/avatar?s=32"), Width("32"), Height("32"), Alt("Avatar")),
		Text(" "+u.Name),
	)
}
