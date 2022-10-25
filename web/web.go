// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/mdrenderer"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

//go:embed static
var assets embed.FS

//go:embed templ
var templateDir embed.FS

// Web is the struct representing this whole package
type Web struct {
	rd    kilonova.MarkdownRenderer
	debug bool

	static fs.FS

	base *sudoapi.BaseAPI

	funcs template.FuncMap
}

func statusPage(w http.ResponseWriter, r *http.Request, statusCode int, err string, shouldLogin bool) {
	Status(w, &StatusParams{
		Ctx:         GenContext(r),
		Code:        statusCode,
		Message:     err,
		ShouldLogin: shouldLogin,
	})
}

// Handler returns a http.Handler
// TODO: Split routes in functions
func (rt *Web) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(rt.initSession)
	r.Use(rt.initLanguage)

	r.Mount("/static", http.HandlerFunc(rt.staticHandler))

	r.Get("/print", func(w http.ResponseWriter, r *http.Request) {
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	})
	r.Get("/", rt.index())
	r.With(mustBeAuthed).Get("/profile", rt.selfProfile())
	r.Get("/profile/{user}", rt.profile())
	r.Get("/settings", rt.justRender("settings.html"))

	r.Route("/problems", func(r chi.Router) {
		r.Get("/", rt.problems())
		r.Route("/{pbid}", func(r chi.Router) {
			r.Use(rt.ValidateProblemID)
			r.Use(rt.ValidateProblemVisible)
			r.Get("/", rt.problem())
			r.Get("/attachments/{aid}", rt.problemAttachment)
			r.With(mustBeProblemEditor).Route("/edit", rt.ProblemEditRouter)
		})
	})

	r.Route("/submissions", func(r chi.Router) {
		r.Get("/", rt.justRender("submissions.html"))
		r.With(rt.ValidateSubmissionID).Get("/{id}", rt.submission())
	})

	r.Route("/contests", func(r chi.Router) {
		//r.Get("/", rt.contestIndex())
		r.Route("/{ctid}", func(r chi.Router) {
			//r.Use(rt.ValidateContestID)
			//r.Use(rt.ValidateContestVisible)
			//r.Get("/", rt.contest())
			//r.Get("/pbid", rt.contestProblem())
			//r.With(validateContestEditor).Route("/edit", rt.contestEditRouter)
		})
	})

	r.Route("/problem_lists", func(r chi.Router) {
		r.With(mustBeProposer).Get("/", rt.justRender("lists/index.html", "modals/pbs.html"))
		r.With(mustBeProposer).Get("/create", rt.justRender("lists/create.html"))
		r.With(rt.ValidateListID).Get("/{id}", rt.pbListView())
	})

	r.Mount("/docs", rt.docs())

	r.With(mustBeAdmin).Route("/admin", func(r chi.Router) {
		r.Get("/", rt.admin())
		r.Get("/users", rt.justRender("admin/users.html"))
	})

	r.With(mustBeVisitor).Get("/login", rt.justRender("auth/login.html", "modals/login.html"))
	r.With(mustBeVisitor).Get("/signup", rt.justRender("auth/signup.html"))

	r.With(mustBeAuthed).Get("/logout", rt.logout)

	// Proposer panel
	r.Route("/proposer", func(r chi.Router) {
		r.Use(mustBeProposer)
		r.Get("/", rt.justRender("proposer/index.html", "proposer/createpb.html"))
		r.Get("/get/subtest_output/{st_id}", rt.subtestOutput)
	})

	// Email verification
	r.Route("/verify", func(r chi.Router) {
		r.With(mustBeAuthed).Get("/resend", rt.resendEmail())
		r.Get("/{vid}", rt.verifyEmail())
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := assets.Open("static/robots.txt")
		if err != nil {
			log.Println("Could not open robots.txt")
			return
		}
		http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Status(w, &StatusParams{GenContext(r), 404, "", false})
	})

	return r
}

func (rt *Web) parse(optFuncs template.FuncMap, files ...string) executor { //*template.Template {
	if optFuncs == nil {
		return parse(rt.funcs, files...)
	}
	for k, v := range rt.funcs {
		optFuncs[k] = v
	}
	return parse(optFuncs, files...)
}

// NewWeb returns a new web instance
func NewWeb(debug bool, base *sudoapi.BaseAPI) *Web {
	rd := mdrenderer.NewLocalRenderer()
	funcs := template.FuncMap{
		"pLanguages": func() map[string]eval.Language {
			return eval.Langs
		},
		"problemSettings": func(problemID int) *kilonova.ProblemEvalSettings {
			settings, err := base.GetProblemSettings(context.Background(), problemID)
			if err != nil {
				zap.S().Warn(err)
				return nil
			}
			return settings
		},
		"problemList": func(id int) *kilonova.ProblemList {
			list, err := base.ProblemList(context.Background(), id)
			if err != nil {
				return nil
			}
			return list
		},
		"visibleProblems": func(user *kilonova.UserBrief) []*kilonova.Problem {
			problems, err := base.Problems(context.Background(), kilonova.ProblemFilter{LookingUser: user})
			if err != nil {
				return nil
			}
			return problems
		},
		"subScore": func(pb *kilonova.Problem, user *kilonova.UserBrief) string {
			score := base.MaxScore(context.Background(), user.ID, pb.ID)
			if score < 0 {
				return "-"
			}
			return strconv.Itoa(score)
		},
		"listProblems": func(user *kilonova.UserBrief, list *kilonova.ProblemList) []*kilonova.Problem {
			pbs, err := base.Problems(context.Background(), kilonova.ProblemFilter{IDs: list.List, LookingUser: user})
			if err != nil {
				return nil
			}
			return pbs
		},
		"renderMarkdown": func(body string) template.HTML {
			val, err := rd.Render([]byte(body))
			if err != nil {
				zap.S().Warn(err)
				return "[Error rendering markdown]"
			}
			return template.HTML(val)
		},
		"genPbListParams": func(user *kilonova.UserBrief, lang string, pbs []*kilonova.Problem) *ProblemListingParams {
			return &ProblemListingParams{user, lang, pbs}
		},
		"numSolved": func(user *kilonova.UserBrief, ids []int) int {
			scores := base.MaxScores(context.Background(), user.ID, ids)
			var rez int
			for _, v := range scores {
				if v == 100 {
					rez++
				}
			}
			return rez
		},
		"problemLists": func() []*kilonova.ProblemList {
			list, err := base.ProblemLists(context.Background(), kilonova.ProblemListFilter{})
			if err != nil {
				return nil
			}
			return list
		},
	}

	var static fs.FS = assets
	if debug {
		static = os.DirFS("web")
	}

	return &Web{rd, debug, static, base, funcs}
}

func (rt *Web) staticHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path
	if filename == "/" {
		filename = "."
	} else {
		filename = strings.TrimPrefix(filename, "/")
	}
	filename = path.Clean(filename)
	f, err := rt.static.Open(filename)
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, http.StatusText(404), 404)
		return
	} else if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer f.Close()

	ff, ok := f.(io.ReadSeeker)
	if !ok {
		zap.S().Warn("Static file is not io.ReadSeeker")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	st, err := f.Stat()
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	if st.IsDir() {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", `public, max-age=3600`)
	http.ServeContent(w, r, st.Name(), st.ModTime(), ff)
}
