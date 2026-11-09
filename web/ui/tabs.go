package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html"
)

// Tab is one entry of a tab bar. A topbar is declared as a []Tab literal (its
// manifest); visibility comes from permission bits the handler already computed.
type Tab struct {
	Key   string // selected when equal to the bar's current key
	Href  string
	Label Node
	Hide  bool // zero value shows the tab
	Attrs Node // extra attributes on the link (hx-get, ...)
}

// Tabs renders the visible tabs separated by "|", highlighting the one whose Key is current.
func Tabs(current string, tabs ...Tab) Group {
	var out Group
	for _, t := range tabs {
		if t.Hide {
			continue
		}
		if len(out) > 0 {
			out = append(out, Text(" "), TabSeparator(), Text(" "))
		}
		out = append(out, h.A(Classes{"p-1": true, "topbar-selected": t.Key == current}, h.Href(t.Href), t.Attrs, t.Label))
	}
	return out
}

// CollapsedTabs hides tabs behind a toggle link; hovering expands them until the
// pointer leaves the enclosing <nav>, clicking keeps them open. Behavior lives in
// web/assets/topbar.ts.
func CollapsedTabs(toggle Node, open bool, tabs Node) Node {
	return h.Span(h.Data("tabs-collapsible", ""), If(open, h.Data("tabs-open", "")),
		h.A(h.Data("tabs-toggle", ""), Classes{"p-1": true, "hidden": open}, h.Href("#"), toggle),
		Text(" "),
		h.Span(h.Data("tabs-panel", ""), If(!open, h.Class("hidden")), tabs),
	)
}

// TabSeparator is the "|" between tabs.
func TabSeparator() Node { return h.Div(h.Class("topbar-separator")) }

// TabGroupSeparator is the larger "|" between groups of tabs.
func TabGroupSeparator() Node { return h.Div(h.Class("topbar-separator topbar-separator-lg")) }
