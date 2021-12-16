package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
)

var (
	status = parse(nil, "util/statusCode.html", "modals/login.html")
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
	User     *kilonova.User
	Language string
}

type ProblemParams struct {
	Ctx           *ReqContext
	ProblemEditor bool

	Problem     *kilonova.Problem
	Author      *kilonova.User
	Attachments []*kilonova.Attachment

	Contest *kilonova.Contest

	Markdown  template.HTML
	Languages map[string]eval.Language
}

type ProblemListParams struct {
	Ctx         *ReqContext
	ProblemList *kilonova.ProblemList
}

type SubTaskEditParams struct {
	Ctx     *ReqContext
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask
	Topbar  *EditTopbar

	ctx context.Context
	db  *db.DB
}

func (s *SubTaskEditParams) ProblemTests() []*kilonova.Test {
	tests, err := s.db.Tests(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

func (s *SubTaskEditParams) ProblemSubTasks() []*kilonova.SubTask {
	sts, err := s.db.SubTasks(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return sts
}

func (s *SubTaskEditParams) TestSubTasks(id int) string {
	sts, err := s.db.SubTasksByTest(s.ctx, s.Problem.ID, id)
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
	Ctx     *ReqContext
	Problem *kilonova.Problem
	Test    *kilonova.Test
	Topbar  *EditTopbar

	db *db.DB
	dm kilonova.DataStore
}

type testDataType struct {
	In  string
	Out string
}

func (t *TestEditParams) GetFullTests() testDataType {
	in, err := t.dm.TestInput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := t.dm.TestOutput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer out.Close()

	inData, err := io.ReadAll(in)
	if err != nil {
		inData = []byte("err")
	}
	outData, err := io.ReadAll(out)
	if err != nil {
		outData = []byte("err")
	}

	return testDataType{In: string(inData), Out: string(outData)}
}

func (t *TestEditParams) ProblemTests() []*kilonova.Test {
	tests, err := t.db.Tests(context.Background(), t.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

type IndexParams struct {
	Ctx *ReqContext

	Version string
	Config  config.IndexConf
}

type ProblemListingParams struct {
	User     *kilonova.User
	Language string
	Problems []*kilonova.Problem
}

type ProfileParams struct {
	Ctx *ReqContext

	ContentUser  *kilonova.User
	UserProblems []*kilonova.Problem
}

type StatusParams struct {
	Ctx *ReqContext

	Code        int
	Message     string
	ShouldLogin bool
}

func Status(w io.Writer, params *StatusParams) (err error) {
	err = status.Execute(w, params)
	if err != nil {
		log.Println(err)
	}
	return
}

type MarkdownParams struct {
	Ctx      *ReqContext
	Markdown template.HTML
	Title    string
}

type SimpleParams struct {
	Ctx *ReqContext
}

type AdminParams struct {
	Ctx          *ReqContext
	IndexDesc    string
	IndexListAll bool
	IndexLists   string
}

func GenContext(r *http.Request) *ReqContext {
	return &ReqContext{
		User:     util.User(r),
		Language: util.Language(r),
	}
}

type VerifiedEmailParams struct {
	Ctx *ReqContext

	ContentUser *kilonova.User
}

type SubParams struct {
	Ctx        *ReqContext
	Submission *kilonova.Submission
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
	"version":    func() string { return kilonova.Version },
	"debug":      func() bool { return config.Common.Debug },
	"intList":    kilonova.SerializeIntList,
	"httpstatus": http.StatusText,
	"dump":       spew.Sdump,
	"getText": func(lang, line string, args ...interface{}) template.HTML {
		if _, ok := translations[line]; !ok {
			log.Printf("Invalid translation key %q\n", line)
			return "ERR"
		}
		if _, ok := translations[line][lang]; !ok {
			return template.HTML(translations[line][config.Common.DefaultLang])
		}
		return template.HTML(fmt.Sprintf(translations[line][lang], args...))
	},
}

func parse(optFuncs template.FuncMap, files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		log.Fatal(err)
	}
	t := template.New("layout.html").Funcs(funcs)
	if optFuncs != nil {
		t = t.Funcs(optFuncs)
	}
	files = append(files, "util/navbar.html", "util/footer.html")
	return template.Must(t.ParseFS(templs, append([]string{"layout.html"}, files...)...))
}
