// Package editors holds the markdown statement editor and the attachment manager,
// shared by the problem and blog post edit pages.
package editors

import (
	"context"
	"embed"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

//go:embed js/*.js
var scripts embed.FS

// script renders an embedded js/<name>.js as an inline <script>, prefixed by pre
// (Go values already JSON-encoded with js).
func script(name string, pre ...string) Node {
	body, err := scripts.ReadFile("js/" + name + ".js")
	if err != nil {
		panic(err)
	}
	return Script(Raw(strings.Join(pre, "") + "\n" + string(body)))
}

// js JSON-encodes v for a JS expression context. json.Marshal escapes <, > and &,
// so the result can't close the surrounding <script>.
func js(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// StatementEditorParams drives the markdown editor for problem statements and blog posts.
type StatementEditorParams struct {
	Variants []*kilonova.StatementVariant // existing variants
	Variant  *kilonova.StatementVariant   // existing or new markdown variant
	Data     string                       // contents of Att
	Att      *kilonova.Attachment         // nil if the variant doesn't exist yet

	APIPrefix string // e.g. /problem/{id} or /blogPosts/{id}, no trailing slash
}

// StatementEditor renders the markdown statement editor.
func StatementEditor(ctx context.Context, p StatementEditorParams) Node {
	attID := "-1"
	if p.Att != nil {
		attID = strconv.Itoa(p.Att.ID)
	}
	langOption := func(lang, label string) Node {
		return Option(Value(lang), If(p.Variant.Language == lang, Selected()), Text(label))
	}
	return Group{
		Iff(len(p.Variants) > 0, func() Node { return variantList(ctx, p.Variants) }),
		Div(Class("page-holder"),
			Form(ID("statementUpdForm"), Class("page-content-full"), AutoComplete("off"),
				H2(ui.T(ctx, "header.edit.description")),
				Label(Class("block mb-2"),
					Span(Class("form-label"), ui.T(ctx, "spokenLanguage"), Text(":")),
					Text(" "),
					Select(ID("variantSelect"), Class("form-select"), AutoComplete("off"),
						langOption("ro", "🇷🇴 Română"),
						langOption("en", "🇬🇧 English"),
						langOption("de", "🇩🇪 Deutsch"),
					),
				),
				Input(Type("hidden"), ID("attID"), Value(attID)),
				Span(ID("selectedVariant"), Class("block mb-2")),
				Input(Type("hidden"), ID("variantType"), Value("")),
				Div(Class("block mb-2"),
					Label(Class("inline-flex items-center text-lg"),
						Input(Class("form-checkbox"), Name("private"), ID("private_checkbox"), Type("checkbox"), If(p.Att != nil && p.Att.Private, Checked())),
						Text(" "),
						Span(Class("ml-2"), ui.T(ctx, "private")),
					),
				),
				Div(Class("mb-2"),
					Textarea(ID("description"), Class("hidden"), AutoFocus(), Text(p.Data)),
				),
				Div(Class("mb-1"),
					Button(Class("btn btn-blue"), ui.T(ctx, "button.update")),
					Span(Class("block mt-2 text-muted text-sm"), ui.T(ctx, "hint.edit.description_save")),
				),
			),
			Details(Class("page-content-full"),
				Summary(H2(Class("inline-block"), ui.T(ctx, "header.edit.advanced_settings"))),
				Div(Class("block mb-2"),
					Label(Class("inline-flex items-center text-sm"),
						Input(ID("autoformat_paste"), Class("form-checkbox"), Type("checkbox")),
						Text(" "),
						Span(Class("form-label ml-2"), ui.T(ctx, "edit.option.autoformat_paste")),
					),
				),
				Div(Class("block mb-2"),
					Button(ID("formatDocumentButton"), Class("btn btn-blue"), ui.T(ctx, "edit.command.format_document")),
				),
			),
		),
		script("statement_unload"),
		script("statement_latex"),
		script("statement",
			"const apiPrefix = "+js(p.APIPrefix)+";\n",
			"const initialVariantType = "+js(p.Variant.Type)+";\n",
			"var currentLang = "+js(p.Variant.Language)+";\n",
		),
	}
}

func variantList(ctx context.Context, variants []*kilonova.StatementVariant) Node {
	var md []*kilonova.StatementVariant
	for _, v := range variants {
		if v.Format == "md" {
			md = append(md, v)
		}
	}
	return Group{
		Div(Class("segment-panel reset-list"), ID("variantDiv"),
			ui.T(ctx, "existingVariants"), Text(":"),
			Ul(ID("variantList"), Map(md, func(v *kilonova.StatementVariant) Node {
				return Li(A(
					Href("?pref_lang="+url.QueryEscape(v.Language)+"&pref_type="+url.QueryEscape(v.Type)), Data("type", v.Type), Data("lang", v.Language),
					Text(ui.StatementVariantName(ctx, v)),
					If(v.Private, Group{Text(" ("), ui.T(ctx, "private"), Text(")")}),
				))
			})),
		),
		// TODO: Remove this hack
		If(len(md) < 2, StyleEl(Raw("#variantDiv {display: none}"))),
	}
}
