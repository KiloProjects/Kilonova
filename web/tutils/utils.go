// Package tutils holds templ shims over web/ui, for templ views that haven't been
// ported yet. Delete it together with the last .templ file.
package tutils

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/a-h/templ"
	"github.com/shopspring/decimal"
)

type PaginatorConfig = ui.PaginatorConfig

var RemoveTrailingZeros = ui.RemoveTrailingZeros

func T(ctx context.Context, key string, vals ...any) string {
	return ui.TS(ctx, key, vals...)
}

func Paginator(cfg PaginatorConfig) templ.Component {
	return ui.ToTempl(ui.Paginator(cfg))
}

func Score(problem *kilonova.Problem, score *decimal.Decimal, showP ...bool) templ.Component {
	return ui.ToTempl(ui.Score(problem, score, len(showP) > 0 && showP[0]))
}
