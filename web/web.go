// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/markbates/pkger"
	"gorm.io/gorm"
)

var templates *template.Template

type templateData struct {
	Title    string
	Params   map[string]string
	User     common.User
	LoggedIn bool

	// Page-specific data
	// it is easier to just put this stuff here instead of in a `Data` interface
	Problems []common.Problem
	Problem  common.Problem

	ContentUser common.User

	Tasks []common.Task

	Task   common.Task
	TaskID uint

	ProblemID uint

	Version string

	Test   common.Test
	TestID uint

	// ProblemEditor tells us if the authed .User is able to edit the .Problem
	ProblemEditor bool

	// TaskEditor tells us if the authed .User is able to change visibility of the .Task
	TaskEditor bool

	// this is to not add a <br> in the test editor
	IsInTestEditor bool
}

// Web is the struct representing this whole package
type Web struct {
	dm     datamanager.Manager
	db     *kndb.DB
	logger *log.Logger
}

func pushStuff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pusher, ok := w.(http.Pusher); ok {
			if err := pusher.Push("/static/jquery.js", nil); err != nil {
				log.Printf("Failed to push: %v", err)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// GetRouter returns a chi.Router
// TODO: Split routes in functions
func (rt *Web) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	// table for gradient, initialize here so it panics if we make a mistake
	colorTable := gTable{
		{mustParseHex("#f11722"), 0.0},
		{mustParseHex("#eaf200"), 0.5},
		{mustParseHex("#64ce3a"), 1.0},
	}

	templates = template.New("web").Funcs(template.FuncMap{
		"dumpStruct":   spew.Sdump,
		"getTestData":  rt.getTestData,
		"getFullTests": rt.getFullTestData,
		"taskStatus": func(id int) template.HTML {
			switch id {
			case common.StatusWaiting:
				return template.HTML("În așteptare...")
			case common.StatusWorking:
				return template.HTML("În lucru...")
			case common.StatusDone:
				return template.HTML("Finalizată")
			default:
				return template.HTML("Stare necunoscută")
			}
		},
		"KBtoMB": func(kb uint64) float64 {
			return float64(kb) / 1024.0
		},
		"gradient": func(score, maxscore int) template.CSS {
			return gradient(score, maxscore, colorTable)
		},
		"zeroto100": func() []int {
			var v []int = make([]int, 0)
			for i := 0; i <= 100; i++ {
				v = append(v, i)
			}
			return v
		},
		"taskScore": func(problem common.Problem, user common.User) string {
			score, err := rt.db.MaxScoreFor(user.ID, problem.ID)
			if err != nil || score < 0 {
				return "-"
			}
			return fmt.Sprint(score)
		},
		"problemTasks": func(problem common.Problem, user common.User) []common.Task {
			tasks, err := rt.db.UserTasksOnProblem(user.ID, problem.ID)
			if err != nil {
				return nil
			}
			return tasks
		},
	})
	templates = template.Must(parseAllTemplates(templates, root))

	r.Mount("/static", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.RequestURI)
		if !strings.HasPrefix(p, "/static") {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		file, err := pkger.Open(p)
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	}))

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		file, err := pkger.Open("/static/favicon.ico")
		if err != nil {
			rt.logger.Println("CAN'T OPEN FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			rt.logger.Println("CAN'T STAT FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	// Enable server push
	r.With(rt.getUser).With(pushStuff).Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			problems, err := rt.db.GetAllVisibleProblems(common.UserFromContext(r))
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				rt.logger.Println("/", err)
				http.Error(w, http.StatusText(500), 500)
				return
			}
			templ := rt.hydrateTemplate(r)
			templ.Problems = problems
			rt.check(templates.ExecuteTemplate(w, "index", templ))
		})

		r.Route("/probleme", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problems, err := rt.db.GetAllVisibleProblems(common.UserFromContext(r))
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					rt.logger.Println("/probleme/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Probleme"
				templ.Problems = problems
				rt.check(templates.ExecuteTemplate(w, "probleme", templ))
			})
			r.With(rt.mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = "Creare problemă"
				rt.check(templates.ExecuteTemplate(w, "createpb", templ))
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)

					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)
					rt.check(templates.ExecuteTemplate(w, "problema", templ))
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
						rt.check(templates.ExecuteTemplate(w, "edit/index", templ))
					})
					r.Get("/enunt", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("ENUNT - EDIT | #%d: %s", problem.ID, problem.Name)
						rt.check(templates.ExecuteTemplate(w, "edit/enunt", templ))
					})
					r.Get("/limite", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("LIMITE - EDIT | #%d: %s", problem.ID, problem.Name)
						rt.check(templates.ExecuteTemplate(w, "edit/limite", templ))
					})
					r.Route("/teste", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							problem := common.ProblemFromContext(r)
							templ := rt.hydrateTemplate(r)
							templ.IsInTestEditor = true
							templ.Title = fmt.Sprintf("TESTE - EDIT | #%d: %s", problem.ID, problem.Name)
							rt.check(templates.ExecuteTemplate(w, "edit/testAdd", templ))
						})
						r.With(rt.ValidateTestID).Get("/{tid}", func(w http.ResponseWriter, r *http.Request) {
							test := common.TestFromContext(r)
							problem := common.ProblemFromContext(r)
							templ := rt.hydrateTemplate(r)
							templ.IsInTestEditor = true
							templ.Title = fmt.Sprintf("Teste - EDIT %d | #%d: %s", test.VisibleID, problem.ID, problem.Name)
							rt.check(templates.ExecuteTemplate(w, "edit/testEdit", templ))
						})
					})
				})
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				tasks, err := rt.db.GetAllTasks()
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					rt.logger.Println("/tasks/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Tasks"
				templ.Tasks = tasks
				rt.check(templates.ExecuteTemplate(w, "tasks", templ))
			})
			r.With(rt.ValidateTaskID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("Task %d", templ.Task.ID)
				rt.check(templates.ExecuteTemplate(w, "task", templ))
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Admin switches"
			rt.check(templates.ExecuteTemplate(w, "admin", templ))
		})

		r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Log In"
			rt.check(templates.ExecuteTemplate(w, "login", templ))
		})
		r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Sign Up"
			rt.check(templates.ExecuteTemplate(w, "signup", templ))
		})

		r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			// i could redirect to /api/auth/logout, but it's easier to do it like this
			common.RemoveSessionCookie(w)
			http.Redirect(w, r, "/", http.StatusFound)
		})
	})

	return r
}

// NewWeb returns a new web instance
func NewWeb(dm datamanager.Manager, db *kndb.DB, logger *log.Logger) *Web {
	return &Web{dm, db, logger}
}

func (rt *Web) check(err error) {
	if err != nil {
		rt.logger.Println(err)
	}
}

func init() {
	pkger.Include("/include")
	pkger.Include("/static")
}
