// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/benbjohnson/hashfs"
	"github.com/davecgh/go-spew/spew"
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
	debug bool

	// db *db.DB

	funcs template.FuncMap

	base *sudoapi.BaseAPI
}

func (rt *Web) statusPage(w http.ResponseWriter, r *http.Request, statusCode int, errMessage string) {
	status := rt.parse(nil, "util/statusCode.html", "modals/login.html")
	rt.runTempl(w, r, status, &StatusParams{
		Ctx:     GenContext(r),
		Code:    statusCode,
		Message: errMessage,
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

	r.With(rt.ValidatePasteID).Get("/pastes/{id}", rt.paste())

	r.Route("/problem_lists", func(r chi.Router) {
		r.Get("/", rt.justRender("lists/index.html", "modals/pblist.html", "modals/pbs.html"))
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
	r.With(rt.mustBeVisitor).Get("/forgot_pwd", rt.justRender("auth/forgot_pwd_send.html"))

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

	// Password reset
	r.With(rt.mustBeVisitor).Get("/resetPassword/{reqid}", rt.resetPassword())

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := embedded.Open("static/robots.txt")
		if err != nil {
			zap.S().Warn("Could not open robots.txt")
			return
		}
		http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		rt.statusPage(w, r, 404, "")
	})

	return r
}

func (rt *Web) parse(optFuncs template.FuncMap, files ...string) *template.Template {
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
		"renderMarkdown": func(body interface{}) template.HTML {
			var bd []byte
			switch body.(type) {
			case string:
				bd = []byte(body.(string))
			case []byte:
				bd = body.([]byte)
			case template.HTML:
				bd = []byte(body.(template.HTML))
			default:
				panic("Unknown renderMarkdown type")
			}
			val, err := base.RenderMarkdown(bd)
			if err != nil {
				zap.S().Warn(err)
				return "[Error rendering markdown]"
			}
			return template.HTML(val)
		},
		"problemEditor": util.IsProblemEditor,
		"submissionEditor": func(user *kilonova.UserBrief, sub *kilonova.Submission) bool {
			return util.IsSubmissionEditor(sub, user)
		},
		"pasteEditor": func(user *kilonova.UserBrief, paste *kilonova.SubmissionPaste) bool {
			return util.IsPasteEditor(paste, user)
		},
		"problemVisible": util.IsProblemVisible,
		"genProblemsParams": func(scoreUser *kilonova.UserBrief, pbs []*kilonova.Problem, showSolved, multiCols bool) *ProblemListingParams {
			return &ProblemListingParams{pbs, showSolved, multiCols, scoreUser}
		},
		"genPblistParams": func(user *kilonova.UserBrief, ctx *ReqContext, pblist *kilonova.ProblemList, open bool) *PblistParams {
			return &PblistParams{user, ctx, pblist, open}
		},
		"numSolved": func(user *kilonova.UserBrief, ids []int) int {
			return base.NumSolved(context.Background(), user.ID, ids)
		},
		"numSolvedPbs": func(user *kilonova.UserBrief, pbs []*kilonova.Problem) int {
			ids := []int{}
			for _, pb := range pbs {
				ids = append(ids, pb.ID)
			}
			return base.NumSolved(context.Background(), user.ID, ids)
		},
		"user": func(uid int) *kilonova.UserBrief {
			user, err := base.UserBrief(context.Background(), uid)
			if err != nil {
				return nil
			}
			return user
		},
		"problemLists": func() []*kilonova.ProblemList {
			list, err := base.ProblemLists(context.Background(), kilonova.ProblemListFilter{Root: true})
			if err != nil {
				return nil
			}
			return list
		},

		"problemTests": func(problem *kilonova.Problem) []*kilonova.Test {
			tests, err := base.Tests(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return tests
		},
		"problemSubtasks": func(problem *kilonova.Problem) []*kilonova.SubTask {
			sts, err := base.SubTasks(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return sts
		},

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

		"signupEnabled": func() bool { return config.Features.Signup },
		"pastesEnabled": func() bool { return config.Features.Pastes },
		"graderEnabled": func() bool { return config.Features.Grader },
		"defaultLang":   func() string { return config.Common.DefaultLang },

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

		"getText": func(key string, vals ...any) string {
			zap.S().Error("Uninitialized `getText`")
			return "FATAL ERR"
		},
		"authed": func() bool {
			zap.S().Error("Uninitialized `authed`")
			return false
		},
		"authedUser": func() *kilonova.UserBrief {
			zap.S().Error("Uninitialized `authedUser`")
			return nil
		},
	}
	return &Web{debug, funcs, base}
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
