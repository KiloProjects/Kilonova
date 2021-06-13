package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/davecgh/go-spew/spew"
)

var (
	settings = parse(nil, "settings.html")
	profile  = parse(nil, "profile.html")
	status   = parse(nil, "util/statusCode.html", "auth/modal/login.html")
	index    = parse(nil, "index.html")

	pbs = parse(nil, "pbs.html")
	pb  = parse(nil, "pb.html")

	subs = parse(nil, "submissions.html")
	sub  = parse(nil, "submission.html")

	login  = parse(nil, "auth/login.html", "auth/modal/login.html")
	signup = parse(nil, "auth/signup.html")

	editIndex       = parse(nil, "edit/index.html", "edit/topbar.html")
	editDesc        = parse(nil, "edit/desc.html", "edit/topbar.html")
	editChecker     = parse(nil, "edit/checker.html", "edit/topbar.html")
	editAttachments = parse(nil, "edit/attachments.html", "edit/topbar.html")

	testAdd    = parse(nil, "edit/testAdd.html", "edit/topbar.html")
	testEdit   = parse(nil, "edit/testEdit.html", "edit/topbar.html")
	testScores = parse(nil, "edit/testScores.html", "edit/topbar.html")

	subtaskAdd   = parse(nil, "edit/subtaskAdd.html", "edit/topbar.html")
	subtaskEdit  = parse(nil, "edit/subtaskEdit.html", "edit/topbar.html")
	subtaskIndex = parse(nil, "edit/subtaskIndex.html", "edit/topbar.html")

	pbListIndex  = parse(nil, "lists/index.html")
	pbListCreate = parse(nil, "lists/create.html")
	pbListView   = parse(nil, "lists/view.html")

	adminPanel = parse(nil, "admin/admin.html")
	knaPanel   = parse(nil, "admin/kna.html")
	testUI     = parse(nil, "admin/test-ui.html")

	adminUserPanel = parse(nil, "admin/users.html")

	markdown = parse(nil, "util/mdrender.html")

	proposerPanel = parse(nil, "proposer/index.html", "proposer/createpb.html", "proposer/cdn_manager.html")

	verifiedEmail = parse(nil, "verified-email.html")
	sentEmail     = parse(nil, "util/sent.html")

	listIndex  = parse(nil, "lists/index.html")
	listCreate = parse(nil, "lists/create.html")
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

type ProblemParams struct {
	User          *kilonova.User
	ProblemEditor bool

	Problem     *kilonova.Problem
	Author      *kilonova.User
	Attachments []*kilonova.Attachment

	Markdown  template.HTML
	Languages map[string]eval.Language
}

type ProblemListParams struct {
	User        *kilonova.User
	ProblemList *kilonova.ProblemList

	ctx context.Context
	db  kilonova.DB
	r   kilonova.MarkdownRenderer
}

func (p *ProblemListParams) RenderMarkdown(body string) template.HTML {
	val, err := p.r.Render([]byte(body))
	if err != nil {
		return ""
	}
	return template.HTML(val)
}

func (p *ProblemListParams) ProblemLists() []*kilonova.ProblemList {
	list, err := p.db.ProblemLists(p.ctx, kilonova.ProblemListFilter{})
	if err != nil {
		return nil
	}
	return list
}

func (p *ProblemListParams) ListProblems(list *kilonova.ProblemList) []*kilonova.Problem {
	var id int
	if p.User != nil {
		id = p.User.ID
		if p.User.Admin {
			id = -1
		}
	}
	pbs, err := p.db.Problems(p.ctx, kilonova.ProblemFilter{IDs: list.List, LookingUserID: &id})
	if err != nil {
		return nil
	}
	return pbs
}

func (p *ProblemListParams) SubScore(pb *kilonova.Problem) string {
	score := p.db.MaxScore(p.ctx, p.User.ID, pb.ID)
	if score < 0 {
		return "-"
	}
	return strconv.Itoa(score)
}

type SubTaskEditParams struct {
	User    *kilonova.User
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask
	Topbar  *EditTopbar

	ctx context.Context
	db  kilonova.DB
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
	User    *kilonova.User
	Problem *kilonova.Problem
	Test    *kilonova.Test
	Topbar  *EditTopbar

	db kilonova.DB
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
	User *kilonova.User

	Version string
	Config  config.IndexConf

	ctx context.Context
	db  kilonova.DB
	r   kilonova.MarkdownRenderer
}

func (p *IndexParams) RenderMarkdown(body string) template.HTML {
	val, err := p.r.Render([]byte(body))
	if err != nil {
		return ""
	}
	return template.HTML(val)
}

func (p *IndexParams) NumSolved(ids []int) int {
	scores := p.db.MaxScores(p.ctx, p.User.ID, ids)
	var rez int
	for _, v := range scores {
		if v == 100 {
			rez++
		}
	}
	return rez
}

func (p *IndexParams) ProblemList(id int) *kilonova.ProblemList {
	list, err := p.db.ProblemList(p.ctx, id)
	if err != nil {
		return nil
	}
	return list
}

func (p *IndexParams) ListProblems(list *kilonova.ProblemList) []*kilonova.Problem {
	var id int
	if p.User != nil {
		id = p.User.ID
		if p.User.Admin {
			id = -1
		}
	}
	pbs, err := p.db.Problems(p.ctx, kilonova.ProblemFilter{IDs: list.List, LookingUserID: &id})
	if err != nil {
		return nil
	}
	return pbs
}

func (p *IndexParams) VisibleProblems() []*kilonova.Problem {
	problems, err := kilonova.VisibleProblems(p.ctx, p.User, p.db)
	if err != nil {
		return nil
	}
	return problems
}

func (p *IndexParams) SubScore(problem *kilonova.Problem, user *kilonova.User) string {
	score := p.db.MaxScore(p.ctx, user.ID, problem.ID)
	if score < 0 {
		return "-"
	}
	return strconv.Itoa(score)
}

type ProfileParams struct {
	User *kilonova.User

	ContentUser  *kilonova.User
	UserProblems []*kilonova.Problem
}

func Profile(w io.Writer, params *ProfileParams) error {
	err := profile.Execute(w, params)
	if err != nil {
		log.Println(err)
	}

	return err
}

type StatusParams struct {
	User *kilonova.User

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
	User     *kilonova.User
	Markdown template.HTML
	Title    string
}

type SimpleParams struct {
	User *kilonova.User
}

type AdminParams struct {
	User         *kilonova.User
	IndexDesc    string
	IndexListAll bool
	IndexLists   string
}

func AdminPanel(w io.Writer, user *kilonova.User) error {
	err := adminPanel.Execute(w, &AdminParams{user, config.Index.Description, config.Index.ShowProblems, kilonova.SerializeIntList(config.Index.Lists)})
	if err != nil {
		log.Println(err)
	}
	return err
}

func TestUI(w io.Writer, user *kilonova.User) error {
	err := testUI.Execute(w, &SimpleParams{user})
	if err != nil {
		log.Println(err)
	}
	return err
}

type VerifiedEmailParams struct {
	User *kilonova.User

	ContentUser *kilonova.User
}

func VerifiedEmail(w io.Writer, params *VerifiedEmailParams) error {
	err := verifiedEmail.Execute(w, params)
	if err != nil {
		log.Println(err)
	}

	return err
}

type SubParams struct {
	User       *kilonova.User
	Submission *kilonova.Submission
}

var _ http.Handler = &CDN{}

type CDN struct {
	CDN kilonova.CDNStore
}

func (s *CDN) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fpath := r.URL.Path
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	file, mod, err := s.CDN.GetFile(fpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		log.Println("Could not get file in CDN:", err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Add("cache-control", "public, max-age=86400, immutable")
	http.ServeContent(w, r, path.Base(fpath), mod, file)
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
	"hashedName": func(name string) string { return fsys.HashName(name) },
	"version":    func() string { return kilonova.Version },
	"debug":      func() bool { return config.Common.Debug },
	"intList":    kilonova.SerializeIntList,
	"httpstatus": http.StatusText,
	"dump":       spew.Sdump,
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
	return template.Must(t.ParseFS(templs, append([]string{"layout.html", "util/navbar.html"}, files...)...))
}
