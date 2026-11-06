package editors

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

func TestStatementEditor(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	t.Run("new variant", func(t *testing.T) {
		got := render(t, StatementEditor(ctx, StatementEditorParams{
			Variant:   &kilonova.StatementVariant{Language: "en", Format: "md"},
			APIPrefix: "/problem/3",
		}))
		must(t, got, `id="attID" value="-1"`, `<option value="en" selected>`, `const apiPrefix = "/problem/3";`, `id="private_checkbox" type="checkbox">`)
		mustNot(t, got, "variantDiv")
	})
	t.Run("script values are JSON and can't break out of the script", func(t *testing.T) {
		got := render(t, StatementEditor(ctx, StatementEditorParams{
			Variant:   &kilonova.StatementVariant{Language: `"</script><b>`, Type: "x\"y", Format: "md"},
			APIPrefix: "/p",
		}))
		must(t, got, `var currentLang = "\"`+"\\u003c/script\\u003e\\u003cb\\u003e"+`";`, `const initialVariantType = "x\"y";`)
		mustNot(t, got, "</script><b>")
	})
	t.Run("existing private attachment, data escaped, variant list hidden with one md variant", func(t *testing.T) {
		vs := []*kilonova.StatementVariant{{Language: "ro", Format: "md", Type: "short", Private: true}, {Language: "en", Format: "pdf"}}
		got := render(t, StatementEditor(ctx, StatementEditorParams{
			Variants: vs, Variant: vs[0], Data: "</textarea>",
			Att: &kilonova.Attachment{ID: 9, Private: true}, APIPrefix: "/p",
		}))
		must(t, got, `id="attID" value="9"`, `id="private_checkbox" type="checkbox" checked>`, "&lt;/textarea&gt;",
			`href="?pref_lang=ro&amp;pref_type=short" data-type="short" data-lang="ro"`, "#variantDiv {display: none}", " (Private)")
		mustNot(t, got, "PDF")
	})
}

func TestAttachmentEditor(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	uid, missing := 5, 6
	atts := []*kilonova.Attachment{
		{ID: 1, Name: "statement-ro.pdf", Visible: true, LastUpdatedAt: time.UnixMilli(1000), LastUpdatedBy: &uid},
		{ID: 2, Name: `a"<b>`, Exec: true, LastUpdatedBy: &missing},
	}
	users := map[int]*kilonova.UserBrief{5: {ID: 5, Name: "alice"}}
	t.Run("problem", func(t *testing.T) {
		got := render(t, AttachmentEditor(ctx, AttachmentEditorParams{Attachments: atts, Problem: &kilonova.Problem{ID: 7}, Updaters: users, APIPrefix: "/problem/7"}))
		must(t, got, `<kn-att-name pbid="7" attid="1" orgname="statement-ro.pdf" editable="true">`,
			`onclick="transcribeStatement(&#34;statement-ro.pdf&#34;)"`, `onclick="toggleEdit(2)"`,
			`orgname="a&#34;&lt;b&gt;"`, `id="visible-att-1" type="checkbox" checked>`,
			`<a href="/profile/alice">`, `<td class="kn-table-cell">-</td>`, `timestamp="1000"`,
			`const apiPrefix = "/problem/7";`)
		mustNot(t, got, "ERROR", "postid", "noAttachments")
	})
	t.Run("blog post has no transcribe button", func(t *testing.T) {
		got := render(t, AttachmentEditor(ctx, AttachmentEditorParams{Attachments: atts, BlogPost: &kilonova.BlogPost{ID: 4, Slug: "s"}}))
		must(t, got, `postid="4" postslug="s"`)
		mustNot(t, got, "Transcribe", "pbid")
	})
	t.Run("empty", func(t *testing.T) {
		got := render(t, AttachmentEditor(ctx, AttachmentEditorParams{}))
		must(t, got, "ERROR")
		mustNot(t, got, `id="row-att-`, "kn-table my-2")
	})
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
