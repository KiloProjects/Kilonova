// Package resources renders a problem's external resources (editorials, videos, ...).
package resources

import (
	"context"
	"strconv"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// TODO stands in for the resource listing and resource page, which aren't built yet.
func TODO(_ context.Context) Node {
	return H1(Text("TODO"))
}

// NewForm creates an external resource for problem problemID; only admins may pre-approve it.
func NewForm(ctx context.Context, problemID int) Node {
	field := func(key string, input Node) Node {
		return Label(Class("block my-2"), Span(Class("form-label"), ui.T(ctx, key)), Text(" "), input)
	}
	opt := func(typ string) Node {
		return Option(Value(typ), ui.T(ctx, "external_resource.type."+typ))
	}
	return Form(Class("segment-panel"), ID("externalResourceForm"), Attr("hx-post", "/problems/"+strconv.Itoa(problemID)+"/externalResources/create"),
		H1(ui.T(ctx, "header.external_resources.new")),
		field("name", Input(Class("form-input"), Type("text"), Name("name"), Required())),
		field("description", Input(Class("form-input"), Type("text"), Name("description"))),
		field("url", Input(Class("form-input"), Type("url"), Name("url"), Required())),
		field("external_resource.type.label", Select(Class("form-select"), Name("type"),
			opt("editorial"), opt("video"), opt("solution"), opt("article"), opt("other"),
		)),
		If(user.UserBriefContext(ctx).IsAdmin(), Div(Class("block my-2"),
			Label(Class("inline-flex items-center text-lg"),
				Input(Class("form-checkbox"), Name("preApproved"), Type("checkbox"), Checked()), Text(" "),
				Span(Class("ml-2"), ui.T(ctx, "preApprovedResource")),
			),
		)),
		ui.SubmitButton(ui.T(ctx, "button.create")),
	)
}
