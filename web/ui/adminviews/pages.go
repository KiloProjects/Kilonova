package adminviews

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/dustin/go-humanize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

var (
	//go:embed scripts/roles.js
	rolesJS string
	//go:embed scripts/users.js
	usersJS string
	//go:embed scripts/flags.js
	flagsJS string
	//go:embed scripts/user_gen.js
	userGenJS string
	//go:embed scripts/debug.js
	debugJS string
)

type PanelParams struct {
	AuthedUserID int

	BoolFlags     []config.Flag[bool]
	StringFlags   []config.Flag[string]
	IntFlags      []config.Flag[int]
	IntArrayFlags []config.Flag[[]int]
	StrArrayFlags []config.Flag[[]string]
}

// Panel is the admin panel: role management and the runtime flags editor.
func Panel(ctx context.Context, params PanelParams) Node {
	loadingGroup := func(id string) Node {
		return Div(ID(id), Class("block list-group mb-2"), Div(Class("list-group-head"), ui.T(ctx, "loading")))
	}
	return Group{
		H1(ui.T(ctx, "panel.admin")),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "changeRoles"), Text(":")),
			Label(Class("block"),
				Span(Class("form-label"), ui.T(ctx, "username"), Text(":")), Text(" "),
				Input(Class("form-input my-1 block"), Type("text"), ID("userToChange"), Placeholder("AlexVasiluta"), AutoComplete("off")),
			),
			Div(Class("my-2 inline-flex"),
				Button(Type("button"), Class("btn-blue font-bold py-2 px-4 rounded-l-sm"), Attr("onclick", "setFormAdmin(true)"), ui.T(ctx, "makeAdmin")), Text(" "),
				Button(Type("button"), Class("btn-blue font-bold py-2 px-4 rounded-r-sm"), Attr("onclick", "setFormProposer(true)"), ui.T(ctx, "makeProposer")),
			),
		),
		Script(Raw(rolesJS)),
		Div(Class("segment-panel"),
			H2(Class("text-2xl"), ui.T(ctx, "admins"), Text(":")),
			loadingGroup("admin-group"),
			H2(Class("text-2xl"), ui.T(ctx, "proposers"), Text(":")),
			loadingGroup("proposer-group"),
		),
		Script(Raw(strings.Replace(usersJS, "AUTHED_USER_ID", strconv.Itoa(params.AuthedUserID), 1))),
		Form(ID("flagsForm"), Class("segment-panel"), AutoComplete("off"),
			H2(Class("text-2xl"), ui.T(ctx, "config.title")),
			flagSection(ctx, "boolFlags", len(params.BoolFlags), Map(params.BoolFlags, func(f config.Flag[bool]) Node {
				return Div(Class("block"),
					Label(Class("inline-flex items-center text-lg"),
						Input(Class("form-checkbox"), Type("checkbox"), Data("name", f.InternalName()), Data("org_value", strconv.FormatBool(f.Value())), If(f.Value(), Checked()), AutoComplete("off")), Text(" "),
						Span(Class("ml-2"), Text(f.HumanName())), Text(" "),
						Span(Class("ml-2 text-sm text-muted"), Text("("+f.InternalName()+")")),
					),
				)
			})),
			flagSection(ctx, "stringFlags", len(params.StringFlags), Map(params.StringFlags, func(f config.Flag[string]) Node {
				return valueFlag(f.HumanName(), f.InternalName(), f.Value(), Type("text"))
			})),
			flagSection(ctx, "intFlags", len(params.IntFlags), Map(params.IntFlags, func(f config.Flag[int]) Node {
				return valueFlag(f.HumanName(), f.InternalName(), strconv.Itoa(f.Value()), Type("number"))
			})),
			flagSection(ctx, "intArrayFlags", len(params.IntArrayFlags), Map(params.IntArrayFlags, func(f config.Flag[[]int]) Node {
				return valueFlag(f.HumanName(), f.InternalName(), jsonList(f.Value()), Type("text"))
			})),
			flagSection(ctx, "strArrayFlags", len(params.StrArrayFlags), Map(params.StrArrayFlags, func(f config.Flag[[]string]) Node {
				return valueFlag(f.HumanName(), f.InternalName(), jsonList(f.Value()), Type("text"))
			})),
			Button(Type("submit"), Class("btn btn-blue"), ui.T(ctx, "button.update")),
		),
		Script(Raw(flagsJS)),
	}
}

// flagSection renders one flag kind; key is both the heading translation key and the div id prefix the script reads.
func flagSection(ctx context.Context, key string, n int, items Node) Node {
	return Div(Class("segment-panel"),
		H3(ui.T(ctx, key)),
		Div(ID(key+"Div"), Class("mb-2"),
			items,
			If(n == 0, Span(Class("text-lg my-2"), Text("N/A"))),
		),
	)
}

func valueFlag(human, name, value string, inputAttrs ...Node) Node {
	return Label(Class("block my-2"),
		Span(Text(human)), Text(" "),
		Span(Class("mr-2 text-sm text-muted"), Text("("+name+")")), Text(" "),
		Input(Class("form-input w-full"), Group(inputAttrs), Data("name", name), Data("org_value", value), Value(value), AutoComplete("off")),
	)
}

