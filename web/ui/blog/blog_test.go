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
	. "maragu.dev/gomponents"
)

var (
	anon   = context.WithValue(context.Background(), util.LangKey, "en")
	author = context.WithValue(anon, user.AuthedUserKey, &kilonova.UserBrief{ID: 2, Name: "bob", Proposer: true})
	other  = context.WithValue(anon, user.AuthedUserKey, &kilonova.UserBrief{ID: 3, Name: "eve"})
	post   = &kilonova.BlogPost{ID: 4, AuthorID: 2, Title: "<T>", Slug: "s", CreatedAt: time.UnixMilli(5)}
)

func TestTopbar(t *testing.T) {
	t.Run("viewer sees only the title", func(t *testing.T) {
		got := render(t, Topbar(anon, TopbarParams{Post: post, Page: "view"}))
		must(t, got, "<b>&lt;T&gt;</b>")
		mustNot(t, got, "/edit")
	})
	t.Run("editor gets tabs with the current one selected", func(t *testing.T) {
		got := render(t, Topbar(anon, TopbarParams{Post: post, Page: "editAttachments", IsPostEditor: true}))
		must(t, got, `<a class="p-1" href="/posts/s">`, `<a class="p-1 topbar-selected" href="/posts/s/edit/attachments">`, "&lt;T&gt; | ")
	})
}

func TestIndex(t *testing.T) {
	hidden := &kilonova.BlogPost{ID: 5, AuthorID: 9, Title: "h", Slug: "h"}
	p := IndexParams{Posts: []*kilonova.BlogPost{post, hidden}, Authors: map[int]*kilonova.UserBrief{2: {Name: "<bob>"}, 9: nil}}
	t.Run("anonymous: no badges or create button, author escaped", func(t *testing.T) {
		got := render(t, Index(anon, p))
		must(t, got, `<a href="/posts/s">&lt;T&gt;</a>`, "&lt;bob&gt;")
		mustNot(t, got, "badge", "/createPost", "<bob>")
	})
	t.Run("author sees own badge only, and the create button", func(t *testing.T) {
		got := render(t, Index(author, p))
		must(t, got, "badge badge-red", "/createPost")
		if strings.Count(got, "badge-") != 1 {
			t.Fatalf("want one badge: %s", got)
		}
	})
	t.Run("non-proposer has no create button", func(t *testing.T) {
		mustNot(t, render(t, Index(other, p)), "/createPost", "badge")
	})
	t.Run("empty", func(t *testing.T) {
		must(t, render(t, Index(anon, IndexParams{})), "No")
	})
}

func TestView(t *testing.T) {
	vs := []*kilonova.StatementVariant{{Language: "en", Format: "md"}, {Language: "ro", Format: "pdf"}}
	got := render(t, View(anon, ViewParams{
		Topbar:      TopbarParams{Post: post, Page: "view"},
		Statement:   "<p>hi</p>",
		Attachments: []*kilonova.Attachment{{ID: 1, Name: "<a>", Size: 10}},
		Variants:    vs, SelectedVariant: &kilonova.StatementVariant{Language: "ro", Format: "pdf"},
	}))
	must(t, got, "<p>hi</p>", `<option value="ro-pdf-" selected>`, `href="/assets/blogPost/s/attachment/&lt;a&gt;"`, `id="att-size-1">10</span>`)
	mustNot(t, got, "created_at", "posted_at", "Created at")

	got = render(t, View(anon, ViewParams{Topbar: TopbarParams{Post: post, Page: "view", IsPostEditor: true}, Variants: vs[:1]}))
	must(t, got, `timestamp="5"`)
	mustNot(t, got, `id="variant-select"`, "grid-cols-1")
}

func TestEditIndex(t *testing.T) {
	evil := &kilonova.BlogPost{ID: 4, Title: "</script>", Slug: "s", Visible: true}
	got := render(t, EditIndex(anon, TopbarParams{Post: evil, Page: "editIndex", IsPostEditor: true},
		editors.StatementEditorParams{Variant: &kilonova.StatementVariant{Language: "en"}, APIPrefix: "/blogPosts/4"}))
	must(t, got, `const post = {"id":4,`, `"title":"\u003c/script\u003e"`, `value="&lt;/script&gt;"`, `id="post_visibility_check" checked>`, `const apiPrefix = "/blogPosts/4";`)
}

func TestCreate(t *testing.T) {
	must(t, render(t, Create(anon)), `id="postCreateForm"`, "/blogPosts/create")
}

func TestEditAttachments(t *testing.T) {
	must(t, render(t, EditAttachments(anon, TopbarParams{Post: post, Page: "editAttachments"}, editors.AttachmentEditorParams{BlogPost: post})), "<b>&lt;T&gt; | ", "attCreate")
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func must(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %s", w, got)
		}
	}
}

func mustNot(t *testing.T, got string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(got, b) {
			t.Fatalf("unexpected %q in %s", b, got)
		}
	}
}
