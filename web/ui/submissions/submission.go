package submissions

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/alecthomas/chroma/v2"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/dustin/go-humanize"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type SubmissionPageParams struct {
	Submission      *kilonova.FullSubmission
	SubmissionFiles []*kilonova.SubmissionFile
	ForceShowCode   bool

	LanguageFormatter func(string) string
	OlderSubmissions  *modals.OlderSubmissionsParams
}

type PastePageParams struct {
	SubmissionPageParams
	Paste *kilonova.SubmissionPaste
}

// pollTrigger re-fetches the page every second until grading is done.
func pollTrigger(sub *kilonova.FullSubmission) string {
	if sub.Status != kilonova.StatusFinished {
		return "kn-poll, every 1s"
	}
	return "kn-poll"
}

func Submission(ctx context.Context, params SubmissionPageParams) Node {
	sub := params.Submission
	u := user.UserBriefContext(ctx)
	outputOnly := sub.Language == "outputOnly" || sub.Language == "ai"

	return Div(ID("sub-holder"), Attr("hx-select", "#sub-holder"), Attr("hx-get", fmt.Sprintf("/submissions/%d", sub.ID)),
		Attr("hx-swap", "morph:outerHTML"), Attr("hx-trigger", pollTrigger(sub)),
		H1(ui.T(ctx, "sub"), Textf(" #%d", sub.ID)),
		Div(Class("page-holder"),
			Aside(Class("page-sidebar lg:order-last"), summary(ctx, sub, nil, params.LanguageFormatter), olderSubs(ctx, params.OlderSubmissions)),
			Div(Class("page-content-wrapper"), scoreBreakdown(ctx, params)),
		),
		Map(params.SubmissionFiles, func(file *kilonova.SubmissionFile) Node {
			if outputOnly {
				return outputOnlyCode(ctx, sub, file)
			}
			return code(ctx, sub, file, params.ForceShowCode)
		}),
		Div(Class("my-2"), Iff(u != nil, func() Node {
			return Group{
				If(sub.CanDelete(u), Group{
					Div(Class("inline-block"),
						Button(Attr("hx-delete", fmt.Sprintf("/submissions/%d", sub.ID)), Class("btn btn-red"),
							Attr("hx-target", "body"), Attr("hx-swap", "outerHTML"), Attr("hx-confirm", ui.TS(ctx, "subDeleteConfirm")),
							ui.T(ctx, "removeSub")),
					),
					Text(" "),
					postForm(fmt.Sprintf("/submissions/%d/reevaluate", sub.ID), ui.T(ctx, "reevaluate")),
				}),
				If(flags.PastesEnabled.Value() && sub.IsEditor(u), postForm(fmt.Sprintf("/submissions/%d/paste", sub.ID), ui.T(ctx, "create_paste"))),
			}
		})),
	)
}

func Paste(ctx context.Context, params PastePageParams) Node {
	sub := params.Submission
	return Div(ID("paste-holder"), Attr("hx-trigger", pollTrigger(sub)),
		H1(ui.T(ctx, "paste_title"), Text(" #"+params.Paste.ID)),
		Div(Class("page-holder"),
			Aside(Class("page-sidebar lg:order-last"), summary(ctx, sub, params.Paste.Author, params.LanguageFormatter), olderSubs(ctx, params.OlderSubmissions)),
			Div(Class("page-content-wrapper"), Map(params.SubmissionFiles, func(file *kilonova.SubmissionFile) Node {
				return code(ctx, sub, file, params.ForceShowCode)
			})),
		),
		scoreBreakdown(ctx, params.SubmissionPageParams),
		If(params.Paste.IsEditor(user.UserBriefContext(ctx)),
			Button(Attr("hx-delete", "/pastes/"+params.Paste.ID), Attr("hx-target", "body"), Attr("hx-swap", "outerHTML"), Class("btn btn-red"),
				Attr("hx-confirm", ui.TS(ctx, "pasteDeleteConfirm")), ui.T(ctx, "del_paste")),
		),
	)
}

