package ui

import (
	"strconv"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

type PaginatorConfig struct {
	Page     int
	NumPages int

	// Optional
	ContextSize int // pages shown either side of the current one; default 2
	ClassName   string
	ShowArrows  bool
}

// Paginator renders page buttons with data-paginate attributes for the frontend to hook:
// the first and last ContextSize+1 pages, ContextSize pages around the current one,
// and "..." for the gaps.
func Paginator(cfg PaginatorConfig) Node {
	cfg.NumPages = max(cfg.NumPages, 1)
	cfg.Page = min(max(cfg.Page, 1), cfg.NumPages)
	if cfg.ContextSize <= 0 {
		cfg.ContextSize = 2
	}
	page, num, size := cfg.Page, cfg.NumPages, cfg.ContextSize

	var items Group
	pageButton := func(i int) { items = append(items, pageItem(i, Text(strconv.Itoa(i)))) }
	ellipsis := func() { items = append(items, h.Span(h.Class("paginator-item"), Text("..."))) }

	if cfg.ShowArrows {
		items = append(items, pageItem(1, icon("fa-angle-double-left")), pageItem(page-1, icon("fa-angle-left")))
	}
	if page > size+1 {
		for i := 1; i <= 1+size && page-i >= 1+size; i++ {
			pageButton(i)
		}
		if page > 2*(size+1) {
			ellipsis()
		}
	}
	for i := max(1, page-size); i < page; i++ {
		pageButton(i)
	}
	items = append(items, h.Button(h.Class("paginator-item paginator-item-active"), Text(strconv.Itoa(page))))
	for i := page + 1; i <= min(num, page+size); i++ {
		pageButton(i)
	}
	if num-page >= size+1 {
		if num-page > 2*size+1 {
			ellipsis()
		}
		for i := max(num-size, size+page+1); i <= num; i++ {
			pageButton(i)
		}
	}
	if cfg.ShowArrows {
		items = append(items, pageItem(page+1, icon("fa-angle-right")), pageItem(num, icon("fa-angle-double-right")))
	}

	return h.Div(Classes{"paginator": true, cfg.ClassName: cfg.ClassName != ""}, items)
}

func pageItem(page int, content Node) Node {
	return h.Button(h.Class("paginator-item"), h.Data("paginate", strconv.Itoa(page)), content)
}

func icon(class string) Node {
	return h.I(h.Class("fas " + class))
}
