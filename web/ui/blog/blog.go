// Package blog holds the blog post pages.
package blog

import (
	"context"
	"encoding/json"
	"html"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/editors"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TopbarParams drives the blog post tab bar. Page is "view", "editIndex" or "editAttachments".
type TopbarParams struct {
	IsPostEditor bool
	Post         *kilonova.BlogPost
	Page         string
}

// Topbar renders the post title and, for editors, the view/edit/attachments tabs.
func Topbar(ctx context.Context, p TopbarParams) Node {
	var title Node
	switch p.Page {
	case "view":
		title = B(Text(p.Post.Title))
	case "editIndex":
		title = B(Text(p.Post.Title+" | "), ui.T(ctx, "postEditIndex"))
	case "editAttachments":
		title = B(Text(p.Post.Title+" | "), ui.T(ctx, "postEditAttachments"))
	}
	base := "/posts/" + p.Post.Slug
	return Div(Class("segment-panel"),
		H1(title),
		If(p.IsPostEditor, Div(Class("mt-1"), ui.Tabs(p.Page,
			ui.Tab{Key: "view", Href: base, Label: ui.T(ctx, "post.nav.view")},
			ui.Tab{Key: "editIndex", Href: base + "/edit", Label: ui.T(ctx, "post.nav.edit")},
			ui.Tab{Key: "editAttachments", Href: base + "/edit/attachments", Label: ui.T(ctx, "post.nav.attachments")},
		))),
	)
}

// IndexParams drives the blog post listing.
type IndexParams struct {
	Posts   []*kilonova.BlogPost
	Authors map[int]*kilonova.UserBrief // nil entries for unknown authors

	Pagination Node
}

// Index renders the blog post listing. Publication badges show to admins and each post's author.
func Index(ctx context.Context, p IndexParams) Node {
	authed := user.UserBriefContext(ctx)
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "title.blogPosts")),
		If(authed.IsProposer(), Div(Class("block my-2"),
			A(Class("btn btn-blue"), Href("/createPost"), ui.T(ctx, "createPost")),
		)),
		p.Pagination,
		Map(p.Posts, func(post *kilonova.BlogPost) Node {
			author := p.Authors[post.AuthorID]
			return Div(Class("segment-panel"),
				H2(Class("mb-1"),
					A(Href("/posts/"+post.Slug), Text(post.Title)),
					Text(" "),
					Iff(author != nil, func() Node {
						return Group{ui.TRaw(ctx, "posted_by", html.EscapeString(author.Name)), Text(" ")}
					}),
					If(authed.IsAuthed() && (authed.IsAdmin() || authed.ID == post.AuthorID), Group{
						If(post.Visible, Span(Class("badge badge-green"), ui.T(ctx, "published"))),
						If(!post.Visible, Span(Class("badge badge-red"), ui.T(ctx, "unpublished"))),
					}),
				),
			)
		}),
		If(len(p.Posts) == 0, ui.T(ctx, "noPostFound")),
	)
}

// ViewParams drives the blog post page.
type ViewParams struct {
	Topbar TopbarParams

	Attachments []*kilonova.Attachment // already filtered to what the viewer may see

	Statement       string               // trusted, rendered HTML
	StatementAtt    *kilonova.Attachment // may be nil
	Variants        []*kilonova.StatementVariant
	SelectedVariant *kilonova.StatementVariant
}

// View renders a blog post.
func View(ctx context.Context, p ViewParams) Node {
	post := p.Topbar.Post
	meta := func(key string, ts Node) Node {
		return Span(Class("block text-sm text-muted italic"), ui.T(ctx, key), Text(": "), ts)
	}
	return Group{
		Topbar(ctx, p.Topbar),
		Div(Class("segment-panel reset-list enhance-tables statement-content"),
			Div(Class("text-justify"), Raw(p.Statement)),
			Div(Style("clear: both")),
			Div(Class("block mt-4"),
				If(len(p.Variants) > 1, Label(Class("block mb-1"),
					Span(Class("text-base"), ui.T(ctx, "postFormat"), Text(":")),
					Text(" "),
					Select(ID("variant-select"), Class("form-select-mini"), AutoComplete("off"),
						Map(p.Variants, func(v *kilonova.StatementVariant) Node {
							return Option(Value(v.Language+"-"+v.Format+"-"+v.Type),
								If(p.SelectedVariant != nil && p.SelectedVariant.Equals(v), Selected()),
								Text(ui.StatementVariantName(ctx, v)))
						}),
					),
				)),
				Iff(p.StatementAtt != nil, func() Node { return meta("updated_at", ui.ServerTimestamp(p.StatementAtt.LastUpdatedAt)) }),
				Iff(post.PublishedAt != nil, func() Node { return meta("posted_at", ui.ServerTimestamp(*post.PublishedAt)) }),
				If(p.Topbar.IsPostEditor, meta("created_at", ui.ServerTimestamp(post.CreatedAt))),
			),
		),
		If(len(p.Attachments) > 0, Div(Class("segment-panel"),
			H2(ui.T(ctx, "attachments")),
			Div(Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-1"),
				Map(p.Attachments, func(a *kilonova.Attachment) Node {
					id := strconv.Itoa(a.ID)
					return A(Style("word-wrap: anywhere;"), Href("/assets/blogPost/"+post.Slug+"/attachment/"+a.Name),
						Class("segment-panel m-0 flex justify-between black-anchor"),
						H3(Class("inline"), ID("att-name-"+id), Text(a.Name)),
						Text(" "),
						Span(Class("badge-lite text-sm"), ID("att-size-"+id), Text(strconv.Itoa(a.Size))),
					)
				}),
			),
		)),
		Script(Raw(viewScript)),
	}
}

