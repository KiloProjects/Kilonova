package ui

import (
	"github.com/KiloProjects/kilonova"
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html" // package ui declares Button, so html can't be dot-imported in any file here
)

// NoRobots asks search engines not to index or follow the page.
func NoRobots() Node {
	return h.Meta(h.Name("robots"), h.Content("none"))
}

// CanonicalURL points search engines at the canonical address for path.
func CanonicalURL(path string) Node {
	return h.Link(h.Rel("canonical"), h.Href(kilonova.HostURL().JoinPath(path).String()))
}
