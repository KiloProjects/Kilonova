// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
}

// Web is the struct representing this whole package
type Web struct {
	dm *datamanager.Manager
}

// GetRouter returns a chi.Router
func (rt *Web) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(rt.getUser)

	templates = template.Must(template.New("web").Funcs(template.FuncMap{
		"dumpStruct":  spew.Sdump,
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
	}).ParseGlob("templ/*.templ"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		templ := rt.hydrateTemplate(r)
		if err := templates.ExecuteTemplate(w, "index.templ", templ); err != nil {
			fmt.Println(err)
		}
	})

	r.Route("/probleme", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			resp, err := http.Get("http://localhost:8080/api/problem/getAll")
			if err != nil {
				fmt.Println("/probleme/", err)
				return
			}
			var ret retData
			json.NewDecoder(resp.Body).Decode(&ret)
			if ret.Status != "success" {
				fmt.Println(ret.Data)
				return
			}
			var problems []common.Problem
			rt.remarshal(ret.Data, &problems)
			templ := rt.hydrateTemplate(r)
			templ.Title = "Probleme"
			templ.Problems = problems
			if err := templates.ExecuteTemplate(w, "probleme.templ", templ); err != nil {
				fmt.Println(err)
			}
		})
		r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
			templ := rt.hydrateTemplate(r)
			templ.Title = "Creare problemă"
			if err := templates.ExecuteTemplate(w, "createpb.templ", templ); err != nil {
				fmt.Println(err)
			}
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(rt.ValidateProblemID)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problem := common.ProblemFromContext(r)

				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)
				templ.Problem = problem
				if err := templates.ExecuteTemplate(w, "problema.templ", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.Get("/edit", func(w http.ResponseWriter, r *http.Request) {
				problem := common.ProblemFromContext(r)
				templ := rt.hydrateTemplate(r)
				templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
				templ.Problem = problem
				if err := templates.ExecuteTemplate(w, "editpb.templ", templ); err != nil {
					fmt.Println(err)
				}
			})
		})
	})

	r.Route("/tasks", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			resp, err := http.Get("http://localhost:8080/api/tasks/get")
			if err != nil {
				fmt.Println("/tasks/:", err)
				http.Error(w, http.StatusText(500), 500)
				return
			}
			var ret struct {
				Status string      `json:"status"`
				Data   interface{} `json:"data"`
			}
			json.NewDecoder(resp.Body).Decode(&ret)
			if ret.Status != "success" {
				http.Error(w, fmt.Sprintf("%#v", ret.Data), http.StatusInternalServerError)
				return
			}
			templ := rt.hydrateTemplate(r)
			var tasks []common.Task
			rt.remarshal(ret.Data, &tasks)
			templ.Title = "Tasks"
			templ.Tasks = tasks
			if err := templates.ExecuteTemplate(w, "tasks.templ", templ); err != nil {
				fmt.Println(err)
			}
		})
		r.With(rt.ValidateTaskID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			task := common.TaskFromContext(r)
			templ := rt.hydrateTemplate(r)
			templ.Title = fmt.Sprintf("Task %d", task.ID)
			templ.Task = task
			if err := templates.ExecuteTemplate(w, "task.templ", templ); err != nil {
				fmt.Println(err)
			}
		})
	})

	r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
		templ := rt.hydrateTemplate(r)
		templ.Title = "Log In"
		if err := templates.ExecuteTemplate(w, "login.templ", templ); err != nil {
			fmt.Println(err)
		}
	})
	r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
		templ := rt.hydrateTemplate(r)
		templ.Title = "Sign Up"
		if err := templates.ExecuteTemplate(w, "signup.templ", templ); err != nil {
			fmt.Println(err)
		}
	})

	r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		common.RemoveSessionCookie(w)
		http.Redirect(w, r, "/", http.StatusFound)
	})

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})
	return r
}

// NewWeb returns a new web instance
func NewWeb(dm *datamanager.Manager) *Web {
	return &Web{dm: dm}
}
