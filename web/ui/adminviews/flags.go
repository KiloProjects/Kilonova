package adminviews

import (
	"context"
	_ "embed"
	"strconv"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/flashes"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

var (
	//go:embed scripts/flags_transfer.js
	flagsTransferJS string
	//go:embed scripts/flashes.js
	flashesJS string
)

// Flag input kinds. Every flag input carries data-flag (its key) and data-type (one of
// these); scripts/flags.js builds the PATCH /api/admin/v2/flags body from them.
const (
	kindBool     = "bool"
	kindString   = "string"
	kindInt      = "int"
	kindIntList  = "int_array"
	kindStrList  = "str_array"
	flagsFormKey = "flags-form" // data attribute marking a form flags.js submits
)

type namedFlag interface {
	InternalName() string
	HumanName() string
}

func boolField(ctx context.Context, f config.Flag[bool]) Node {
	v := f.Get(ctx)
	return Div(Class("block"),
		Label(Class("inline-flex items-center text-lg"),
			Input(Class("form-checkbox"), Type("checkbox"), flagAttrs(f, kindBool, strconv.FormatBool(v)), If(v, Checked()), AutoComplete("off")), Text(" "),
			Span(Class("ml-2"), Text(f.HumanName())), Text(" "),
			Span(Class("ml-2 text-sm text-muted"), Text("("+f.InternalName()+")")),
		),
	)
}

func valueField(f namedFlag, kind, value, inputType string) Node {
	return Label(Class("block my-2"),
		Span(Text(f.HumanName())), Text(" "),
		Span(Class("mr-2 text-sm text-muted"), Text("("+f.InternalName()+")")), Text(" "),
		Input(Class("form-input w-full"), Type(inputType), flagAttrs(f, kind, value), Value(value), AutoComplete("off")),
	)
}

func flagAttrs(f namedFlag, kind, value string) Node {
	return Group{Data("flag", f.InternalName()), Data("type", kind), Data("org_value", value)}
}

func stringField(ctx context.Context, f config.Flag[string]) Node {
	return valueField(f, kindString, f.Get(ctx), "text")
}

func intField(ctx context.Context, f config.Flag[int]) Node {
	return valueField(f, kindInt, strconv.Itoa(f.Get(ctx)), "number")
}

func intListField(ctx context.Context, f config.Flag[[]int]) Node {
	return valueField(f, kindIntList, jsonList(f.Get(ctx)), "text")
}

func strListField(ctx context.Context, f config.Flag[[]string]) Node {
	return valueField(f, kindStrList, jsonList(f.Get(ctx)), "text")
}

// flagField renders the input for the flag named key, whatever its type. It panics on
// an unknown key: groups are compile-time data, and the coverage test renders them all.
func flagField(ctx context.Context, key string) Node {
	if f, ok := config.GetFlag[bool](key); ok {
		return boolField(ctx, f)
	}
	if f, ok := config.GetFlag[string](key); ok {
		return stringField(ctx, f)
	}
	if f, ok := config.GetFlag[int](key); ok {
		return intField(ctx, f)
	}
	if f, ok := config.GetFlag[[]int](key); ok {
		return intListField(ctx, f)
	}
	if f, ok := config.GetFlag[[]string](key); ok {
		return strListField(ctx, f)
	}
	panic("adminviews: no flag named " + key)
}

// flatFlags is the fallback editor listing every flag by type, collapsed by default.
// Secret flags are pulled into their own nested, also collapsed, section.
func flatFlags(ctx context.Context, params PanelParams) Node {
	var secrets []string
	section := func(key string, names []string) Node {
		return flagSection(ctx, key, len(names), fields(ctx, names))
	}
	return Details(ID("flatFlags"), Class("segment-panel"),
		Summary(H2(Class("inline-block text-2xl"), ui.T(ctx, "config.title"))),
		Form(ID("flagsForm"), Data(flagsFormKey, ""), AutoComplete("off"),
			section("boolFlags", splitSecrets(params.BoolFlags, &secrets)),
			section("stringFlags", splitSecrets(params.StringFlags, &secrets)),
			section("intFlags", splitSecrets(params.IntFlags, &secrets)),
			section("intArrayFlags", splitSecrets(params.IntArrayFlags, &secrets)),
			section("strArrayFlags", splitSecrets(params.StrArrayFlags, &secrets)),
			Details(ID("secretFlags"), Class("segment-panel"),
				Summary(H3(Class("inline-block"), ui.T(ctx, "config.secret_flags"))),
				fields(ctx, secrets),
			),
			Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
		),
	)
}

// splitSecrets returns the names of the non-secret flags in fl and appends the secret ones to secrets.
func splitSecrets[T any](fl []config.Flag[T], secrets *[]string) []string {
	var public []string
	for _, f := range fl {
		if f.Secret() {
			*secrets = append(*secrets, f.InternalName())
		} else {
			public = append(public, f.InternalName())
		}
	}
	return public
}

func fields(ctx context.Context, names []string) Node {
	return Map(names, func(name string) Node { return flagField(ctx, name) })
}

// flagsTransfer holds the export (with an opt-in for secrets) and import controls.
func flagsTransfer(ctx context.Context) Node {
	return Div(Class("segment-panel"),
		H2(Class("text-2xl"), ui.T(ctx, "flags_transfer.title")),
		Div(Class("my-2"),
			Label(Class("inline-flex items-center"),
				Input(Class("form-checkbox"), Type("checkbox"), ID("exportSecrets"), AutoComplete("off")),
				Span(Class("ml-2"), ui.T(ctx, "flags_transfer.include_secrets")),
			), Text(" "),
			Button(Type("button"), ID("exportFlags"), Class("btn btn-blue ml-2"), ui.T(ctx, "flags_transfer.export")),
		),
		Div(Class("my-2"),
			Input(Type("file"), ID("importFile"), Accept("application/json,.json"), Class("form-input")), Text(" "),
			Button(Type("button"), ID("importFlags"), Class("btn btn-blue ml-2"), ui.T(ctx, "flags_transfer.import")),
		),
		Div(ID("importResult"), Class("my-2")),
		Script(Raw(flagsTransferJS)),
	)
}

// settingsGroup is one section of the grouped settings page: a title and the flags it edits.
type settingsGroup struct {
	titleKey string
	flags    []string
}

// settingsGroups lay flags out by what they control. A flag belongs to at most one group;
// TestEveryFlagHasAHome keeps the rest on a shrinking waiting list.
var settingsGroups = []settingsGroup{
	{"settings.group.front_page", []string{
		"frontend.front_page_announcement",
		"frontend.front_page.root_problem_list",
		"frontend.front_page.pinned_problem_list",
		"frontend.front_page.show_trending",
		"frontend.banned_hot_problems",
		"feature.frontend.front_page_pbs",
		"feature.frontend.front_page_pbs_links",
		"feature.frontend.front_page_latest_pbs",
		"feature.frontend.front_page_random_pb",
		"feature.frontend.front_page_csidebar",
		"feature.frontend.main_page_login",
		"feature.frontend.quick_search",
	}},
	{"settings.group.navbar_footer", []string{
		"frontend.navbar.branding",
		"feature.frontend.navbar.problems_btn",
		"feature.frontend.navbar.contests_btn",
		"feature.frontend.navbar.submissions_btn",
		"frontend.footer.discord_id",
		"frontend.footer.cc_disclaimer",
		"feature.frontend.footer.time_statistics",
	}},
	{"settings.group.donations", []string{
		"frontend.donations.enabled",
		"frontend.donation.frontpage_nag",
		"frontend.donation.bmac_name",
		"frontend.donation.paypal_btn_id",
		"frontend.donation.stripe_button_id",
		"frontend.donation.stripe_payment_link",
		"frontend.donation.stripe_publishable_key",
	}},
}

// Settings is the grouped flags editor.
func Settings(ctx context.Context) Node {
	return Group{
		H1(ui.T(ctx, "settings.title")),
		Form(ID("settingsForm"), Data(flagsFormKey, ""), AutoComplete("off"),
			Map(settingsGroups, func(g settingsGroup) Node {
				return Div(Class("segment-panel"),
					H2(ui.T(ctx, g.titleKey)),
					fields(ctx, g.flags),
				)
			}),
			Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
		),
		Script(Raw(flagsJS)),
	}
}

// Flashes lists every registered flash with a toggle hiding it on this instance. The
// form edits flags.HiddenFlashes, hence its data-flag.
func Flashes(ctx context.Context) Node {
	return Group{
		H1(ui.T(ctx, "flashes.title")),
		Form(ID("flashesForm"), Class("segment-panel"), Data("flag", flags.HiddenFlashes.InternalName()), AutoComplete("off"),
			Map(flashes.All(), func(f flashes.Flash) Node {
				return Div(Class("block my-2"),
					Label(Class("inline-flex items-center text-lg"),
						Input(Class("form-checkbox"), Type("checkbox"), Data("flash", f.Name()), If(flashes.IsHidden(ctx, f), Checked())), Text(" "),
						Span(Class("ml-2"), ui.T(ctx, "flashes.hide")), Text(" "),
						Code(Class("ml-2 text-sm"), Text(f.Name())),
					),
					P(Class("text-muted ml-6"), Text(f.Description())),
				)
			}),
			Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
		),
		Script(Raw(flagsJS)),
		Script(Raw(flashesJS)),
	}
}
