package tags

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

var ctx = context.WithValue(context.Background(), util.LangKey, "en")

func TestIndex(t *testing.T) {
	tags := map[kilonova.TagType][]*kilonova.Tag{kilonova.TagTypeMethod: {{ID: 1, Name: `<x>`}}}
	got := uitest.Render(t, Index(ctx, IndexParams{Tags: tags}))
	uitest.Contains(t, got, `data-name="&lt;x&gt;"`, "No tag found")
	uitest.NotContains(t, got, "tagCreateForm")
	uitest.Contains(t, uitest.Render(t, Index(ctx, IndexParams{CanCreate: true})), "tagCreateForm")
}

func TestTag(t *testing.T) {
	tag := &kilonova.Tag{ID: 3, Name: "</script>", Type: kilonova.TagTypeContest}
	t.Run("visitor", func(t *testing.T) {
		got := uitest.Render(t, Tag(ctx, TagParams{Tag: tag, ProblemCount: 5}))
		uitest.Contains(t, got, `tagid="3"`, `href="/problems/random?tag_id=3"`)
		uitest.NotContains(t, got, "tag_update_form", "unsolved=true")
	})
	t.Run("proposer edit form keeps the tag JSON safe; delete only for admins", func(t *testing.T) {
		got := uitest.Render(t, Tag(ctx, TagParams{Tag: tag, Authed: true, CanEdit: true}))
		uitest.Contains(t, got, `</script>`, `<option value="contest" selected>`)
		uitest.NotContains(t, got, "tag_del_btn")
		uitest.Contains(t, uitest.Render(t, Tag(ctx, TagParams{Tag: tag, CanEdit: true, CanDelete: true})), "tag_del_btn")
	})
}
