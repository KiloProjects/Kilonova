package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	tparse "text/template/parse"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

type EditTopbar struct {
	Page   string
	PageID int
}

func (t *EditTopbar) IsOnTest(test *kilonova.Test) bool {
	return t.Page == "tests" && test.VisibleID == t.PageID
}

func (t *EditTopbar) IsOnSubtask(stk *kilonova.SubTask) bool {
	return t.Page == "subtasks" && stk.VisibleID == t.PageID
}

type ReqContext struct {
	User     *kilonova.UserFull
	Language string
}

type ProblemParams struct {
	Ctx *ReqContext
	// User          *kilonova.User
	ProblemEditor bool

	Problem     *kilonova.Problem
	Attachments []*kilonova.Attachment

	Markdown  template.HTML
	Languages map[string]eval.Language
}

type ProblemListParams struct {
	Ctx *ReqContext
	// User          *kilonova.User
	ProblemList *kilonova.ProblemList
}

type SubTaskEditParams struct {
	Ctx *ReqContext
	// User          *kilonova.User
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask
	Topbar  *EditTopbar

	ctx  context.Context
	base *sudoapi.BaseAPI
}

func (s *SubTaskEditParams) ProblemTests() []*kilonova.Test {
	tests, err := s.base.Tests(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

func (s *SubTaskEditParams) ProblemSubTasks() []*kilonova.SubTask {
	sts, err := s.base.SubTasks(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return sts
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
	Ctx *ReqContext
	// User          *kilonova.User
	Problem *kilonova.Problem
	Test    *kilonova.Test
	Topbar  *EditTopbar

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

	inData, okIn := ReadOrTruncate(in)
	outData, okOut := ReadOrTruncate(out)

	return testDataType{
		In:   string(inData),
		OkIn: okIn,

		Out:   string(outData),
		OkOut: okOut,
	}
}

func (t *TestEditParams) ProblemTests() []*kilonova.Test {
	tests, err := t.base.Tests(context.Background(), t.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

type IndexParams struct {
	Ctx *ReqContext

	Version     string
	Description string
}

type ProblemListingParams struct {
	User      *kilonova.UserBrief
	Language  string
	Problems  []*kilonova.Problem
	ShowScore bool
}

type PblistParams struct {
	User   *kilonova.UserBrief
	Ctx    *ReqContext
	Pblist *kilonova.ProblemList
	Open   bool
}

type ProfileParams struct {
	Ctx *ReqContext
	// User          *kilonova.User

	ContentUser  *kilonova.UserFull
	UserProblems []*kilonova.Problem
}

type AuditLogParams struct {
	Ctx *ReqContext

	Logs     []*kilonova.AuditLog
	NumPages int
}

type StatusParams struct {
	Ctx *ReqContext
	// User          *kilonova.User

	Code        int
	Message     string
	ShouldLogin bool
}

type MarkdownParams struct {
	Ctx *ReqContext
	// User          *kilonova.User
	Markdown template.HTML
	Title    string
}

type SimpleParams struct {
	Ctx *ReqContext
	// User          *kilonova.User
}

func GenContext(r *http.Request) *ReqContext {
	return &ReqContext{
		User:     util.UserFull(r),
		Language: util.Language(r),
	}
}

type VerifiedEmailParams struct {
	Ctx *ReqContext
	// User          *kilonova.User

	ContentUser *kilonova.UserBrief
}

type PasswordResetParams struct {
	Ctx *ReqContext

	User      *kilonova.UserFull
	RequestID string
}

type SubParams struct {
	Ctx        *ReqContext
	Submission *kilonova.Submission
}

type PasteParams struct {
	Ctx   *ReqContext
	Paste *kilonova.SubmissionPaste
}

var funcs = template.FuncMap{
	"ispdflink": func(link string) bool {
		u, err := url.Parse(link)
		if err != nil {
			return false
		}
		return path.Ext(u.Path) == ".pdf"
	},
	"encodeJSON": func(data interface{}) (string, error) {
		d, err := json.Marshal(data)
		return base64.StdEncoding.EncodeToString(d), err
	},
	"KBtoMB":     func(kb int) float64 { return float64(kb) / 1024.0 },
	"hashedName": fsys.HashName,
	"version":    func() string { return kilonova.Version },
	"debug":      func() bool { return config.Common.Debug },
	"intList": func(ids []int) string {
		if ids == nil {
			return ""
		}
		var b strings.Builder
		for i, id := range ids {
			b.WriteString(strconv.Itoa(id))
			if i != len(ids)-1 {
				b.WriteRune(',')
			}
		}
		return b.String()
	},
	"shallowPblistIDs": func(lists []*kilonova.ShallowProblemList) []int {
		rez := []int{}
		for _, l := range lists {
			rez = append(rez, l.ID)
		}
		return rez
	},
	"httpstatus": http.StatusText,
	"dump":       spew.Sdump,
	"getText": func(lang, line string, args ...any) template.HTML {
		return template.HTML(kilonova.GetText(lang, line, args...))
	},
}

type executor interface {
	Execute(io.Writer, any) error
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
					panic("Wtf")
				}
				ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
			}
		}
		if nodes := tp.FieldByName("Nodes"); nodes.IsValid() {
			if nodes.Kind() != reflect.Slice {
				panic("Wtf")
			}
			ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
		}
		//spew.Dump(node.Type(), node.Position(), node.String())
		if rnode, ok := node.(*tparse.ActionNode); ok {
			for _, cmd := range rnode.Pipe.Cmds {
				if len(cmd.Args) == 0 {
					continue
				}
				val, ok := cmd.Args[0].(*tparse.IdentifierNode)
				if !ok || val.Ident != "getText" || len(cmd.Args) < 3 {
					continue
				}
				key := cmd.Args[2].(*tparse.StringNode).Text
				if !kilonova.TranslationKeyExists(key) {
					zap.S().Infof("Template static analysis failed: Unknown translation key %q in file %s", key, filename)
					ok = false
				}
			}
			//spew.Dump(rnode)
		}
	}
	return ok
}

func parse(optFuncs template.FuncMap, files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		zap.S().Fatal(err)
	}
	t := template.New("layout.html").Funcs(funcs)
	if optFuncs != nil {
		t = t.Funcs(optFuncs)
	}
	files = append(files, "util/navbar.html", "util/footer.html")
	if true { //config.Common.Debug { // && false {
		f, err := fs.ReadFile(templs, files[0])
		if err != nil {
			zap.S().Fatal(err)
		}
		ptrees, err := tparse.Parse(files[0], string(f), "{{", "}}", funcs, optFuncs, builtinTemporaryTemplate())
		if err != nil {
			zap.S().Fatal(err)
		}
		tree := ptrees["content"]
		doWalk(files[0], tree.Root)
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
