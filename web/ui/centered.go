package ui

import (
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CenteredLayout centres a 500px-wide column on the page, for small forms like login.
func CenteredLayout(children ...Node) Node {
	return h.Div(h.Class("flex items-center justify-center min-h-[70vh]"),
		h.Div(h.Style("width: 500px;"), Group(children)),
	)
}
