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
	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	settings = parse("settings.html")
	profile  = parse("profile.html")
	status   = parse("util/statusCode.html")
	index    = parse("index.html")

	pbs = parse("pbs.html")
	pb  = parse("pb.html")

	subs = parse("submissions.html")
	sub  = parse("submission.html")

	login  = parse("auth/login.html")
	signup = parse("auth/signup.html")

	editIndex   = parse("edit/index.html")
	editDesc    = parse("edit/desc.html")
	editChecker = parse("edit/checker.html")

	testAdd    = parse("edit/testAdd.html", "edit/testTopbar.html")
	testEdit   = parse("edit/testEdit.html", "edit/testTopbar.html")
	testScores = parse("edit/testScores.html", "edit/testTopbar.html")

	subtaskAdd   = parse("edit/subtaskAdd.html", "edit/subtaskTopbar.html")
	subtaskEdit  = parse("edit/subtaskEdit.html", "edit/subtaskTopbar.html")
	subtaskIndex = parse("edit/subtaskIndex.html", "edit/subtaskTopbar.html")

	pbListIndex  = parse("lists/index.html")
	pbListCreate = parse("lists/create.html")
	pbListView   = parse("lists/view.html")

	adminPanel = parse("admin/admin.html")
	knaPanel   = parse("admin/kna.html")
	testUI     = parse("admin/test-ui.html")

	adminUserPanel = parse("admin/users.html")

	markdown = parse("util/mdrender.html")

	proposerPanel = parse("proposer/index.html", "proposer/createpb.html", "proposer/cdn_manager.html")

	verifiedEmail = parse("verified-email.html")
	sentEmail     = parse("util/sent.html")

	listIndex  = parse("lists/index.html")
	listCreate = parse("lists/create.html")
)

type ProblemParams struct {
	User          *kilonova.User
	ProblemEditor bool

	Problem *kilonova.Problem
	Author  *kilonova.User

	Markdown  template.HTML
	Languages map[string]config.Language
}

type ProblemEditParams struct {
	User    *kilonova.User
	Problem *kilonova.Problem
}

type ProblemListParams struct {
	User        *kilonova.User
	ProblemList *kilonova.ProblemList

	ctx    context.Context
	plserv kilonova.ProblemListService
	pserv  kilonova.ProblemService
	sserv  kilonova.SubmissionService
	r      kilonova.MarkdownRenderer
}

func (p *ProblemListParams) RenderMarkdown(body string) template.HTML {
	val, err := p.r.Render([]byte(body))
	if err != nil {
		return ""
	}
	return template.HTML(val)
}

func (p *ProblemListParams) ProblemLists() []*kilonova.ProblemList {
	list, err := p.plserv.ProblemLists(p.ctx, kilonova.ProblemListFilter{})
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
	pbs, err := p.pserv.Problems(p.ctx, kilonova.ProblemFilter{IDs: list.List, LookingUserID: &id})
	if err != nil {
		return nil
	}
	return pbs
}

func (p *ProblemListParams) SubScore(pb *kilonova.Problem) string {
	score := p.sserv.MaxScore(p.ctx, p.User.ID, pb.ID)
	if score < 0 {
		return "-"
	}
	return strconv.Itoa(score)
}

type SubTaskEditParams struct {
	User    *kilonova.User
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask

	ctx    context.Context
	tserv  kilonova.TestService
	stserv kilonova.SubTaskService
}

func (s *SubTaskEditParams) ProblemTests() []*kilonova.Test {
	tests, err := s.tserv.Tests(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

func (s *SubTaskEditParams) ProblemSubTasks() []*kilonova.SubTask {
	sts, err := s.stserv.SubTasks(s.ctx, s.Problem.ID)
	if err != nil {
		return nil
	}
	return sts
}

func (s *SubTaskEditParams) TestSubTasks(id int) string {
	sts, err := s.stserv.SubTasksByTest(s.ctx, id)
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

	tserv kilonova.TestService
	dm    kilonova.DataStore
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
	tests, err := t.tserv.Tests(context.Background(), t.Problem.ID)
	if err != nil {
		return nil
	}
	return tests
}

type IndexParams struct {
	User *kilonova.User

	Version string
	Config  config.IndexConf

	ctx    context.Context
	sserv  kilonova.SubmissionService
	plserv kilonova.ProblemListService
	pserv  kilonova.ProblemService
	r      kilonova.MarkdownRenderer
}

func (p *IndexParams) RenderMarkdown(body string) template.HTML {
	val, err := p.r.Render([]byte(body))
	if err != nil {
		return ""
	}
	return template.HTML(val)
}

func (p *IndexParams) NumSolved(ids []int) int {
	scores := p.sserv.MaxScores(p.ctx, p.User.ID, ids)
	var rez int
	for _, v := range scores {
		if v == 100 {
			rez++
		}
	}
	return rez
}

func (p *IndexParams) ProblemList(id int) *kilonova.ProblemList {
	list, err := p.plserv.ProblemList(p.ctx, id)
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
	pbs, err := p.pserv.Problems(p.ctx, kilonova.ProblemFilter{IDs: list.List, LookingUserID: &id})
	if err != nil {
		return nil
	}
	return pbs
}

func (p *IndexParams) VisibleProblems() []*kilonova.Problem {
	problems, err := kilonova.VisibleProblems(p.ctx, p.User, p.pserv)
	if err != nil {
		return nil
	}
	return problems
}

func (p *IndexParams) SubScore(problem *kilonova.Problem, user *kilonova.User) string {
	score := p.sserv.MaxScore(p.ctx, user.ID, problem.ID)
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

	Code    int
	Message string
}

func Status(w io.Writer, params *StatusParams) error {
	err := status.Execute(w, params)
	if err != nil {
		log.Println(err)
	}

	return err
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
}

func parse(files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		log.Fatal(err)
	}
	return template.Must(template.New("layout.html").Funcs(funcs).ParseFS(templs, append([]string{"layout.html", "util/navbar.html"}, files...)...))
}
