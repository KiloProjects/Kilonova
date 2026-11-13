package ui

import (
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestPaginator(t *testing.T) {
	t.Run("collapses distant pages into ellipses", func(t *testing.T) {
		got := uitest.Render(t, Paginator(PaginatorConfig{Page: 10, NumPages: 20}))
		for _, want := range []string{`data-paginate="1"`, `data-paginate="8"`, "paginator-item-active\">10<", `data-paginate="12"`, `data-paginate="20"`} {
			require.Contains(t, got, want)
		}
		require.Equal(t, 2, strings.Count(got, "..."), "expected two ellipses: %s", got)
		require.NotContains(t, got, `data-paginate="6"`)
	})
	t.Run("clamps out-of-range pages", func(t *testing.T) {
		require.Contains(t, uitest.Render(t, Paginator(PaginatorConfig{Page: 99, NumPages: 3})), "paginator-item-active\">3<")
	})
	t.Run("arrows point at first, prev, next, last", func(t *testing.T) {
		got := uitest.Render(t, Paginator(PaginatorConfig{Page: 5, NumPages: 9, ShowArrows: true, ClassName: "mt-2"}))
		require.Contains(t, got, `class="mt-2 paginator"`)
		require.Contains(t, got, `data-paginate="4"><i class="fas fa-angle-left">`)
		require.Contains(t, got, `data-paginate="9"><i class="fas fa-angle-double-right">`)
	})
}
