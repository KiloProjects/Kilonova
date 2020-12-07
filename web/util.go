package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/KiloProjects/Kilonova/internal/version"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff like Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request, title string) templateData {
	return templateData{
		Title: title,

		Params:   globParams(r),
		User:     util.User(r),
		LoggedIn: util.IsRAuthed(r),
		Version:  version.Version,
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
	in, out, err := rt.dm.Test(test.ProblemID, test.VisibleID)
	if err != nil {
		in = []byte("err")
		out = []byte("err")
	}
	return testDataType{In: string(in), Out: string(out)}
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

func (rt *Web) newTemplate() *template.Template {
	// table for gradient, initialize here so it panics if we make a mistake
	colorTable := gTable{
		{mustParseHex("#f45d64"), 0.0},
		{mustParseHex("#eaf200"), 0.5},
		{mustParseHex("#64ce3a"), 1.0},
	}

	return template.Must(parseAllTemplates(template.New("web").Funcs(template.FuncMap{
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
		"gradient": func(score, maxscore int32) template.CSS {
			return gradient(int(score), int(maxscore), colorTable)
		},
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
	}), root))
}

func (rt *Web) build(w http.ResponseWriter, r *http.Request, name string, temp templateData) {
	if err := templates.ExecuteTemplate(w, name, temp); err != nil {
		fmt.Println(err)
		log.Printf("%s: %v\n", temp.OGUrl, err)
	}
}