func olderSubs(ctx context.Context, params *modals.OlderSubmissionsParams) Node {
	return Iff(params != nil, func() Node { return Div(Class("segment-panel"), modals.OlderSubmissions(ctx, params)) })
}

func postForm(action string, label Node) Node {
	return Form(Method("post"), Action(action), Class("inline-block"), ui.SubmitButton(label))
}

func scoreBreakdown(ctx context.Context, params SubmissionPageParams) Node {
	sub := params.Submission
	var tests Node
	switch {
	case sub.CompileError != nil && *sub.CompileError:
	case len(sub.SubTasks) > 0 && sub.SubmissionType == kilonova.EvalTypeClassic:
		tests = subTasks(ctx, sub, false)
	default:
		tests = Group{H2(ui.T(ctx, "tests")), testTable(ctx, testTableParams{
			Subtests: sub.SubTests, Subtasks: sub.SubTasks, ProblemEditor: sub.ProblemEditor, Precision: sub.ScorePrecision, SubType: sub.SubmissionType,
		})}
	}
	return Div(Class("segment-panel"), compileErrorInfo(ctx, sub), tests)
}

// scoreOutOf renders "earned / max" at the problem's precision.
func scoreOutOf(percentage, maxScore decimal.Decimal, precision int32) string {
	earned := ui.RemoveTrailingZeros(percentage.Shift(-2).Mul(maxScore).StringFixed(precision))
	return earned + " / " + ui.RemoveTrailingZeros(maxScore.StringFixed(precision))
}

type subTaskParams struct {
	SubTask       *kilonova.SubmissionSubTask
	Precision     int32
	Subtests      []*kilonova.SubTest
	ProblemEditor bool
	DefaultOpen   bool
	// BreakdownMode links to the submission the subtask's score came from, for the
	// max-score breakdown of sum-of-subtasks problems (currently rendered in Preact).
	BreakdownMode bool
}

func subTask(ctx context.Context, params subTaskParams) Node {
	st := params.SubTask
	badge := Span(Class("float-right badge"), I(Class("fas fa-cog animate-spin")))
	if st.FinalPercentage != nil {
		badge = Span(Class("float-right badge score-cell"), Data("score", st.FinalPercentage.String()),
			Text(scoreOutOf(*st.FinalPercentage, st.Score, params.Precision)))
	}
	return Details(ID(fmt.Sprintf("sub-subtask-%d", st.ID)), Class("list-group-item"), If(params.DefaultOpen, Attr("open")),
		Summary(Class("pb-1 mt-1"),
			Span(ui.T(ctx, "nthSubTask", st.VisibleID), Text(" "),
				If(params.BreakdownMode, Group{
					Text("("), ui.T(ctx, "from_sub"), Text(" "),
					A(Href(fmt.Sprintf("/submissions/%d", st.SubmissionID)), Textf("#%d", st.SubmissionID)), Text(")"),
				}),
			),
			Text(" "), badge,
		),
		testTable(ctx, testTableParams{
			Subtests: params.Subtests, ProblemEditor: params.ProblemEditor, Precision: params.Precision, SubType: kilonova.EvalTypeClassic, Subtask: st,
		}),
	)
}

func subTasks(ctx context.Context, sub *kilonova.FullSubmission, expandedTests bool) Node {
	return Group{
		Details(ID("sub-subtasks"), Attr("open"),
			Summary(H2(Class("inline-block"), ui.T(ctx, "subTasks"))),
			Div(Class("list-group mb-2"), Map(sub.SubTasks, func(st *kilonova.SubmissionSubTask) Node {
				return subTask(ctx, subTaskParams{
					SubTask: st, Subtests: sub.SubTests, ProblemEditor: sub.ProblemEditor, Precision: sub.ScorePrecision, DefaultOpen: len(sub.SubTasks) == 1,
				})
			})),
		),
		Details(ID("expanded-tests"), If(expandedTests, Attr("open")),
			Summary(H2(Class("inline-block"), ui.T(ctx, "individualTests"))),
			testTable(ctx, testTableParams{
				Subtests: sub.SubTests, Subtasks: sub.SubTasks, ProblemEditor: sub.ProblemEditor, Precision: sub.ScorePrecision, SubType: sub.SubmissionType,
			}),
		),
	}
}

