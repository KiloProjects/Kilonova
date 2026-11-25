package ui

import (
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PDFViewer links to the PDF at src and shows it inline with the bundled pdf.js viewer.
// script is the URL of the built web/assets/pdf.ts, which defines <kn-pdf>; being a
// module, it runs once however many viewers end up on a page. Without JavaScript the
// browser's own viewer is embedded instead; <kn-pdf> replaces its children when it starts.
//
// The same embed is the fallback when the script cannot be fetched (onerror) or runs
// without defining the element, as when it throws in an old browser (onload). With
// scripting on, a <noscript> holds its content as text, so the handler parses it.
func PDFViewer(src, script, linkText string) Node {
	const fallback = `for (const n of document.querySelectorAll("kn-pdf > noscript")) n.parentNode.innerHTML = n.textContent`
	return Group{
		h.A(h.Class("btn btn-blue"), h.Target("_blank"), h.Href(src), Text(linkText)),
		El("kn-pdf", h.Src(src),
			h.NoScript(h.Embed(h.Class("mx-2 my-2"), h.Type("application/pdf"), h.Src(src),
				h.Style("width: 95%; height: 90vh; background: white; object-fit: contain;"))),
		),
		h.Script(h.Type("module"), h.Src(script),
			Attr("onerror", fallback), Attr("onload", `customElements.get("kn-pdf") || this.onerror()`)),
	}
}
