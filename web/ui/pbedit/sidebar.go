package pbedit

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type sidebarItem struct{ ID, VisibleID int }

type sidebarParams struct {
	page, path, panelClass  string
	headKey, addKey, nthKey string
	items                   []sidebarItem
}

// sidebar is the test/subtask navigation: the index link, the "create" link, then one link per item.
func sidebar(ctx context.Context, t *problems.TopbarParams, s sidebarParams) Node {
	base := t.URLPrefix + "/problems/" + strconv.Itoa(t.Problem.ID) + "/edit/" + s.path
	return Aside(Class("page-sidebar lg:order-last"),
		Div(Class(s.panelClass),
			Div(Class("list-group grid grid-cols-2 xl:grid-cols-3"),
				A(Class(selectedClass("list-group-head col-span-full", t.Page == s.page && t.PageID == -2)), Href(base),
					H2(ui.T(ctx, s.headKey)),
				),
				A(Class(selectedClass("list-group-item col-span-full", t.PageID == -1)), Href(base+"/add"), ui.T(ctx, s.addKey)),
				Map(s.items, func(it sidebarItem) Node {
					return A(Class(selectedClass("list-group-item", t.PageID == it.ID)), Href(base+"/"+strconv.Itoa(it.ID)), ui.T(ctx, s.nthKey, it.VisibleID))
				}),
			),
		),
	)
}
