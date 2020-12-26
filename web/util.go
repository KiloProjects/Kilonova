package web

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/microcosm-cc/bluemonday"
)

// hydrateTemplate fills a templateData struct with generic stuff like Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request, title string) templateData {
	return templateData{
		Title: title,

		Params:   globParams(r),
		User:     util.User(r),
		LoggedIn: util.IsRAuthed(r),
		Version:  logic.Version,
		Debug:    rt.debug,

		Problem:    util.Problem(r),
		Submission: util.Submission(r),
		Test:       util.Test(r),

		ProblemID: util.ID(r, util.PbID),
		SubID:     util.ID(r, util.SubID),
		TestID:    util.ID(r, util.TestID),

		// HACK: Move this somewhere else
		ProblemEditor: util.IsRProblemEditor(r),
		SubEditor:     util.IsRSubmissionEditor(r),

		OGUrl: r.URL.RequestURI(),
	}
}

func globParams(r *http.Request) map[string]string {
	ctx := chi.RouteContext(r.Context())
	params := make(map[string]string)
	for i := 0; i < len(ctx.URLParams.Keys); i++ {
		params[ctx.URLParams.Keys[i]] = ctx.URLParams.Values[i]
	}
	return params
}

type testDataType struct {
	In  string
	Out string
}

func (rt *Web) getFullTestData(test *db.Test) testDataType {
	in, err := rt.dm.TestInput(test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := rt.dm.TestOutput(test.ID)
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

func (rt *Web) getTestData(test *db.Test) testDataType {
	t := rt.getFullTestData(test)
	if len(t.In) > 128*1024 { // 128KB
		t.In = "too long to show here"
	}
	if len(t.Out) > 128*1024 { // 128KB
		t.Out = "too long to show here"
	}
	return t
}

func (rt *Web) maxScore(userID int64, problemID int64) int {
	user, err := rt.kn.DB.User(context.Background(), userID)
	if err != nil {
		return 0
	}
	return user.MaxScore(problemID)
}

func gradient(score, maxscore int32) template.CSS {
	col := "#e3dd71" // a yellow hue, indicating something is wrong
	var rap float64
	if maxscore == 0 || score == 0 {
		rap = 0
	} else {
		rap = float64(score) / float64(maxscore)
	}
	if rap == 1.0 {
		col = "#7fff00"
	}

	if rap < 1.0 {
		col = "#67cf39"
	}

	if rap <= 0.8 {
		col = "#9fdd2e"
	}

	if rap <= 0.6 {
		col = "#d2eb19"
	}

	if rap <= 0.4 {
		col = "#f1d011"
	}

	if rap <= 0.2 {
		col = "#f6881e"
	}

	if rap == 0 {
		col = "#f11722"
	}

	return template.CSS(fmt.Sprintf("background-color: %s\n", col))
}

// isPdfLink does a simple analysis if it is a link, note that it does not check if the content is actually a PDF
// I should check it sometime, or use it as a db field for the problem
func isPdfLink(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return path.Ext(u.Path) == ".pdf"
}

func sanitize(input string) string {
	return bluemonday.UGCPolicy().Sanitize(input)
}

func (rt *Web) newTemplate() *template.Template {

	return template.Must(parseAllTemplates(template.New("web").Funcs(template.FuncMap{
		"ispdflink":    isPdfLink,
		"dumpStruct":   spew.Sdump,
		"getTestData":  rt.getTestData,
		"getFullTests": rt.getFullTestData,
		"subStatus": func(status db.Status) template.HTML {
			switch status {
			case db.StatusWaiting:
				return template.HTML("În așteptare...")
			case db.StatusWorking:
				return template.HTML("În lucru...")
			case db.StatusFinished:
				return template.HTML("Finalizată")
			default:
				return template.HTML("Stare necunoscută")
			}
		},
		"KBtoMB": func(kb int32) float64 {
			return float64(kb) / 1024.0
		},
		"gradient": gradient,
		"zeroto100": func() []int {
			var v []int = make([]int, 0)
			for i := 0; i <= 100; i++ {
				v = append(v, i)
			}
			return v
		},
		"subScore": func(problem *db.Problem, user *db.User) string {
			score := user.MaxScore(problem.ID)
			if score <= 0 {
				return "-"
			}
			return fmt.Sprint(score)
		},
		"problemSubs": func(problem *db.Problem, user *db.User) []*db.Submission {
			subs, err := user.ProblemSubs(problem.ID)
			if err != nil {
				return nil
			}
			return subs
		},
		"problemTests": func(problem *db.Problem) []*db.Test {
			tests, err := problem.Tests()
			if err != nil {
				return nil
			}
			return tests
		},
		"problemAuthor": func(problem *db.Problem) *db.User {
			user, err := problem.GetAuthor()
			if err != nil {
				return &db.User{}
			}
			user.Password = ""
			return user
		},
		"subAuthor": func(sub *db.Submission) *db.User {
			user, err := rt.kn.DB.User(context.Background(), sub.UserID)
			if err != nil {
				return &db.User{}
			}
			user.Password = ""
			return user
		},
		"subProblem": func(sub *db.Submission) *db.Problem {
			pb, err := rt.kn.DB.Problem(context.Background(), sub.ProblemID)
			if err != nil {
				return &db.Problem{}
			}
			return pb
		},
		"subTests": func(sub *db.Submission) []*db.SubTest {
			tests, err := rt.kn.DB.SubTests(context.Background(), sub.ID)
			if err != nil {
				return nil
			}
			return tests
		},
		"getTest": func(id int64) *db.Test {
			test, err := rt.kn.DB.TestByID(context.Background(), id)
			if err != nil {
				return &db.Test{}
			}
			return test
		},
		"timeToUnix": func(t time.Time) int64 {
			return t.Unix()
		},
		"sanitize": sanitize,
		"html": func(s string) template.HTML {
			return template.HTML(sanitize(s))
		},
		"unsafeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"solvedProblems": func(user *db.User) []*db.Problem {
			pbs, err := user.SolvedProblems()
			if err != nil {
				return nil
			}
			return pbs
		},
	}), root))
}

func (rt *Web) build(w http.ResponseWriter, r *http.Request, name string, temp templateData) {
	if err := templates.ExecuteTemplate(w, name, temp); err != nil {
		fmt.Println(err)
		log.Printf("%s: %v\n", temp.OGUrl, err)
	}
}