const viewScript = `
document.getElementById("variant-select")?.addEventListener("change", e => {
    var params = new URLSearchParams(window.location.search);
    params.set("var", e.target.value.replace(/-+$/, ""))
    window.location.search = params.toString()
});
(() => {
    for (let el of document.querySelectorAll("[id^='att-name-']")) {
        el.innerHTML = ` + "`" + `<i class="fas ${bundled.getFileIcon(el.innerText)} fa-fw"></i> ${el.innerHTML}` + "`" + `
    }
    for (let el of document.querySelectorAll("[id^='att-size-']")) {
        el.innerText = bundled.sizeFormatter(Number(el.innerText))
    }
})()
`

// Create renders the new post form.
func Create(ctx context.Context) Node {
	return Group{
		Form(ID("postCreateForm"), Class("segment-panel"), AutoComplete("off"),
			H1(ui.T(ctx, "title.create_post")),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "title"), Text(":")),
				Text(" "),
				Input(ID("postTitle"), Type("text"), Class("form-input"), Required()),
			),
			Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.create")),
		),
		Script(Raw(createScript)),
	}
}

const createScript = `
async function createPost(e) {
    e.preventDefault();
    let title = document.getElementById("postTitle").value;
    let res = await bundled.postCall("/blogPosts/create", {title});
    if(res.status === "error") {
        bundled.apiToast(res)
        return
    }
    window.location.assign(` + "`/posts/${res.data.slug}`" + `)
}

document.getElementById("postCreateForm").addEventListener("submit", createPost)
`

// EditIndex renders the post settings form and the content editor.
func EditIndex(ctx context.Context, topbar TopbarParams, editor editors.StatementEditorParams) Node {
	post := topbar.Post
	// json.Marshal escapes <, > and &, so the post can't close the script.
	postJSON, err := json.Marshal(post)
	if err != nil {
		panic(err)
	}
	return Group{
		Topbar(ctx, topbar),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "header.edit.index")),
			Form(ID("updatePostForm"), AutoComplete("off"),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "title"), Text(":")),
					Text(" "),
					Input(ID("postTitle"), Class("form-input"), Type("text"), Value(post.Title)),
				),
				Label(Class("block my-2"),
					Span(Class("form-label"), ui.T(ctx, "post_slug"), Text(":")),
					Text(" "),
					Input(ID("postSlug"), Class("form-input"), Type("text"), Value(post.Slug)),
				),
				Div(Class("block my-2"),
					Label(Class("inline-flex items-center text-lg"),
						Input(Class("form-checkbox"), Type("checkbox"), ID("post_visibility_check"), If(post.Visible, Checked())),
						Text(" "),
						Span(Class("ml-2"), ui.T(ctx, "visible")),
					),
				),
				Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "button.update")),
			),
		),
		Script(Raw("\n    const post = " + string(postJSON) + ";" + editIndexScript)),
		editors.StatementEditor(ctx, editor),
	}
}

const editIndexScript = `
    async function updatePost(e) {
        e.preventDefault();
        const args = {
            title: document.getElementById("postTitle").value,
            slug: document.getElementById("postSlug").value,
            visible: document.getElementById("post_visibility_check").checked,
        }
        const rez = await bundled.postCall(` + "`/blogPosts/${post.id}/update`" + `, args)
        if(rez.status === "success" && (rez.data.slug != post.slug || args.title != post.title)) {
            window.location.assign(` + "`/posts/${rez.data.slug}/edit`" + `)
        }
        bundled.apiToast({status:rez.status, data: rez.data.message})
    }
    document.getElementById("updatePostForm").addEventListener("submit", updatePost)
`

// EditAttachments renders the post's attachment manager.
func EditAttachments(ctx context.Context, topbar TopbarParams, editor editors.AttachmentEditorParams) Node {
	return Group{Topbar(ctx, topbar), editors.AttachmentEditor(ctx, editor)}
}
