package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	tparse "text/template/parse"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/a-h/templ"
	"github.com/bwmarrin/discordgo"
)

type ProblemTopbar struct {
	IsProblemEditor bool
	IsContestEditor bool
	CanViewTests    bool

	Contest *kilonova.Contest
	Problem *kilonova.Problem

	URLPrefix string

	Page   string
	PageID int
}

func (rt *Web) problemTopbar(r *http.Request, page string, pageID int) *ProblemTopbar {
	prefix := ""
	if util.Contest(r) != nil {
		prefix = fmt.Sprintf("/contests/%d", util.Contest(r).ID)
	}
	return &ProblemTopbar{
		IsProblemEditor: rt.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)),
		IsContestEditor: util.Contest(r).IsEditor(util.UserBrief(r)),
		CanViewTests:    rt.base.CanViewTests(util.UserBrief(r), util.Problem(r)),

		Contest: util.Contest(r),
		Problem: util.Problem(r),

		URLPrefix: prefix,

		Page:   page,
		PageID: pageID,
	}
}

type PostTopbar struct {
	IsPostEditor bool

	Post *kilonova.BlogPost

	Page string
}

func (rt *Web) postTopbar(r *http.Request, page string) *PostTopbar {
	return &PostTopbar{
		IsPostEditor: rt.base.IsBlogPostEditor(util.UserBrief(r), util.BlogPost(r)),

		Post: util.BlogPost(r),
		Page: page,
	}
}

type BlogPostIndexParams struct {
	Posts   []*kilonova.BlogPost
	Authors map[int]*kilonova.UserBrief

	Page     int
	NumPages int
}

type BlogPostParams struct {
	Topbar *PostTopbar

	StatementEditor  *StatementEditorParams
	AttachmentEditor *AttachmentEditorParams

	Attachments []*kilonova.Attachment

	Statement    template.HTML
	StatementAtt *kilonova.Attachment
	Variants     []*kilonova.StatementVariant

	SelectedVariant *kilonova.StatementVariant
}

type ContestsIndexParams struct {
	Contests []*kilonova.Contest

	Page string

	ContestCount int
	PageNum      int
}

type GraderInfoParams struct {
	Languages []*sudoapi.GraderLanguage
}

type DonateParams struct {
	Donations []*kilonova.Donation

	Status string
}

type ResourcesIndexParams struct {
	Resources []*kilonova.ExternalResource
}

type ResourcesPageParams struct {
	Resource *kilonova.ExternalResource
	Problem  *kilonova.Problem
	Author   *kilonova.UserBrief
}

type ResourcesNewParams struct {
	Problem *kilonova.Problem
}

type ContestParams struct {
	Topbar *ProblemTopbar

	Contest *kilonova.Contest

	ContestInvitations []*kilonova.ContestInvitation
	MOSSResults        []*kilonova.MOSSSubmission
}

type ContestInviteParams struct {
	Contest *kilonova.Contest
	Invite  *kilonova.ContestInvitation

	AlreadyRegistered bool

	// may be nil
	Inviter *kilonova.UserBrief
}

type ProblemParams struct {
	Topbar *ProblemTopbar

	Problem     *kilonova.Problem
	Attachments []*kilonova.Attachment
	Tags        []*kilonova.Tag

	Statement template.HTML
	Languages []*sudoapi.Language
	Variants  []*kilonova.StatementVariant

	OlderSubmissions templ.Component

	SelectedVariant *kilonova.StatementVariant

	ShowExternalResources bool
	ExternalResources     []*kilonova.ExternalResource
}

type ProblemTopbarParams struct {
	Topbar *ProblemTopbar

	Languages []*sudoapi.Language
	Problem   *kilonova.Problem
}

type ProblemStatisticsParams struct {
	Topbar *ProblemTopbar

	Problem           *kilonova.Problem
	ProblemStatistics *sudoapi.ProblemStatistics
}

type ProblemArchiveParams struct {
	Topbar *ProblemTopbar

	Tests    []*kilonova.Test
	Problem  *kilonova.Problem
	Settings *kilonova.ProblemEvalSettings
}

type ProblemListParams struct {
	ProblemList *kilonova.ProblemList
	Lists       []*kilonova.ProblemList

	RootProblemList int

	ProblemSources templ.Component
}

type ProblemListProgressParams struct {
	ProblemList *sudoapi.FullProblemList
	CheckedUser *kilonova.UserBrief
}

type SubTaskEditParams struct {
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask
	Topbar  *ProblemTopbar

	ctx  context.Context
	base *sudoapi.BaseAPI
}

func (s *SubTaskEditParams) TestSubTasks(id int) string {
	sts, err := s.base.SubTasksByTest(s.ctx, s.Problem.ID, id)
	if err != nil || sts == nil || len(sts) == 0 {
		return "-"
	}
	out := strconv.Itoa(sts[0].VisibleID)
	for id, st := range sts {
		if id > 0 {
			out += fmt.Sprintf(", %d", st.VisibleID)
		}
	}
	return out
}

