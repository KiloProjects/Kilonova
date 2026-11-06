package ui

import (
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	h "maragu.dev/gomponents/html" // not dot-imported: h.Button would collide with Button
)

// Buttons take any children: text, icons, and attributes (h.Class, hx-*, x-*).
// Caller classes are merged into the button's own class attribute.
// type and colour are fixed per function because they can't be merged.

func Button(children ...Node) Node             { return button("button", "btn btn-blue", children) }
func SubmitButton(children ...Node) Node       { return button("submit", "btn btn-blue", children) }
func DangerButton(children ...Node) Node       { return button("button", "btn btn-red", children) }
func DangerSubmitButton(children ...Node) Node { return button("submit", "btn btn-red", children) }

func button(typ, class string, children []Node) Node {
	return h.Button(h.Type(typ), JoinAttrs("class", Group(children), h.Class(class)))
}
