// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"context"
	"embed"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/mdrenderer"
	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var templates *template.Template

//go:embed static
var embedded embed.FS

//go:embed templ
var templateDir embed.FS

var fsys = hashfs.NewFS(embedded)

// Web is the struct representing this whole package
type Web struct {
	rd    kilonova.MarkdownRenderer
	debug bool

	// db *db.DB

	funcs template.FuncMap

	base *sudoapi.BaseAPI
}

func (rt *Web) statusPage(w http.ResponseWriter, r *http.Request, statusCode int, err string, shouldLogin bool) {
	rt.Status(w, &StatusParams{
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

	r.Mount("/static", hashfs.FileServer(fsys))

	r.Get("/", rt.index())
	r.With(rt.mustBeAuthed).Get("/profile", rt.selfProfile())
	r.Get("/profile/{user}", rt.profile())
	r.Get("/settings", rt.justRender("settings.html"))

	r.Route("/problems", func(r chi.Router) {
		r.Get("/", rt.problems())
		r.Route("/{pbid}", func(r chi.Router) {
			r.Use(rt.ValidateProblemID)
			r.Use(rt.ValidateProblemVisible)
			r.Get("/", rt.problem())
			r.Get("/attachments/{aid}", rt.problemAttachment)
			r.With(rt.mustBeProblemEditor).Route("/edit", rt.ProblemEditRouter)
		})
	})

	r.Route("/submissions", func(r chi.Router) {
		r.Get("/", rt.justRender("submissions.html"))
		r.With(rt.ValidateSubmissionID).Get("/{id}", rt.submission())
	})

	r.Route("/problem_lists", func(r chi.Router) {
		r.Get("/", rt.justRender("lists/index.html", "modals/pbs.html"))
		r.With(rt.mustBeProposer).Get("/create", rt.justRender("lists/create.html"))
		r.With(rt.ValidateListID).Get("/{id}", rt.pbListView())
	})

	r.Mount("/docs", rt.docs())

	r.With(rt.mustBeAdmin).Route("/admin", func(r chi.Router) {
		r.Get("/", rt.justRender("admin/admin.html"))
		r.Get("/users", rt.justRender("admin/users.html"))
		r.Get("/auditLog", rt.auditLog())
	})

	r.With(rt.mustBeVisitor).Get("/login", rt.justRender("auth/login.html", "modals/login.html"))
	r.With(rt.mustBeVisitor).Get("/signup", rt.justRender("auth/signup.html"))

	r.With(rt.mustBeAuthed).Get("/logout", rt.logout)

	// Proposer panel
	r.Route("/proposer", func(r chi.Router) {
		r.Use(rt.mustBeProposer)
		r.Get("/", rt.justRender("proposer/index.html", "proposer/createpb.html"))
		r.Get("/get/subtest_output/{st_id}", rt.subtestOutput)
	})

	// Email verification
	r.Route("/verify", func(r chi.Router) {
		r.With(rt.mustBeAuthed).Get("/resend", rt.resendEmail())
		r.Get("/{vid}", rt.verifyEmail())
	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := embedded.Open("static/robots.txt")
		if err != nil {
			log.Println("Could not open robots.txt")
			return
		}
		http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
	})

	return r
}

func (rt *Web) parse(optFuncs template.FuncMap, files ...string) executor {
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
		"pLanguages": func() map[string]*WebLanguage {
			return webLanguages
		},
		"problemSettings": func(problemID int) *kilonova.ProblemEvalSettings {
			settings, err := base.ProblemSettings(context.Background(), problemID)
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
			problems, err := base.Problems(context.Background(), kilonova.ProblemFilter{LookingUser: user, Look: true})
			if err != nil {
				return nil
			}
			return problems
		},
		"unassociatedProblems": func(user *kilonova.UserBrief) []*kilonova.Problem {
			problems, err := base.Problems(context.Background(), kilonova.ProblemFilter{LookingUser: user, Look: true, Unassociated: true})
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
			pbs, err := base.ProblemListProblems(context.Background(), list.List, user)
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
		"genPbListParams": func(user *kilonova.UserBrief, lang string, pbs []*kilonova.Problem, showSolved bool) *ProblemListingParams {
			return &ProblemListingParams{user, lang, pbs, showSolved}
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
		"numSolvedPbs": func(user *kilonova.UserBrief, pbs []*kilonova.Problem) int {
			ids := []int{}
			for _, pb := range pbs {
				ids = append(ids, pb.ID)
			}
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
	return &Web{rd, debug, funcs, base}
}

var webLanguages map[string]*WebLanguage

type WebLanguage struct {
	Disabled bool   `json:"disabled"`
	Name     string `json:"name"`
	// Extensions []string `json:"extensions"`
}

func init() {
	webLanguages = make(map[string]*WebLanguage)
	for name, lang := range eval.Langs {
		webLanguages[name] = &WebLanguage{
			Disabled: lang.Disabled,
			Name:     lang.PrintableName,
			// Extensions: lang.Extensions,
		}
	}
}