func (s *SubTaskEditParams) TestInSubTask(test *kilonova.Test) bool {
	for _, id := range s.SubTask.Tests {
		if id == test.ID {
			return true
		}
	}
	return false
}

type TestEditParams struct {
	Problem *kilonova.Problem
	Test    *kilonova.Test
	Topbar  *ProblemTopbar

	base *sudoapi.BaseAPI
	ctx  context.Context
}

type testDataType struct {
	In  string
	Out string

	OkIn  bool
	OkOut bool
}

const readLimit = 1024 * 1024 // 1MB

func ReadOrTruncate(ctx context.Context, r io.Reader) ([]byte, bool) {
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, readLimit); err != nil {
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), true
		}
		slog.WarnContext(ctx, "Could not read until limit", slog.Any("err", err))
		return []byte("err"), false
	}

	return []byte("Files larger than 1MB cannot be displayed"), false
}

func (t *TestEditParams) NextVID() int {
	return t.base.NextVID(t.ctx, t.Problem.ID)
}

func (t *TestEditParams) GetFullTests() testDataType {
	in, err := t.base.TestInput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := t.base.TestOutput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer out.Close()

	inData, okIn := ReadOrTruncate(t.ctx, in)
	outData, okOut := ReadOrTruncate(t.ctx, out)

	return testDataType{
		In:   string(inData),
		OkIn: okIn,

		Out:   string(outData),
		OkOut: okOut,
	}
}

type IndexParams struct {
	FutureContests  []*kilonova.Contest
	RunningContests []*kilonova.Contest

	ChildProblemLists []*kilonova.ProblemList

	HotProblems     []*kilonova.ScoredProblem
	MoreHotProblems bool

	LatestProblems     []*sudoapi.FullProblem
	MoreLatestProblems bool

	PinnedLists []*kilonova.ProblemList
}

type ProblemListingParams struct {
	Problems []*kilonova.ScoredProblem
	ShowID   bool

	ShowPublished bool

	ContestIDScore int

	ListID int
}

type PblistParams struct {
	Pblist *kilonova.ProblemList
	Open   bool
}

type ProfileParams struct {
	ContentUser       *kilonova.UserFull
	SolvedProblems    []*sudoapi.FullProblem
	SolvedCount       int
	AttemptedProblems []*sudoapi.FullProblem
	AttemptedCount    int

	ChangeHistory []*kilonova.UsernameChange
}

type DiscordLinkParams struct {
	ContentUser *kilonova.UserFull

	DiscordUser *discordgo.User
}

type SessionsParams struct {
	ContentUser *kilonova.UserFull
	Sessions    []*sudoapi.Session
	Page        int
	NumPages    int
}

type AuditLogParams struct {
	Logs     []*kilonova.AuditLog
	Page     int
	NumPages int
}

type ProblemSearchParams struct {
	ProblemList *kilonova.ProblemList

	Results   []*sudoapi.FullProblem
	Groups    []*kilonova.TagGroup
	GroupTags []*kilonova.Tag
	Count     int
}

type VerifiedEmailParams struct {
	ContentUser *kilonova.UserBrief
}

type PasswordResetParams struct {
	User      *kilonova.UserFull
	RequestID string
}

type SubParams struct {
	Submission *kilonova.FullSubmission
}

type PasteParams struct {
	Paste *kilonova.SubmissionPaste

	FullSub *sudoapi.FullSubmission
}

// HTMX modals

type ModalParams struct {
	Small bool

	ChildParams any
}

var calledFunctions = make(map[string]bool)
var addedTemplates = make(map[string]bool)

