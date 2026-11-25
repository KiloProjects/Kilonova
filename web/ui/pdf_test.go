package ui

import (
	"testing"

	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func TestPDFViewer(t *testing.T) {
	got := uitest.Render(t, PDFViewer(`/assets/a"b.pdf?t=1`, "/static/misc/pdf-abc.js", "Download"))
	uitest.Contains(t, got,
		`<a class="btn btn-blue" target="_blank" href="/assets/a&#34;b.pdf?t=1">Download</a>`,
		`<kn-pdf src="/assets/a&#34;b.pdf?t=1"><noscript><embed class="mx-2 my-2" type="application/pdf" src="/assets/a&#34;b.pdf?t=1"`,
		`</noscript></kn-pdf>`,
		`<script type="module" src="/static/misc/pdf-abc.js"></script>`,
	)
}
