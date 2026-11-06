package ui

import (
	"strings"
	"testing"
)

func TestPaginator(t *testing.T) {
	t.Run("collapses distant pages into ellipses", func(t *testing.T) {
		got := render(t, Paginator(PaginatorConfig{Page: 10, NumPages: 20}))
		for _, want := range []string{`data-paginate="1"`, `data-paginate="8"`, "paginator-item-active\">10<", `data-paginate="12"`, `data-paginate="20"`} {
			if !strings.Contains(got, want) {
				t.Fatalf("missing %s in %s", want, got)
			}
		}
		if strings.Count(got, "...") != 2 || strings.Contains(got, `data-paginate="6"`) {
			t.Fatalf("expected two ellipses and no page 6: %s", got)
		}
	})
	t.Run("clamps out-of-range pages", func(t *testing.T) {
		if got := render(t, Paginator(PaginatorConfig{Page: 99, NumPages: 3})); !strings.Contains(got, "paginator-item-active\">3<") {
			t.Fatalf("got %s", got)
		}
	})
	t.Run("arrows point at first, prev, next, last", func(t *testing.T) {
		got := render(t, Paginator(PaginatorConfig{Page: 5, NumPages: 9, ShowArrows: true, ClassName: "mt-2"}))
		if !strings.Contains(got, `class="mt-2 paginator"`) || !strings.Contains(got, `data-paginate="4"><i class="fas fa-angle-left">`) || !strings.Contains(got, `data-paginate="9"><i class="fas fa-angle-double-right">`) {
			t.Fatalf("got %s", got)
		}
	})
}
