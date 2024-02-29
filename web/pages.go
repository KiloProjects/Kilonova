package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"reflect"
	"strconv"
	tparse "text/template/parse"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
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
		IsContestEditor: rt.base.IsContestEditor(util.UserBrief(r), util.Contest(r)),
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
	Languages    map[string]eval.Language
	Variants     []*kilonova.StatementVariant

	SelectedLang   string
	SelectedFormat string
}

type ContestsIndexParams struct {
	Contests []*kilonova.Contest

	Page string

	ContestCount int
	PageNum      int
}

type DonateParams struct {
	Donations []*kilonova.Donation

	Status   string
	BMACName string
	PayPalID string
}

type ContestParams struct {
	Topbar *ProblemTopbar

	Contest *kilonova.Contest

	ContestInvitations []*kilonova.ContestInvitation
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

	Submissions *sudoapi.Submissions

	Statement template.HTML
	Languages map[string]eval.Language
	Variants  []*kilonova.StatementVariant

	SelectedLang   string
	SelectedFormat string
}

type ProblemTopbarParams struct {
	Topbar *ProblemTopbar

	Languages map[string]eval.Language
	Problem   *kilonova.Problem
}

type ProblemListParams struct {
	ProblemList *kilonova.ProblemList
	Lists       []*kilonova.ProblemList

	RootProblemList int
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
}

type testDataType struct {
	In  string
	Out string

	OkIn  bool
	OkOut bool
}

const readLimit = 1024 * 1024 // 1MB

func ReadOrTruncate(r io.Reader) ([]byte, bool) {
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, readLimit); err != nil {
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), true
		}
		zap.S().Warn(err)
		return []byte("err"), false
	}

	return []byte("Files larger than 1MB cannot be displayed"), false
}

func (t *TestEditParams) GetFullTests() testDataType {
	in, err := t.base.GraderStore().TestInput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := t.base.GraderStore().TestOutput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer out.Close()

	inData, okIn := ReadOrTruncate(in)
	outData, okOut := ReadOrTruncate(out)

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

	HotProblems  []*kilonova.ScoredProblem
	MoreProblems bool

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

type StatusParams struct {
	Code    int
	Message string
}

type MarkdownParams struct {
	Markdown template.HTML
	Title    string
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

func doWalk(filename string, nodes ...tparse.Node) bool {
	ok := true
	for _, node := range nodes {
		tp := reflect.Indirect(reflect.ValueOf(node))
		if val := tp.FieldByName("List"); val.IsValid() {
			if val.Kind() == reflect.Pointer {
				val = reflect.Indirect(val)
			}
			if nodes := val.FieldByName("Nodes"); nodes.IsValid() {
				if nodes.Kind() != reflect.Slice {
					zap.S().Fatalf("Invalid template static analysis tree")
				}
				ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
			}
		}
		if nodes := tp.FieldByName("Nodes"); nodes.IsValid() {
			if nodes.Kind() != reflect.Slice {
				zap.S().Fatalf("Invalid template static analysis tree")
			}
			ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
		}
		// spew.Dump(node.Type(), node.Position(), node.String())
		if rnode, valid := node.(*tparse.ActionNode); valid {
			for _, cmd := range rnode.Pipe.Cmds {
				if len(cmd.Args) == 0 {
					continue
				}
				val, valid := cmd.Args[0].(*tparse.IdentifierNode)
				if !valid || val.Ident != "getText" || len(cmd.Args) < 2 {
					continue
				}
				switch node := cmd.Args[1].(type) {
				case *tparse.StringNode:
					key := node.Text
					if !kilonova.TranslationKeyExists(key) {
						zap.S().Infof("Template static analysis failed: Unknown translation key %q in file %s", key, filename)
						ok = false
					}
				case *tparse.VariableNode:
				default:
					zap.S().Warnf("Template static analysis warning: Unknown type for translation string node: %v (value: %s)", node.Type(), node.String())
				}
			}
			// spew.Dump(rnode)
		}
	}
	return ok
}

func parse(optFuncs template.FuncMap, files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		zap.S().Fatal(err)
	}
	t := template.New("layout.html")
	if optFuncs != nil {
		t = t.Funcs(optFuncs)
	}
	files = append(files, "util/navbar.html", "util/footer.html")
	if true { //config.Common.Debug { // && false {
		f, err := fs.ReadFile(templs, files[0])
		if err != nil {
			zap.S().Fatal(err)
		}
		ptrees, err := tparse.Parse(files[0], string(f), "{{", "}}", optFuncs, builtinTemporaryTemplate())
		if err != nil {
			zap.S().Fatal(err)
		}

		// Check title
		if _, ok := ptrees["title"]; !ok {
			zap.S().Warnf("Page %s lacks a title", files[0])
		}

		// Check content
		tree := ptrees["content"]
		if tree != nil {
			doWalk(files[0], tree.Root)
		}
	}
	return template.Must(t.ParseFS(templs, append([]string{"layout.html"}, files...)...))
}

func builtinTemporaryTemplate() template.FuncMap {
	names := []string{"and", "call", "html", "index", "slice", "js", "len", "not", "or", "print", "printf", "println", "urlquery", "eq", "ge", "gt", "le", "lt", "ne"}
	rez := make(template.FuncMap)
	for _, name := range names {
		rez[name] = func() {}
	}
	return rez
}
