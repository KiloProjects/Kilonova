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
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff like Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request, title string) templateData {
	// halfmoon stuff for when v1.2.0 is launched
	var shouldDarkMode bool
	darkModeCookie, err := r.Cookie("halfmoon_preferredMode")
	if err == nil {
		shouldDarkMode = darkModeCookie.Value == "dark-mode"
	}

	return templateData{
		Title: title,

		Params:   globParams(r),
		User:     util.User(r),
		LoggedIn: util.IsRAuthed(r),
		Version:  kilonova.Version,
		Debug:    rt.debug,
		DarkMode: shouldDarkMode,

		Languages: config.Languages,

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

func (rt *Web) getFullTestData(test *kilonova.Test) testDataType {
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

func (rt *Web) maxScore(userID int, problemID int) int {
	return rt.sserv.MaxScore(context.Background(), userID, problemID)
}

func gradient(score, maxscore int) template.CSS {
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
	return kilonova.Sanitizer.Sanitize(input)
}

func (rt *Web) newTemplate() *template.Template {
	var f fs.FS
	if rt.debug {
		f = os.DirFS("./web/templ")
	} else {
		f, _ = fs.Sub(templateDir, "templ")
	}

	return template.Must(parseAllTemplates(template.New("web").Funcs(template.FuncMap{
		"ispdflink":    isPdfLink,
		"dumpStruct":   spew.Sdump,
		"getFullTests": rt.getFullTestData,
		"hashedName": func(s string) string {
			return fsys.HashName(s)
		},
		"subStatus": func(status kilonova.Status) template.HTML {
			switch status {
			case kilonova.StatusWaiting:
				return template.HTML("În așteptare...")
			case kilonova.StatusWorking:
				return template.HTML("În lucru...")
			case kilonova.StatusFinished:
				return template.HTML("Finalizată")
			default:
				return template.HTML("Stare necunoscută")
			}
		},
		"KBtoMB": func(kb int) float64 {
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
		"subScore": func(problem *kilonova.Problem, user *kilonova.User) string {
			score := rt.sserv.MaxScore(context.Background(), user.ID, problem.ID)
			if score < 0 {
				return "-"
			}
			return fmt.Sprint(score)
		},
		"problemSubs": func(problem *kilonova.Problem, user *kilonova.User) []*kilonova.Submission {
			subs, err := rt.sserv.Submissions(context.Background(), kilonova.SubmissionFilter{UserID: &user.ID, ProblemID: &problem.ID})
			if err != nil {
				return nil
			}
			return subs
		},
		"problemTests": func(problem *kilonova.Problem) []*kilonova.Test {
			tests, err := rt.tserv.Tests(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return tests
		},
		"problemAuthor": func(problem *kilonova.Problem) *kilonova.User {
			user, err := rt.userv.UserByID(context.Background(), problem.AuthorID)
			if err != nil {
				return &kilonova.User{}
			}
			user.Password = ""
			return user
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
		"solvedProblems": func(user *kilonova.User) []*kilonova.Problem {
			ids, err := rt.sserv.SolvedProblems(context.Background(), user.ID)
			if err != nil {
				return nil
			}
			var pbs []*kilonova.Problem
			for _, id := range ids {
				pb, err := rt.pserv.ProblemByID(context.Background(), id)
				if err != nil {
					log.Println(err)
					return pbs
				}

				pbs = append(pbs, pb)
			}
			return pbs
		},
		"encodeJSON": func(data interface{}) (string, error) {
			d, err := json.Marshal(data)
			return base64.StdEncoding.EncodeToString(d), err
		},
	}), f))
}

func (rt *Web) build(w http.ResponseWriter, r *http.Request, name string, temp templateData) {
	if err := templates.ExecuteTemplate(w, name, temp); err != nil {
		fmt.Println(err)
		log.Printf("%s: %v\n", temp.OGUrl, err)
	}
}