type testTableParams struct {
	Subtests      []*kilonova.SubTest
	Subtasks      []*kilonova.SubmissionSubTask // when set, adds a column listing each test's subtasks
	Subtask       *kilonova.SubmissionSubTask   // when set, only its tests are shown, scored against it
	ProblemEditor bool
	Precision     int32
	SubType       kilonova.EvalType
}

func testTable(ctx context.Context, params testTableParams) Node {
	classic := params.SubType == kilonova.EvalTypeClassic
	subtests := slices.Clone(params.Subtests)
	if params.Subtask != nil {
		subtests = slices.DeleteFunc(subtests, func(st *kilonova.SubTest) bool { return !slices.Contains(params.Subtask.Subtests, st.ID) })
	}
	slices.SortFunc(subtests, func(a, b *kilonova.SubTest) int { return cmp.Compare(a.VisibleID, b.VisibleID) })

	return Table(Classes{"kn-table": true, "default-background": params.Subtask != nil, "mb-2": true},
		THead(Tr(
			Th(Class("py-2"), Scope("col"), ui.T(ctx, "id")),
			Th(Scope("col"), ui.T(ctx, "time")),
			Th(Scope("col"), ui.T(ctx, "memory")),
			Th(Scope("col"), ui.T(ctx, "verdict")),
			If(classic, Th(Scope("col"), ui.T(ctx, "score"))),
			If(classic && len(params.Subtasks) > 0, Th(Scope("col"), ui.T(ctx, "subTasks"))),
			If(params.ProblemEditor, Th(Scope("col"), ui.T(ctx, "output"))),
		)),
		TBody(Map(subtests, func(st *kilonova.SubTest) Node { return testRow(ctx, params, st) })),
	)
}

func testRow(ctx context.Context, params testTableParams, st *kilonova.SubTest) Node {
	classic := params.SubType == kilonova.EvalTypeClassic
	dash := Td(Text("-"))
	var cells Node
	switch {
	case st.Skipped:
		cells = Group{dash, dash, Td(testVerdict(ctx, st.Verdict)), If(classic, dash)}
	case st.Done:
		maxScore := st.Score
		if params.Subtask != nil {
			maxScore = params.Subtask.Score
		}
		score := Text(scoreOutOf(st.Percentage, maxScore, params.Precision))
		if len(params.Subtasks) > 0 {
			score = Group{Text(st.Percentage.String() + "% "), ui.T(ctx, "correct")}
		}
		cells = Group{
			Td(Text(fmt.Sprint(math.Floor(st.Time*1000)) + " ms")),
			Td(Text(humanize.IBytes(uint64(st.Memory * 1024)))),
			Td(testVerdict(ctx, st.Verdict)),
			If(classic, Td(Class("text-black score-cell"), Data("score", st.Percentage.String()), score)),
		}
	default:
		cells = Group{Td(), Td(), Td(Div(Class("fas fa-spinner animate-spin"), Role("status")), Text(" "), ui.T(ctx, "waiting")), If(classic, dash)}
	}

	return Tr(Class("kn-table-row"),
		Th(Class("py-1"), Scope("row"), ID(fmt.Sprintf("test-%d", st.VisibleID)), Text(strconv.Itoa(st.VisibleID))),
		cells,
		Iff(classic && len(params.Subtasks) > 0, func() Node {
			var ids []string
			for _, stk := range params.Subtasks {
				if slices.Contains(stk.Subtests, st.ID) {
					ids = append(ids, strconv.Itoa(stk.VisibleID))
				}
			}
			return Td(Text(strings.Join(ids, ", ")))
		}),
		If(params.ProblemEditor, Td(A(Href(fmt.Sprintf("/assets/subtest/%d", st.ID)), ui.T(ctx, "output")))),
	)
}

