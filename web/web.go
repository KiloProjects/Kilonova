// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"bytes"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/mdrenderer"
	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi"
)

var templates *template.Template

//go:embed static/*
var embedded embed.FS

//go:embed templ
var templateDir embed.FS

var fsys = hashfs.NewFS(embedded)

type templateData struct {
	Version  string
	Title    string
	Params   map[string]string
	User     *kilonova.User
	LoggedIn bool
	Debug    bool
	DarkMode bool

	// for the status code page
	Code  string
	Error string

	// For code submission page
	Languages map[string]config.Language

	// ProblemEditor tells us if the authed .User is able to edit the .Problem
	ProblemEditor bool

	// SubEditor tells us if the authed .User is able to change visibility of the .Submission
	SubEditor bool

	// Page-specific data
	// it is easier to just put this stuff here instead of in a `Data` interface
	Problems []*kilonova.Problem

	Problem   *kilonova.Problem
	ProblemID int

	// for problem page
	Markdown         template.HTML
	IsPdfDescription bool

	ContentUser *kilonova.User
	IsCUser     bool

	Submissions []*kilonova.Submission

	Submission *kilonova.Submission
	SubID      int

	Test   *kilonova.Test
	TestID int

	// Since codemirror and vue are particulairly big libraries, we should load them only when needed
	Codemirror bool
	Vue        bool

	Sidebar bool

	// OpenGraph stuff
	OGTitle string
	OGType  string
	OGUrl   string
	OGImage string
	OGDesc  string
}

// Web is the struct representing this whole package
type Web struct {
	kn    *logic.Kilonova
	dm    kilonova.DataStore
	rd    kilonova.MarkdownRenderer
	debug bool

	userv  kilonova.UserService
	sserv  kilonova.SubmissionService
	pserv  kilonova.ProblemService
	tserv  kilonova.TestService
	stserv kilonova.SubTestService
}

func (rt *Web) status(w http.ResponseWriter, r *http.Request, statusCode int, err string) {
	code := fmt.Sprintf("%d: %s", statusCode, http.StatusText(statusCode))
	templ := rt.hydrateTemplate(r, code)
	templ.Code = code
	templ.Error = err

	w.WriteHeader(statusCode)
	rt.build(w, r, "util/statusCode", templ)
}

func (rt *Web) notFound(w http.ResponseWriter, r *http.Request) {
	rt.status(w, r, 404, "")
}

