package blog

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/editors"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

var (
	anon   = context.WithValue(context.Background(), util.LangKey, "en")
	author = context.WithValue(anon, user.AuthedUserKey, &kilonova.UserBrief{ID: 2, Name: "bob", Proposer: true})
	other  = context.WithValue(anon, user.AuthedUserKey, &kilonova.UserBrief{ID: 3, Name: "eve"})
	post   = &kilonova.BlogPost{ID: 4, AuthorID: 2, Title: "<T>", Slug: "s", CreatedAt: time.UnixMilli(5)}
)

func TestTopbar(t *testing.T) {
	t.Run("viewer sees only the title", func(t *testing.T) {
		got := uitest.Render(t, Topbar(anon, TopbarParams{Post: post, Page: "view"}))
		uitest.Contains(t, got, "<b>&lt;T&gt;</b>")
		uitest.NotContains(t, got, "/edit")
	})
	t.Run("editor gets tabs with the current one selected", func(t *testing.T) {
		got := uitest.Render(t, Topbar(anon, TopbarParams{Post: post, Page: "editAttachments", IsPostEditor: true}))
		uitest.Contains(t, got, `<a class="p-1" href="/posts/s">`, `<a class="p-1 topbar-selected" href="/posts/s/edit/attachments">`, "&lt;T&gt; | ")
	})
}

func TestIndex(t *testing.T) {
	hidden := &kilonova.BlogPost{ID: 5, AuthorID: 9, Title: "h", Slug: "h"}
	p := IndexParams{Posts: []*kilonova.BlogPost{post, hidden}, Authors: map[int]*kilonova.UserBrief{2: {Name: "<bob>"}, 9: nil}}
	t.Run("anonymous: no badges or create button, author escaped", func(t *testing.T) {
		got := uitest.Render(t, Index(anon, p))
		uitest.Contains(t, got, `<a href="/posts/s">&lt;T&gt;</a>`, "&lt;bob&gt;")
		uitest.NotContains(t, got, "badge", "/createPost", "<bob>")
	})
	t.Run("author sees own badge only, and the create button", func(t *testing.T) {
		got := uitest.Render(t, Index(author, p))
		uitest.Contains(t, got, "badge badge-red", "/createPost")
		require.Equal(t, 1, strings.Count(got, "badge-"), "want one badge: %s", got)
	})
	t.Run("non-proposer has no create button", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, Index(other, p)), "/createPost", "badge")
	})
	t.Run("empty", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, Index(anon, IndexParams{})), "No")
	})
}

func TestView(t *testing.T) {
	vs := []*kilonova.StatementVariant{{Language: "en", Format: "md"}, {Language: "ro", Format: "pdf"}}
	got := uitest.Render(t, View(anon, ViewParams{
		Topbar:      TopbarParams{Post: post, Page: "view"},
		Statement:   "<p>hi</p>",
		Attachments: []*kilonova.Attachment{{ID: 1, Name: "<a>", Size: 10}},
		Variants:    vs, SelectedVariant: &kilonova.StatementVariant{Language: "ro", Format: "pdf"},
	}))
	uitest.Contains(t, got, "<p>hi</p>", `<option value="ro-pdf-" selected>`, `href="/assets/blogPost/s/attachment/&lt;a&gt;"`, `id="att-size-1">10</span>`)
	uitest.NotContains(t, got, "created_at", "posted_at", "Created at")

	got = uitest.Render(t, View(anon, ViewParams{Topbar: TopbarParams{Post: post, Page: "view", IsPostEditor: true}, Variants: vs[:1]}))
	uitest.Contains(t, got, `timestamp="5"`)
	uitest.NotContains(t, got, `id="variant-select"`, "grid-cols-1")
}

func TestEditIndex(t *testing.T) {
	evil := &kilonova.BlogPost{ID: 4, Title: "</script>", Slug: "s", Visible: true}
	got := uitest.Render(t, EditIndex(anon, TopbarParams{Post: evil, Page: "editIndex", IsPostEditor: true},
		editors.StatementEditorParams{Variant: &kilonova.StatementVariant{Language: "en"}, APIPrefix: "/blogPosts/4"}))
	uitest.Contains(t, got, `const post = {"id":4,`, `"title":"\u003c/script\u003e"`, `value="&lt;/script&gt;"`, `id="post_visibility_check" checked>`, `const apiPrefix = "/blogPosts/4";`)
}

func TestCreate(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, Create(anon)), `id="postCreateForm"`, "/blogPosts/create")
}

func TestEditAttachments(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, EditAttachments(anon, TopbarParams{Post: post, Page: "editAttachments"}, editors.AttachmentEditorParams{BlogPost: post})), "<b>&lt;T&gt; | ", "attCreate")
}