func compileErrorInfo(ctx context.Context, sub *kilonova.FullSubmission) Node {
	var msg string
	if sub.CompileMessage != nil {
		msg = *sub.CompileMessage
	}
	pre := func(text string) Node { return Pre(Class("mb-2"), Style("word-break: break-all;"), Text(text)) }
	summary := Summary(H2(Class("inline-block"), ui.T(ctx, "compileMsg")))

	if sub.CompileError != nil && *sub.CompileError {
		return Group{
			H2(ui.T(ctx, "compileErr")),
			Details(ID("compile-error-details"), Attr("open"), summary, pre(cmp.Or(msg, "No compilation message available"))),
		}
	}
	if msg != "" {
		return Details(ID("compile-error-details"), summary, pre(msg))
	}
	return nil
}

// summary is the submission info table. pasteAuthor adds "shared by" rows on paste pages.
func summary(ctx context.Context, sub *kilonova.FullSubmission, pasteAuthor *kilonova.UserBrief, langFmt func(string) string) Node {
	row := func(key string, value ...Node) Node {
		return Tr(Class("kn-table-simple-border"), Td(Class("kn-table-cell"), ui.T(ctx, key)), Td(Class("kn-table-cell"), Group(value)))
	}
	dashOr := func(v float64, format func() string) Node {
		if v == -1 {
			return Text("-")
		}
		return Text(format())
	}
	problemURL := fmt.Sprintf("/problems/%d", sub.Problem.ID)
	if sub.ContestID != nil {
		problemURL = fmt.Sprintf("/contests/%d%s", *sub.ContestID, problemURL)
	}
	graded := sub.Status == kilonova.StatusFinished || sub.Status == kilonova.StatusReevaling
	u := user.UserBriefContext(ctx)

	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "info")),
		Table(Class("kn-table"), TBody(
			Iff(pasteAuthor != nil, func() Node {
				return Group{
					row("shared_by", A(Href("/profile/"+pasteAuthor.Name), Text(pasteAuthor.Name))),
					row("sub_id", A(Href(fmt.Sprintf("/submissions/%d", sub.ID)), Textf("#%d", sub.ID))),
				}
			}),
			row("sub_author", A(Href("/profile/"+sub.Author.Name), Text(sub.Author.Name))),
			row("problemSingle", A(Href(problemURL), Text(sub.Problem.Name))),
			row("uploadDate", ui.ServerTimestamp(sub.CreatedAt)),
			If(graded && sub.SubmissionType == kilonova.EvalTypeClassic,
				row("score", Span(Class("badge-lite font-bold score-cell"), Data("score", sub.Score.String()), ui.Score(sub.Problem, &sub.Score, false)))),
			If(graded && sub.SubmissionType != kilonova.EvalTypeClassic, row("verdict", Span(Class("badge-lite font-bold"), icpcVerdict(ctx, sub)))),
			If(graded, row("time", dashOr(sub.MaxTime, func() string { return fmt.Sprint(math.Floor(sub.MaxTime*1000)) + " ms" }))),
			If(graded, row("memory", dashOr(float64(sub.MaxMemory), func() string { return humanize.IBytes(uint64(sub.MaxMemory * 1024)) }))),
			If(sub.Problem.DefaultPoints.IsPositive(), row("defaultPoints", Text(sub.Problem.DefaultPoints.String()))),
			row("language", Text(langFmt(sub.Language))),
			If(sub.CodeSize > 0, row("codeSize", Text(humanize.IBytes(uint64(sub.CodeSize))))),
			row("status", Text(string(sub.Status))),
			Iff(sub.CompileTime != nil, func() Node { return row("compileTime", Text(fmt.Sprint(math.Floor(*sub.CompileTime*1000))+" ms")) }),
			Iff(u.IsAdmin() && sub.IP != nil, func() Node { return row("ipAddr", Text(sub.IP.String())) }),
		)),
	)
}

