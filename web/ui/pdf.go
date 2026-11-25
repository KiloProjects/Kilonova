package ui

import (
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PDFViewer links to the PDF at src and shows it inline with the bundled pdf.js viewer.
// script is the URL of the built web/assets/pdf.ts, which defines <kn-pdf>; being a
// module, it runs once however many viewers end up on a page. Without JavaScript the
// browser's own viewer is embedded instead; <kn-pdf> replaces its children when it starts.
func PDFViewer(src, script, linkText string) Node {
	return Group{
		h.A(h.Class("btn btn-blue"), h.Target("_blank"), h.Href(src), Text(linkText)),
		El("kn-pdf", h.Src(src),
			h.NoScript(h.Embed(h.Class("mx-2 my-2"), h.Type("application/pdf"), h.Src(src),
				h.Style("width: 95%; height: 90vh; background: white; object-fit: contain;"))),
		),
		h.Script(h.Type("module"), h.Src(script)),
	}
}