// jsonList renders a list flag as its JSON array, never "null".
func jsonList[T any](vals []T) string {
	if vals == nil {
		vals = []T{}
	}
	b, _ := json.Marshal(vals)
	return string(b)
}

// UserGen is the admin account generator form.
func UserGen(ctx context.Context, defaultLang string) Node {
	field := func(key, id, typ string, extra ...Node) Node {
		return Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, key), Text(":")), Text(" "),
			Input(ID(id), Class("form-input"), Type(typ), Group(extra)),
		)
	}
	off := AutoComplete("off")
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "user_gen.title")),
		P(Class("text-muted"), ui.T(ctx, "user_gen.optional_fields")),
		Form(ID("userGenForm"), AutoComplete("off"),
			field("username", "username", "text", off, Required()),
			field("user_gen.display_name", "display_name", "text", off),
			field("email", "email", "email", off),
			field("password", "password", "password", AutoComplete("new-password")),
			field("contestID", "contestID", "number", off),
			Label(Class("block mb-2"),
				Span(Class("form-label"), ui.T(ctx, "displayLang"), Text(":")), Text(" "),
				Select(ID("langSelect"), Class("form-select"),
					Option(Value("ro"), If(defaultLang == "ro", Selected()), Text("🇷🇴 Română")),
					Option(Value("en"), If(defaultLang == "en", Selected()), Text("🇬🇧 English")),
				),
			),
			Details(Class("segment-panel"),
				Summary(H2(Class("inline-block"), ui.T(ctx, "advancedOptions"))),
				Label(Class("block my-2"),
					Span(Class("form-label block"), ui.T(ctx, "createBio"), Text(": ")), Text(" "),
					Textarea(ID("bio"), Class("form-textarea")),
				),
				Div(Class("block mb-2"),
					Label(Class("inline-flex items-center text-lg"),
						Input(Class("form-checkbox"), ID("pwd_mail_enable"), Type("checkbox"), off), Text(" "),
						Span(Class("ml-2"), ui.T(ctx, "user_gen.password_by_mail")),
					),
				),
				field("user_gen.password_mail_address", "pwd_email_address", "email"),
				field("user_gen.password_mail_subject", "pwd_email_subject", "text"),
			),
			Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "button.create")),
		),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "user_gen.created_accounts")),
			P(Class("text-muted"), ui.T(ctx, "user_gen.created_accounts_info")),
			Div(ID("output")),
		),
		Script(Raw(userGenJS)),
	)
}

type AuditLogParams struct {
	Logs       []*kilonova.AuditLog
	Pagination Node // nil when there's a single page
}

func AuditLog(ctx context.Context, params AuditLogParams) Node {
	cell := func(children ...Node) Node { return Td(Class("text-center px-2 py-1"), Group(children)) }
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "panel.audit_log")),
		params.Pagination,
		Table(Class("kn-table mt-4"),
			THead(Tr(
				Th(Scope("col"), Class("w-12 text-center px-4 py-2"), ui.T(ctx, "id")),
				Th(Scope("col"), ui.T(ctx, "author")),
				Th(Scope("col"), ui.T(ctx, "text")),
				Th(Scope("col"), ui.T(ctx, "logDate")),
			)),
			TBody(Map(params.Logs, func(l *kilonova.AuditLog) Node {
				return Tr(Class("kn-table-row"),
					Th(Scope("row"), Class("text-center px-2 py-1"), Text(strconv.Itoa(l.ID))),
					cell(logAuthor(ctx, l)),
					cell(Text(l.Message)),
					cell(ui.ServerTimestamp(l.LogTime)),
				)
			})),
		),
	)
}

func logAuthor(ctx context.Context, l *kilonova.AuditLog) Node {
	switch {
	case l.Author != nil:
		return Group{
			I(Class("fas fa-user fa-fw")), Text(" "),
			A(Href("/profile/"+l.Author.Name), Text(l.Author.Name)),
			Text(" (#" + strconv.Itoa(l.Author.ID) + ")"),
		}
	case l.SystemLog:
		return Group{I(Class("fas fa-cog fa-fw")), Text(" "), ui.T(ctx, "system")}
	default:
		return Group{I(Class("fas fa-question-square fa-fw")), Text(" "), ui.T(ctx, "unknownUser")}
	}
}

// Metric is one runtime/metrics sample. At most one of Int and Float is set.
type Metric struct {
	Name        string
	Description string

	Int   *uint64
	Float *float64
}

type DebugParams struct {
	Metrics     []*Metric
	BucketStats []*datastore.BucketStats
}

