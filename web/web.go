// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"

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
	User     *common.User
	LoggedIn bool

	// Page-specific data
	// it is easier to just put this stuff here instead of in a `Data` interface
	Problems *[]common.Problem
	Problem  *common.Problem

	ContentUser *common.User

	Tasks *[]common.Task
	Task  *common.Task

	ProblemID uint
}

// Web is the struct representing this whole package
type Web struct {
	dm datamanager.Manager
	db *kndb.DB
}

// GetRouter returns a chi.Router
func (rt *Web) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	templates = template.New("web").Funcs(template.FuncMap{
		"dumpStruct": spew.Sdump,
		"dumpAsJson": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "    ")
			return string(b)
		},
		"getTestData": rt.getTestData,
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
		"KBtoMB": func(kb int) float64 {
			return float64(kb) / 1024.0
		},
		"gradient": gradient,
		"zeroto100": func() []int {
			var v []int = make([]int, 0)
			for i := 0; i <= 20; i++ {
				v = append(v, i)
			}
			return v
		},
	})
	templates = template.Must(parseAllTemplates(templates, root))

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
		file.Stat()
		http.ServeContent(w, r, fstat.Name(), fstat.ModTime(), file)
	})

	// Optimization: get the user only on routes that need to
	r.With(rt.getUser).Route("/", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			if err := templates.ExecuteTemplate(w, "index", templ); err != nil {
				fmt.Println(err)
			}
		})

		r.Route("/probleme", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problems, err := rt.db.GetAllProblems()
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					fmt.Println("/probleme/", err)
					http.Error(w, http.StatusText(500), 500)
					return
				}
				templ := rt.hydrateTemplate(r)
				templ.Title = "Probleme"
				templ.Problems = &problems
				if err := templates.ExecuteTemplate(w, "probleme", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
				templ := rt.hydrateTemplate(r)
				templ.Title = "Creare problemă"
				if err := templates.ExecuteTemplate(w, "createpb", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.Route("/{id}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)

					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)
					templ.Problem = &problem
					if err := templates.ExecuteTemplate(w, "problema", templ); err != nil {
						fmt.Println(err)
					}
				})
				r.Get("/edit", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)
					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
					templ.Problem = &problem
					if err := templates.ExecuteTemplate(w, "edit/index", templ); err != nil {
						fmt.Println(err)
					}
				})
				r.Get("/edit/enunt", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)
					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("ENUNT - EDIT | #%d: %s", problem.ID, problem.Name)
					templ.Problem = &problem
					if err := templates.ExecuteTemplate(w, "edit/enunt", templ); err != nil {
						fmt.Println(err)
					}
				})
				r.Get("/edit/limite", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)
					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("LIMITE - EDIT | #%d: %s", problem.ID, problem.Name)
					templ.Problem = &problem
					if err := templates.ExecuteTemplate(w, "edit/limite", templ); err != nil {
						fmt.Println(err)
					}
				})
				r.Get("/edit/teste", func(w http.ResponseWriter, r *http.Request) {
					problem := common.ProblemFromContext(r)
					templ := rt.hydrateTemplate(r)
					templ.Title = fmt.Sprintf("TESTE - EDIT | #%d: %s", problem.ID, problem.Name)
					templ.Problem = &problem
					if err := templates.ExecuteTemplate(w, "edit/tests", templ); err != nil {
						fmt.Println(err)
					}
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
				templ.Tasks = &tasks
				if err := templates.ExecuteTemplate(w, "tasks", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.With(rt.ValidateTaskID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				task := common.TaskFromContext(r)
				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("Task %d", task.ID)
				templ.Task = &task
				if err := templates.ExecuteTemplate(w, "task", templ); err != nil {
					fmt.Println(err)
				}
			})
		})

		r.With(rt.mustBeAdmin).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Admin switches"
			if err := templates.ExecuteTemplate(w, "admin", templ); err != nil {
				fmt.Println(err)
			}
		})

		r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Log In"
			if err := templates.ExecuteTemplate(w, "login", templ); err != nil {
				fmt.Println(err)
			}
		})
		r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Sign Up"
			if err := templates.ExecuteTemplate(w, "signup", templ); err != nil {
				fmt.Println(err)
			}
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

func init() {
	pkger.Include("/include")
	pkger.Include("/static")
}
