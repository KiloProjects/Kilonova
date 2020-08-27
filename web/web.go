// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"encoding/json"
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
	Task  common.Task

	ProblemID uint

	Version string

	// ProblemEditor tells us if the authed .User is able to edit the .Problem
	ProblemEditor bool

	// TaskEditor tells us if the authed .User is able to change visibility of the .Task
	TaskEditor bool
}

// Web is the struct representing this whole package
type Web struct {
	dm datamanager.Manager
	db *kndb.DB
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
	colorTable := gradientTable{
		{mustParseHex("#ff8279"), 0.0},
		{mustParseHex("#eaf200"), 0.45},
		{mustParseHex("#00933e"), 1.0},
	}

	templates = template.New("web").Funcs(template.FuncMap{
		"dumpStruct": spew.Sdump,
		"dumpAsJson": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "    ")
			return string(b)
		},
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
			fmt.Println("CAN'T OPEN FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		fstat, err := file.Stat()
		if err != nil {
			fmt.Println("CAN'T STAT FAVICON")
			http.Error(w, http.StatusText(500), 500)
			return
		}
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	// Optimization: get the user only on routes that need to
	// Also enable server push
	r.With(rt.getUser).With(pushStuff).Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			problems, err := rt.db.GetAllVisibleProblems(common.UserFromContext(r))
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				fmt.Println("/", err)
				http.Error(w, http.StatusText(500), 500)
				return
			}
			templ := rt.hydrateTemplate(r)
			templ.Problems = problems
			if err := templates.ExecuteTemplate(w, "index", templ); err != nil {
				fmt.Println(err)
			}
		})

		r.Route("/probleme", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problems, err := rt.db.GetAllVisibleProblems(common.UserFromContext(r))
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("/probleme/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Probleme"
				templ.Problems = problems
				if err := templates.ExecuteTemplate(w, "probleme", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.With(rt.mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = "Creare problemă"
				if err := templates.ExecuteTemplate(w, "createpb", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)

					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)

					if err := templates.ExecuteTemplate(w, "problema", templ); err != nil {
						fmt.Println(err)
					}
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
						if err := templates.ExecuteTemplate(w, "edit/index", templ); err != nil {
							fmt.Println(err)
						}
					})
					r.Get("/enunt", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("ENUNT - EDIT | #%d: %s", problem.ID, problem.Name)
						if err := templates.ExecuteTemplate(w, "edit/enunt", templ); err != nil {
							fmt.Println(err)
						}
					})
					r.Get("/limite", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("LIMITE - EDIT | #%d: %s", problem.ID, problem.Name)
						if err := templates.ExecuteTemplate(w, "edit/limite", templ); err != nil {
							fmt.Println(err)
						}
					})
					r.Get("/teste", func(w http.ResponseWriter, r *http.Request) {
						problem := common.ProblemFromContext(r)
						templ := rt.hydrateTemplate(r)
						templ.Title = fmt.Sprintf("TESTE - EDIT | #%d: %s", problem.ID, problem.Name)
						if err := templates.ExecuteTemplate(w, "edit/tests", templ); err != nil {
							fmt.Println(err)
						}
					})
					r.Get("/teste/{id}", func(w http.ResponseWriter, r *http.Request) {
					})
				})
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				tasks, err := rt.db.GetAllTasks()
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("/tasks/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Tasks"
				templ.Tasks = tasks
				check(templates.ExecuteTemplate(w, "tasks", templ))
			})
			r.With(rt.ValidateTaskID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("Task %d", templ.Task.ID)
				check(templates.ExecuteTemplate(w, "task", templ))
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Admin switches"
			check(templates.ExecuteTemplate(w, "admin", templ))
		})

		r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Log In"
			check(templates.ExecuteTemplate(w, "login", templ))
		})
		r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Sign Up"
			check(templates.ExecuteTemplate(w, "signup", templ))
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
func NewWeb(dm datamanager.Manager, db *kndb.DB) *Web {
	return &Web{dm: dm, db: db}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func init() {
	pkger.Include("/include")
	pkger.Include("/static")
}
