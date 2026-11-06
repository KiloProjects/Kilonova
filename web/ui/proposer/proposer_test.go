package proposer

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

var base = context.WithValue(context.Background(), util.LangKey, "en")

func TestCreateContestForm(t *testing.T) {
	t.Run("official is disabled for non-admins", func(t *testing.T) {
		must(t, render(t, CreateContestForm(base)), `<option value="official" disabled>`)
	})
	t.Run("admins can pick official", func(t *testing.T) {
		admin := context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 1, Admin: true})
		mustNot(t, render(t, CreateContestForm(admin)), "disabled")
	})
}

func TestCreateProblemList(t *testing.T) {
	t.Run("nested list gets a hidden parent id", func(t *testing.T) {
		must(t, render(t, CreateProblemList(base, &kilonova.ProblemList{ID: 7})), `id="cr_pblist_parent_id" type="hidden" value="7"`)
	})
	t.Run("top-level list asks for a parent id", func(t *testing.T) {
		must(t, render(t, CreateProblemList(base, nil)), `id="cr_pblist_parent_id" type="number"`)
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

func TestPanel(t *testing.T) {
	got := render(t, Panel(context.WithValue(base, user.AuthedUserKey, &kilonova.UserBrief{ID: 5})))
	must(t, got, `href="/problems?editor_user=5"`, `id="problem_create_form"`, `id="problem_import_form"`, `id="cr_pblist_form"`, `id="contest_form"`,
		`<option value="ro" selected>`, `<option value="" selected>`, `title: "No files`)
}
