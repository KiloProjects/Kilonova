package tags

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestIndex(t *testing.T) {
	tags := map[kilonova.TagType][]*kilonova.Tag{kilonova.TagTypeMethod: {{ID: 1, Name: `<x>`}}}
	got := render(t, Index(ctx, IndexParams{Tags: tags}))
	must(t, got, `data-name="&lt;x&gt;"`, "No tag found")
	mustNot(t, got, "tagCreateForm")
	must(t, render(t, Index(ctx, IndexParams{CanCreate: true})), "tagCreateForm")
}

func TestTag(t *testing.T) {
	tag := &kilonova.Tag{ID: 3, Name: "</script>", Type: kilonova.TagTypeContest}
	t.Run("visitor", func(t *testing.T) {
		got := render(t, Tag(ctx, TagParams{Tag: tag, ProblemCount: 5}))
		must(t, got, `tagid="3"`, `href="/problems/random?tag_id=3"`)
		mustNot(t, got, "tag_update_form", "unsolved=true")
	})
	t.Run("proposer edit form keeps the tag JSON safe; delete only for admins", func(t *testing.T) {
		got := render(t, Tag(ctx, TagParams{Tag: tag, Authed: true, CanEdit: true}))
		must(t, got, `</script>`, `<option value="contest" selected>`)
		mustNot(t, got, "tag_del_btn")
		must(t, render(t, Tag(ctx, TagParams{Tag: tag, CanEdit: true, CanDelete: true})), "tag_del_btn")
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
	for _, w := range bad {
		if strings.Contains(got, w) {
			t.Fatalf("unexpected %q in %s", w, got)
		}
	}
}