func icpcVerdict(ctx context.Context, sub *kilonova.FullSubmission) Node {
	switch {
	case sub.Score.Equal(decimal.NewFromInt(100)):
		return Group{I(Class("fas fa-fw fa-check")), Text(" "), ui.T(ctx, "accepted")}
	case sub.ICPCVerdict != nil && *sub.ICPCVerdict != "":
		return Text(icpcVerdictText(ctx, *sub.ICPCVerdict))
	}
	return ui.T(ctx, "rejected")
}

func showCodePrompt(ctx context.Context) Node {
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "showSourceCodeQ")),
		P(Class("mb-4"), ui.T(ctx, "showSourceCodeExpl")),
		A(Class("inline-block btn btn-blue mx-auto"), Href("?forceCode=1"), ui.T(ctx, "showSourceCodeBtn")),
	)
}

func downloadLink(ctx context.Context, sub *kilonova.FullSubmission) Node {
	return Div(Class("block my-2"),
		A(Href(fmt.Sprintf("/submissions/%d/download", sub.ID)), Class("btn btn-blue text-semibold text-lg"), ui.T(ctx, "download")),
	)
}

// outputOnlyCode offers the uploaded output for download instead of highlighting it.
func outputOnlyCode(ctx context.Context, sub *kilonova.FullSubmission, file *kilonova.SubmissionFile) Node {
	if file == nil || len(file.Data) == 0 {
		return nil
	}
	if !sub.CodeTrulyVisible {
		return showCodePrompt(ctx)
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "sourceCode"), Text(" ("), Code(Text(file.Filename)), Text(")")),
		downloadLink(ctx, sub),
	)
}

// code shows the highlighted source, or a prompt to reveal it. forceShow is set on pastes.
func code(ctx context.Context, sub *kilonova.FullSubmission, file *kilonova.SubmissionFile, forceShow bool) Node {
	if file == nil || len(file.Data) == 0 {
		return nil
	}
	if !forceShow && !sub.CodeTrulyVisible {
		return showCodePrompt(ctx)
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "sourceCode"), Text(" ("), Code(Text(file.Filename)), Text("):")),
		syntaxHighlight(file.Data, sub.Language),
		downloadLink(ctx, sub),
	)
}

var chromaFmt = chtml.New(chtml.WithClasses(true), chtml.TabWidth(4))

// syntaxHighlight renders code as chroma HTML, which escapes the source itself.
func syntaxHighlight(code []byte, lang string) Node {
	return NodeFunc(func(w io.Writer) error {
		switch lang {
		case "pascal":
			lang = "pas"
		case "nodejs":
			lang = "js"
		}
		lexer := lexers.Get(strings.TrimFunc(lang, unicode.IsDigit))
		if lexer == nil {
			lexer = lexers.Fallback
		}
		it, err := chroma.Coalesce(lexer).Tokenise(nil, string(code))
		if err != nil {
			return err
		}
		return chromaFmt.Format(w, styles.Get("github"), it)
	})
}

var (
	icpcVerdictRe = regexp.MustCompile(`test_verdict.([a-z_]+)`)
	testVerdictRe = regexp.MustCompile(`translate:([a-z_]+)`)
)

// testVerdict translates "translate:key" markers in a checker verdict and keeps its line breaks.
func testVerdict(ctx context.Context, verdict string) Node {
	translated := testVerdictRe.ReplaceAllStringFunc(verdict, func(s string) string {
		return kilonova.MaybeGetText(util.LanguageContext(ctx), strings.ReplaceAll(s, "translate:", "test_verdict."))
	})
	var lines Group
	for i, line := range strings.Split(translated, "\n") {
		if i > 0 {
			lines = append(lines, Br())
		}
		lines = append(lines, Text(line))
	}
	return lines
}
