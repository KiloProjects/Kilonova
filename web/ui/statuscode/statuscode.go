// Package statuscode renders the body of an HTTP error page.
package statuscode

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// View shows code with its standard text, then message, or a generic hint if message is empty.
func View(ctx context.Context, code int, message string) Node {
	return Div(Class("text-center"),
		H2(Textf("%d: %s", code, http.StatusText(code))),
		If(message != "", H1(Text(message))),
		If(message == "", P(ui.T(ctx, "error.message"))),
	)
}