// Handler returns a http.Handler
// TODO: Split routes in functions
func (rt *Web) Handler() http.Handler {
	r := chi.NewRouter()

	templates = rt.newTemplate()

	if rt.debug {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				templates = rt.newTemplate()
				next.ServeHTTP(w, r)
			})
		})
	}

	r.Mount("/static", hashfs.FileServer(fsys))

	r.Group(func(r chi.Router) {
		r.Use(rt.getUser)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var problems []*kilonova.Problem
			var err error
			if util.User(r) != nil && util.User(r).Admin {
				problems, err = rt.pserv.Problems(r.Context(), kilonova.ProblemFilter{})
			} else {
				var uid int
				if util.User(r) != nil {
					uid = util.User(r).ID
				}
				problems, err = rt.pserv.Problems(r.Context(), kilonova.ProblemFilter{LookingUserID: &uid})
			}
			if err != nil {
				log.Println("/:", err)
				rt.status(w, r, 500, "")
				return
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Println("/", err)
				rt.status(w, r, 500, "")
				return
			}
			templ := rt.hydrateTemplate(r, "")
			templ.Problems = problems
			rt.build(w, r, "index", templ)
		})

		r.Route("/profile", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/", func(w http.ResponseWriter, r *http.Request) {
				user := util.User(r)

				templ := rt.hydrateTemplate(r, fmt.Sprintf("Profil %s", user.Name))
				templ.ContentUser = user
				templ.IsCUser = true
				rt.build(w, r, "profile", templ)
			})
			r.Route("/{user}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					name := strings.TrimSpace(chi.URLParam(r, "user"))
					users, err := rt.userv.Users(r.Context(), kilonova.UserFilter{Name: &name})
					if err != nil || len(users) == 0 {
						if errors.Is(err, sql.ErrNoRows) || len(users) == 0 {
							rt.status(w, r, 404, "")
							return
						}
						fmt.Println(err)
						rt.status(w, r, 500, "")
						return
					}
					user := users[0]
					templ := rt.hydrateTemplate(r, fmt.Sprintf("Profil %s", user.Name))
					templ.ContentUser = user
					rt.build(w, r, "profile", templ)
				})
			})
		})

		r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r, "Setări")
			rt.build(w, r, "settings", templ)
		})

		r.Get("/changelog", func(w http.ResponseWriter, r *http.Request) {
			file, err := kilonova.Docs.Open("docs/CHANGELOG.md")
			if err != nil {
				log.Println("CAN'T OPEN CHANGELOG")
				rt.status(w, r, 500, "Can't load changelog")
				return
			}
			changelog, _ := io.ReadAll(file)
			ch, err := rt.rd.Render(changelog)
			if err != nil {
				log.Println("CAN'T RENDER CHANGELOG")
				rt.status(w, r, 500, "Can't render changelog")
				return
			}

			templ := rt.hydrateTemplate(r, "Changelog")
			templ.Markdown = template.HTML(ch)
			rt.build(w, r, "util/mdrender", templ)
		})

		r.Get("/todo", func(w http.ResponseWriter, r *http.Request) {
			file, err := kilonova.Docs.Open("docs/TODO.md")
			if err != nil {
				log.Println("CAN'T OPEN TODO")
				rt.status(w, r, 500, "Can't load todo list")
				return
			}

			todo, _ := io.ReadAll(file)
			t, err := rt.rd.Render(todo)
			if err != nil {
				log.Println("CAN'T RENDER TODO")
				rt.status(w, r, 500, "Can't render todo list")
				return
			}

			templ := rt.hydrateTemplate(r, "Todo list")
			templ.Markdown = template.HTML(t)
			rt.build(w, r, "util/mdrender", templ)
		})

		r.Get("/about", func(w http.ResponseWriter, r *http.Request) {
			file, err := kilonova.Docs.Open("docs/ABOUT.md")
			if err != nil {
				log.Println("CAN'T OPEN ABOUT")
				rt.status(w, r, 500, "Can't load About page")
				return
			}

			about, _ := io.ReadAll(file)
			t, err := rt.rd.Render(about)
			if err != nil {
				log.Println("CAN'T RENDER ABOUT")
				rt.status(w, r, 500, "Can't render About page")
				return
			}

			templ := rt.hydrateTemplate(r, "To do list")
			templ.Markdown = template.HTML(t)
			rt.build(w, r, "util/mdrender", templ)
		})

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				var problems []*kilonova.Problem
				var err error
				if util.User(r).Admin {
					problems, err = rt.pserv.Problems(r.Context(), kilonova.ProblemFilter{})
				} else {
					problems, err = rt.pserv.Problems(r.Context(), kilonova.ProblemFilter{LookingUserID: &util.User(r).ID})
				}
				if err != nil {
					fmt.Println(err)
					rt.status(w, r, 500, "")
					return
				}
				templ := rt.hydrateTemplate(r, "Probleme")
				templ.Problems = problems
				rt.build(w, r, "pbs", templ)
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := util.Problem(r)

					buf, err := rt.rd.Render([]byte(problem.Description))
					if err != nil {
						log.Println(err)
					}
					templ := rt.hydrateTemplate(r, fmt.Sprintf("Problema #%d: %s", problem.ID, problem.Name))
					templ.Codemirror = true
					templ.Vue = true
					templ.Markdown = template.HTML(buf)
					templ.OGDesc = problem.ShortDesc
					rt.build(w, r, "pb", templ)
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDIT | Problema #%d: %s", problem.ID, problem.Name))
						templ.Vue = true
						rt.build(w, r, "edit/index", templ)
					})
					r.Get("/desc", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE ENUNȚ | Problema #%d: %s", problem.ID, problem.Name))
						templ.Codemirror = true
						rt.build(w, r, "edit/desc", templ)
					})
					r.Get("/checker", func(w http.ResponseWriter, r *http.Request) {
						problem := util.Problem(r)
						templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE CHECKER | Problema #%d: %s", problem.ID, problem.Name))
						templ.Codemirror = true
						rt.build(w, r, "edit/checker", templ)
					})
					r.Route("/test", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							problem := util.Problem(r)
							templ := rt.hydrateTemplate(r, fmt.Sprintf("CREARE TEST | Problema #%d: %s", problem.ID, problem.Name))
							templ.Sidebar = true
							templ.Codemirror = true
							rt.build(w, r, "edit/testAdd", templ)
						})
						r.With(rt.ValidateTestID).Get("/{tid}", func(w http.ResponseWriter, r *http.Request) {
							test := util.Test(r)
							problem := util.Problem(r)
							templ := rt.hydrateTemplate(r, fmt.Sprintf("EDITARE TESTUL %d | Problema #%d: %s", test.VisibleID, problem.ID, problem.Name))
							templ.Sidebar = true
							templ.Codemirror = true
							rt.build(w, r, "edit/testEdit", templ)
						})
					})
				})
			})
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, "Submisii")
				templ.Vue = true
				rt.build(w, r, "submissions", templ)
			})
			r.With(rt.ValidateSubmissionID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, fmt.Sprintf("Submisia %d", util.Submission(r).ID))
				rt.build(w, r, "submission", templ)
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", rt.simpleTempl("Panoul de administrare", "admin/admin"))
		r.With(rt.mustBeAdmin).Get("/uitest", rt.simpleTempl("Testare UI", "test-ui"))

		r.With(rt.mustBeVisitor).Get("/login", rt.simpleTempl("Log In", "auth/login"))
		r.With(rt.mustBeVisitor).Get("/signup", rt.simpleTempl("Înregistrare", "auth/signup"))

		r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			// i could redirect to /api/auth/logout, but it's easier to do it like this
			rt.kn.RemoveSessionCookie(w, r)
			http.Redirect(w, r, "/", http.StatusFound)
		})

		// Proposer panel
		r.Route("/proposer", func(r chi.Router) {
			r.Use(rt.mustBeProposer)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r, "Panoul propunătorului")
				templ.Vue = true
				rt.build(w, r, "proposer/index", templ)
			})
			r.Route("/get", func(r chi.Router) {
				r.Get("/subtest_output/{st_id}", func(w http.ResponseWriter, r *http.Request) {
					id, err := strconv.Atoi(chi.URLParam(r, "st_id"))
					if err != nil {
						http.Error(w, "Bad ID", 400)
						return
					}
					subtest, err := rt.stserv.SubTest(r.Context(), id)
					if err != nil {
						http.Error(w, "Inexistent subtest", 400)
						return
					}
					sub, err := rt.sserv.SubmissionByID(r.Context(), subtest.SubmissionID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					pb, err := rt.pserv.ProblemByID(r.Context(), sub.ProblemID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					if !util.IsProblemEditor(util.User(r), pb) {
						http.Error(w, "You aren't allowed to do that!", 401)
						return
					}
					rc, err := rt.kn.DM.SubtestReader(subtest.ID)
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
			r.With(rt.mustBeAuthed).Get("/resend", func(w http.ResponseWriter, r *http.Request) {
				u := util.User(r)
				if u.VerifiedEmail {
					rt.status(w, r, http.StatusForbidden, "Deja ai verificat email-ul!")
					return
				}
				t := time.Now().Sub(u.EmailVerifSentAt.Time)
				if t < 5*time.Minute {
					text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", t)
					rt.status(w, r, http.StatusForbidden, text)
					return
				}
				if err := rt.kn.SendVerificationEmail(u.Email, u.Name, u.ID); err != nil {
					log.Println(err)
					rt.status(w, r, 500, "N-am putut retrimite email-ul de verificare")
					return
				}

				now := time.Now()
				if err := rt.userv.UpdateUser(r.Context(), u.ID, kilonova.UserUpdate{EmailVerifSentAt: &now}); err != nil {
					log.Println("Couldn't update verification email timestamp:", err)
				}
				rt.simpleTempl("Email retrimis", "util/sent")(w, r)
			})
			r.Get("/{vid}", func(w http.ResponseWriter, r *http.Request) {
				vid := chi.URLParam(r, "vid")
				if !rt.kn.CheckVerificationEmail(vid) {
					rt.notFound(w, r)
					return
				}

				uid, err := rt.kn.Verif.GetVerification(r.Context(), vid)
				if err != nil {
					log.Println(err)
					rt.notFound(w, r)
					return
				}

				user, err := rt.userv.UserByID(r.Context(), uid)
				if err != nil {
					log.Println(err)
					rt.notFound(w, r)
					return
				}

				// Do this to disable the popup
				if user != nil && user.ID == util.User(r).ID {
					util.User(r).VerifiedEmail = true
				}

				if err := rt.kn.ConfirmVerificationEmail(vid, user); err != nil {
					log.Println(err)
					rt.notFound(w, r)
					return
				}

				templ := rt.hydrateTemplate(r, "E-mail verificat")
				templ.ContentUser = user
				rt.build(w, r, "verified-email", templ)
			})
		})

	})

	r.NotFound(rt.notFound)

	return r
}

func (rt *Web) simpleTempl(title, templName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		templ := rt.hydrateTemplate(r, title)
		rt.build(w, r, templName, templ)
	}
}

// NewWeb returns a new web instance
func NewWeb(kn *logic.Kilonova, ts kilonova.TypeServicer) *Web {
	rd := mdrenderer.NewLocalRenderer()
	//rd := mdrenderer.NewExternalRenderer("http://0.0.0.0:8040")
	return &Web{kn, kn.DM, rd, kn.Debug,
		ts.UserService(), ts.SubmissionService(), ts.ProblemService(), ts.TestService(), ts.SubTestService()}
}
