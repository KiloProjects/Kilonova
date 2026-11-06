package contests

import (
	"context"

	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/proposer"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// TabsParams drives the contest index tab bar.
type TabsParams struct {
	Page      string // URL path of the current tab, or "create"
	Authed    bool
	CanCreate bool // shown only when Authed
}

// IndexParams is the contest list page.
type IndexParams struct {
	TabsParams
	Contests   []Info
	Pagination Node // nil when there is a single page
}

// Index lists contests under the tab bar.
func Index(ctx context.Context, p IndexParams) Node {
	return Group{
		Tabs(ctx, p.TabsParams),
		p.Pagination,
		Div(Class("contest-grid"),
			Map(p.Contests, func(c Info) Node { return Brief(ctx, c) }),
			If(len(p.Contests) == 0, Div(Class("segment-panel col-span-full"),
				P(ui.T(ctx, "noContests")),
			)),
		),
	}
}

// Create is the contest creation page.
func Create(ctx context.Context, p TabsParams) Node {
	return Group{
		Tabs(ctx, p),
		Div(Class("segment-panel"), proposer.CreateContestForm(ctx)),
	}
}

// Tabs is the contest index tab bar.
func Tabs(ctx context.Context, p TabsParams) Node {
	tab := func(page, href, key string) Node {
		return A(Classes{"p-1": true, "topbar-selected": p.Page == page}, Href(href), ui.T(ctx, key))
	}
	sep := Text(" | ")
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "contests")),
		Div(Class("mt-1"),
			tab("/contests", "/contests", "contest_index.all"), sep,
			tab("/contests/official", "/contests/official", "contest_index.official"), sep,
			tab("/contests/virtual", "/contests/virtual", "contest_index.virtual"),
			If(p.Authed, Group{
				sep, tab("/contests/personal", "/contests/personal", "contest_index.personal"),
				If(p.CanCreate, Group{sep, tab("create", "/contests/create", "contest_index.create")}),
			}),
		),
	)
}