func Debug(ctx context.Context, params DebugParams) Node {
	return Group{
		H1(ui.T(ctx, "debug_info")),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "actions")),
			Div(Class("block my-2"),
				Button(Class("btn btn-blue font-bold mr-2"), Attr("onclick", "resetSubs()"), ui.T(ctx, "resetSubs")), Text(" "),
				Button(Class("btn btn-blue font-bold mr-2"), Attr("onclick", "mdCacheWarmup()"), ui.T(ctx, "mdCacheWarmup")),
			),
			Script(Raw(debugJS)),
		),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "buckets")),
			Div(Class("grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4"), Map(params.BucketStats, func(b *datastore.BucketStats) Node {
				return bucketCard(ctx, b)
			})),
		),
		Div(Class("segment-panel"),
			H2(ui.T(ctx, "metrics")),
			Table(Class("kn-table table-fixed"),
				THead(Tr(
					Th(Class("kn-table-cell w-1/2"), ui.T(ctx, "name")),
					Th(Class("kn-table-cell w-1/2"), ui.T(ctx, "value")),
				)),
				TBody(Map(params.Metrics, func(m *Metric) Node {
					return Tr(Class("kn-table-row"),
						Th(Class("kn-table-cell"), Details(Summary(Text(m.Name)), Text(m.Description))),
						Td(Class("kn-table-cell"), Text(metricValue(m))),
					)
				})),
			),
		),
	}
}

func metricValue(m *Metric) string {
	switch {
	case m.Float != nil:
		return strconv.FormatFloat(*m.Float, 'g', -1, 64)
	case m.Int != nil:
		return strconv.FormatUint(*m.Int, 10)
	default:
		return "???"
	}
}

func bucketCard(ctx context.Context, b *datastore.BucketStats) Node {
	// JSON-encode the name so it is a valid JS string literal inside the handler attribute.
	jsName, _ := json.Marshal(b.Name)
	call := func(fn string) Node { return Attr("onclick", fn+"("+string(jsName)+")") }
	return Div(Class("segment-panel reset-list"),
		H3(Text(cases.Title(language.English).String(b.Name))),
		Ul(
			Li(ui.T(ctx, "objectCount"), Text(": "+strconv.Itoa(b.NumItems))),
			Li(ui.T(ctx, "diskSize"), Text(": "+humanize.IBytes(uint64(b.OnDiskSize)))),
			If(b.MaxSize > 1024, Li(ui.T(ctx, "maxBucketSize"), Text(": "+humanize.IBytes(uint64(b.MaxSize))))),
			If(b.MaxTTL != 0, Li(ui.T(ctx, "maxBucketTTL"), Text(": "+b.MaxTTL.String()))),
			Li(ui.T(ctx, "last_updated_at"), Text(": "),
				El("server-timestamp", Attr("timestamp", strconv.FormatInt(b.CreatedAt.UnixMilli(), 10)), Attr("extended")), Text(" "),
				A(call("refreshBucket"), Href("#"), I(Class("fas fa-arrows-rotate"))), Text(" "),
			),
		),
		Div(Class("mt-2"),
			If(!b.Persistent && (b.MaxSize > 1024 || b.MaxTTL != 0),
				Button(Class("btn btn-red font-bold mr-2"), call("evictBucket"), ui.T(ctx, "evictBucket"))),
			If(b.Cache, Button(Class("btn btn-red font-bold mr-2"), call("cleanBucket"), ui.T(ctx, "clearCache"))),
		),
	)
}

// GraderInfo lists the grader's languages. It is visible to everyone, not just admins.
func GraderInfo(ctx context.Context, langs []*sudoapi.GraderLanguage) Node {
	return Group{
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "compilerVersions")),
			Table(Class("kn-table"),
				THead(Tr(
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "languageName")),
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "compileCommand")),
					Th(Class("kn-table-cell"), Scope("col"), ui.T(ctx, "compilerVersion")),
				)),
				TBody(Map(langs, func(l *sudoapi.GraderLanguage) Node {
					return Tr(Class("kn-table-row"),
						Th(Class("kn-table-cell"), Scope("row"), Text(l.Name)),
						Td(Class("kn-table-cell"), Scope("row"), Code(Text(l.Command))),
						Td(Class("kn-table-cell"), Scope("row"), Code(Text(l.Version))),
					)
				})),
			),
		),
		Div(Class("segment-panel"), Text("Note: Page still WIP")),
	}
}

// ContestGen is the bulk contest account generator: it uploads a profiles bundle and downloads the results as a zip.
func ContestGen(ctx context.Context) Node {
	return Div(Class("segment-panel"),
		H1(ui.T(ctx, "contest_gen.title")),
		P(Class("text-muted"), ui.T(ctx, "contest_gen.description")),
		Form(Method("post"), Action("/admin/contest_gen"), EncType("multipart/form-data"),
			Label(Class("block my-2"),
				Span(Class("form-label"), ui.T(ctx, "contest_gen.bundle"), Text(":")), Text(" "),
				Input(Class("form-input"), Type("file"), Name("bundle"), Accept(".json,application/json"), Required()),
			),
			Label(Class("inline-flex items-center text-lg my-2"),
				Input(Class("form-checkbox"), Type("checkbox"), Name("dryRun"), Value("1")),
				Span(Class("ml-2"), ui.T(ctx, "contest_gen.dry_run")),
			),
			Div(Button(Class("btn btn-blue"), Type("submit"), ui.T(ctx, "contest_gen.submit"))),
		),
	)
}