func doWalk(filename string, nodes ...tparse.Node) bool {
	ok := true
	for _, node := range nodes {
		tp := reflect.Indirect(reflect.ValueOf(node))
		if chainNode, ok := node.(*tparse.ChainNode); ok {
			doWalk(filename, chainNode.Node)
		}
		if val := tp.FieldByName("Ident"); val.IsValid() {
			if val.Kind() == reflect.Slice {
				idents := val.Interface().([]string)
				for _, ident := range idents {
					if _, ok := calledFunctions[ident]; !ok {
						calledFunctions[ident] = true
					}
				}
			} else {
				ident := val.Interface().(string)
				if _, ok := calledFunctions[ident]; !ok {
					calledFunctions[ident] = true
				}
			}
		}
		if val := tp.FieldByName("Field"); val.IsValid() {
			if val.Kind() == reflect.Slice {
				idents := val.Interface().([]string)
				for _, ident := range idents {
					if _, ok := calledFunctions[ident]; !ok {
						calledFunctions[ident] = true
					}
				}
			} else {
				ident := val.Interface().(string)
				if _, ok := calledFunctions[ident]; !ok {
					calledFunctions[ident] = true
				}
			}
		}
		if val := tp.FieldByName("BranchNode"); val.IsValid() {
			node := val.Interface().(tparse.BranchNode)
			doWalk(filename, &node)
		}
		if val := tp.FieldByName("List"); val.IsValid() {
			if val.Kind() == reflect.Pointer {
				val = reflect.Indirect(val)
			}
			if nodes := val.FieldByName("Nodes"); nodes.IsValid() {
				if nodes.Kind() != reflect.Slice {
					slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
					os.Exit(1)
				}
				ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
			}
		}
		if val := tp.FieldByName("ElseList"); val.IsValid() && !val.IsNil() {
			if val.Kind() == reflect.Pointer {
				val = reflect.Indirect(val)
			}
			if nodes := val.FieldByName("Nodes"); nodes.IsValid() {
				if nodes.Kind() != reflect.Slice {
					slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
					os.Exit(1)
				}
				ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
			}
		}
		if nodes := tp.FieldByName("Nodes"); nodes.IsValid() {
			if nodes.Kind() != reflect.Slice {
				slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
				os.Exit(1)
			}
			ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
		}
		if nodes := tp.FieldByName("Node"); nodes.IsValid() {
			ok = ok && doWalk(filename, nodes.Interface().(tparse.Node))
		}
		if nodes := tp.FieldByName("Args"); nodes.IsValid() {
			if nodes.Kind() != reflect.Slice {
				slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
				os.Exit(1)
			}
			ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
		}
		// spew.Dump(node.Type(), node.Position(), node.String())
		if pipe := tp.FieldByName("Pipe"); pipe.IsValid() {
			if pipe.Kind() != reflect.Pointer {
				slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
				os.Exit(1)
			}
			pipe, valid := pipe.Interface().(*tparse.PipeNode)
			if !valid {
				slog.ErrorContext(context.TODO(), "Invalid template static analysis tree")
				os.Exit(1)
			}
			for _, cmd := range pipe.Cmds {
				if len(cmd.Args) == 0 {
					continue
				}
				doWalk(filename, cmd.Args...)
				val, valid := cmd.Args[0].(*tparse.IdentifierNode)
				if !valid {
					continue
				}
				if _, ok := calledFunctions[val.Ident]; !ok {
					calledFunctions[val.Ident] = true
				}
				if val.Ident != "getText" || len(cmd.Args) < 2 {
					continue
				}
				switch node := cmd.Args[1].(type) {
				case *tparse.StringNode:
					key := node.Text
					if !kilonova.TranslationKeyExists(key) {
						slog.InfoContext(context.TODO(), "Template static analysis failed: Unknown translation key", slog.String("key", key), slog.String("filename", filename))
						ok = false
					}
				case *tparse.VariableNode:
				default:
					slog.WarnContext(context.TODO(), "Template static analysis warning: Unknown type for translation string node", slog.Any("type", node.Type()), slog.String("value", node.String()))
				}
			}
			// spew.Dump(rnode)
		}
	}
	return ok
}

func parseTempl(optFuncs template.FuncMap, modal bool, files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		slog.ErrorContext(context.TODO(), "Could not read template directory", slog.Any("err", err))
		os.Exit(1)
	}
	t := template.New(files[0])
	if optFuncs != nil {
		t = t.Funcs(optFuncs)
	}
	f, err := fs.ReadFile(templs, files[0])
	if err != nil {
		slog.ErrorContext(context.TODO(), "Could not read template file", slog.Any("err", err))
		os.Exit(1)
	}
	ptrees, err := tparse.Parse(files[0], string(f), "{{", "}}", optFuncs, builtinTemporaryTemplate())
	if err != nil {
		slog.ErrorContext(context.TODO(), "Could not parse template file", slog.Any("err", err))
		os.Exit(1)
	}
	for _, file := range files {
		addedTemplates["templ/"+file] = true
	}

	if !modal {
		// Check title
		if _, ok := ptrees["title"]; !ok {
			slog.WarnContext(context.TODO(), "Page lacks a title", slog.String("path", files[0]))
		}

		for _, tree := range ptrees {
			doWalk(files[0], tree.Root)
		}
	}
	return template.Must(t.ParseFS(templs, files...))
}

func parseModal(optFuncs template.FuncMap, files ...string) *template.Template {
	return parseTempl(optFuncs, true, slices.Concat([]string{"modals/htmx/helpers.html"}, files)...)
}

func parse(optFuncs template.FuncMap, files ...string) *template.Template {
	return parseTempl(optFuncs, false, files...)
}

func builtinTemporaryTemplate() template.FuncMap {
	names := []string{"and", "call", "html", "index", "slice", "js", "len", "not", "or", "print", "printf", "println", "urlquery", "eq", "ge", "gt", "le", "lt", "ne"}
	rez := make(template.FuncMap)
	for _, name := range names {
		rez[name] = func() {}
	}
	return rez
}
