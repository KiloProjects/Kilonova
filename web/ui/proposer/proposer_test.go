package proposer

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

var base = context.WithValue(context.Background(), util.LangKey, "en")

func TestCreateContestForm(t *testing.T) {
	t.Run("official is disabled for non-admins", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, CreateContestForm(base)), `<option value="official" disabled>`)
	})
	t.Run("admins can pick official", func(t *testing.T) {
		admin := context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 1, Admin: true})
		uitest.NotContains(t, uitest.Render(t, CreateContestForm(admin)), "disabled")
	})
}

func TestCreateProblemList(t *testing.T) {
	t.Run("nested list gets a hidden parent id", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, CreateProblemList(base, &kilonova.ProblemList{ID: 7})), `id="cr_pblist_parent_id" type="hidden" value="7"`)
	})
	t.Run("top-level list asks for a parent id", func(t *testing.T) {
		uitest.Contains(t, uitest.Render(t, CreateProblemList(base, nil)), `id="cr_pblist_parent_id" type="number"`)
	})
}

func TestPanel(t *testing.T) {
	got := uitest.Render(t, Panel(context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 5})))
	uitest.Contains(t, got, `href="/problems?editor_user=5"`, `id="problem_create_form"`, `id="problem_import_form"`, `id="cr_pblist_form"`, `id="contest_form"`,
		`<option value="ro" selected>`, `<option value="" selected>`, `title: "No files`)
}
