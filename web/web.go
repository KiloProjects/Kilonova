// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"bytes"
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
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/mdrenderer"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

var templates *template.Template

//go:embed static
var assets embed.FS

//go:embed templ
var templateDir embed.FS

// Web is the struct representing this whole package
type Web struct {
	dm    kilonova.DataStore
	rd    kilonova.MarkdownRenderer
	debug bool

	static fs.FS

	db     *db.DB
	mailer kilonova.Mailer

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
	//r.Mount("/static", hashfs.FileServer(fsys))

	r.Get("/", rt.index())
	r.With(mustBeAuthed).Get("/profile", rt.selfProfile())
	r.Get("/profile/{user}", rt.profile())
	r.Get("/settings", rt.justRender("settings.html"))

	r.Route("/problems", func(r chi.Router) {
		r.Get("/", rt.problems())
		r.Route("/{pbid}", func(r chi.Router) {
			r.Use(rt.ValidateProblemID)
			r.Use(rt.ValidateVisible)
			r.Get("/", rt.problem())
			r.Get("/attachments/{aid}", rt.problemAttachment)
			r.With(mustBeEditor).Route("/edit", rt.ProblemEditRouter)
		})
	})

	r.Route("/submissions", func(r chi.Router) {
		r.Get("/", rt.justRender("submissions.html"))
		r.With(rt.ValidateSubmissionID).Get("/{id}", rt.submission())
	})

	r.Route("/problem_lists", func(r chi.Router) {
		r.With(mustBeProposer).Get("/", rt.justRender("lists/index.html", "modals/pbs.html"))
		r.With(mustBeProposer).Get("/create", rt.justRender("lists/create.html"))
		r.With(rt.ValidateListID).Get("/{id}", rt.pbListView())
	})

	r.Mount("/docs", rt.docs())

	r.With(mustBeAdmin).Route("/admin", func(r chi.Router) {
		r.Get("/", rt.admin())
		r.Get("/makeKNA", rt.genKNA)
	})

	r.With(mustBeVisitor).Get("/login", rt.justRender("auth/login.html", "modals/login.html"))
	r.With(mustBeVisitor).Get("/signup", rt.justRender("auth/signup.html"))

	r.With(mustBeAuthed).Get("/logout", rt.logout)

	// Proposer panel
	r.Route("/proposer", func(r chi.Router) {
		r.Use(mustBeProposer)
		r.Get("/", rt.justRender("proposer/index.html", "proposer/createpb.html"))
		r.Route("/get", func(r chi.Router) {
			r.Get("/subtest_output/{st_id}", func(w http.ResponseWriter, r *http.Request) {
				id, err := strconv.Atoi(chi.URLParam(r, "st_id"))
				if err != nil {
					http.Error(w, "Bad ID", 400)
					return
				}
				subtest, err := rt.db.SubTest(r.Context(), id)
				if err != nil {
					http.Error(w, "Inexistent subtest", 400)
					return
				}
				sub, err := rt.db.Submission(r.Context(), subtest.SubmissionID)
				if err != nil {
					log.Println(err)
					http.Error(w, "Internal server error", 500)
					return
				}
				pb, err := rt.db.Problem(r.Context(), sub.ProblemID)
				if err != nil {
					log.Println(err)
					http.Error(w, "Internal server error", 500)
					return
				}
				if !util.IsProblemEditor(util.User(r), pb) {
					http.Error(w, "You aren't allowed to do that!", 401)
					return
				}
				rc, err := rt.dm.SubtestReader(subtest.ID)
				if err != nil {
					http.Error(w, "The subtest may have been purged as a routine data-saving process", 404)
					return
				}
				defer rc.Close()
				data, err := io.ReadAll(rc)
				if err != nil {
					http.Error(w, "Internal server error", 500)
					return
				}
				buf := bytes.NewReader(data)
				http.ServeContent(w, r, "subtest.out", time.Now(), buf)
			})
		})
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
func NewWeb(debug bool, db *db.DB, dm kilonova.DataStore, mailer kilonova.Mailer) *Web {
	rd := mdrenderer.NewLocalRenderer()
	funcs := template.FuncMap{
		"problemList": func(id int) *kilonova.ProblemList {
			list, err := db.ProblemList(context.Background(), id)
			if err != nil {
				return nil
			}
			return list
		},
		"visibleProblems": func(user *kilonova.User) []*kilonova.Problem {
			problems, err := db.VisibleProblems(context.Background(), user)
			if err != nil {
				return nil
			}
			return problems
		},
		"subScore": func(pb *kilonova.Problem, user *kilonova.User) string {
			score := db.MaxScore(context.Background(), user.ID, pb.ID)
			if score < 0 {
				return "-"
			}
			return strconv.Itoa(score)
		},
		"listProblems": func(user *kilonova.User, list *kilonova.ProblemList) []*kilonova.Problem {
			var id int
			if user != nil {
				id = user.ID
				if user.Admin {
					id = -1
				}
			}
			pbs, err := db.Problems(context.Background(), kilonova.ProblemFilter{IDs: list.List, LookingUserID: &id})
			if err != nil {
				return nil
			}
			return pbs
		},
		"renderMarkdown": func(body string) template.HTML {
			val, err := rd.Render([]byte(body))
			if err != nil {
				return ""
			}
			return template.HTML(val)
		},
		"genPbListParams": func(user *kilonova.User, lang string, pbs []*kilonova.Problem) *ProblemListingParams {
			return &ProblemListingParams{user, lang, pbs}
		},
		"numSolved": func(user *kilonova.User, ids []int) int {
			scores := db.MaxScores(context.Background(), user.ID, ids)
			var rez int
			for _, v := range scores {
				if v == 100 {
					rez++
				}
			}
			return rez
		},
		"problemLists": func() []*kilonova.ProblemList {
			list, err := db.ProblemLists(context.Background(), kilonova.ProblemListFilter{})
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

	return &Web{dm, rd, debug, static, db, mailer, funcs}
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
		http.Error(w, http.StatusText(403), 403)
		return
	}
	w.Header().Set("Cache-Control", `public, max-age=3600`)
	http.ServeContent(w, r, st.Name(), st.ModTime(), ff)
}
