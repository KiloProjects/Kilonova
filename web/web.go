// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
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
	Problems    []common.Problem
	Problem     common.Problem
	ContentUser common.User
	ProblemID   uint
}

// GetRouter returns a chi.Router
func GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(getUser)

	templates = template.Must(template.New("web").Funcs(template.FuncMap{
		"dumpStruct": spew.Sdump,
	}).ParseGlob("templ/*.templ"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		templ := hydrateTemplate(r)
		if err := templates.ExecuteTemplate(w, "index.templ", templ); err != nil {
			fmt.Println(err)
		}
	})

	r.Route("/probleme", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			resp, _ := http.Get("http://localhost:8080/api/problem/getAll")
			var retData struct {
				Status string           `json:"status"`
				Data   []common.Problem `json:"data"`
			}
			json.NewDecoder(resp.Body).Decode(&retData)

			templ := hydrateTemplate(r)
			templ.Title = "Probleme"
			templ.Problems = retData.Data
			if err := templates.ExecuteTemplate(w, "probleme.templ", templ); err != nil {
				fmt.Println(err)
			}
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(ValidateProblemID)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				problem := common.ProblemFromContext(r)

				templ := hydrateTemplate(r)
				templ.Title = fmt.Sprintf("#%d: %s", problem.ID, problem.Name)
				templ.Problem = problem
				if err := templates.ExecuteTemplate(w, "problema.templ", templ); err != nil {
					fmt.Println(err)
				}
			})
			r.Get("/edit", func(w http.ResponseWriter, r *http.Request) {
				problem := common.ProblemFromContext(r)
				templ := hydrateTemplate(r)
				templ.Title = fmt.Sprintf("EDIT | #%d: %s", problem.ID, problem.Name)
				templ.Problem = problem
				if err := templates.ExecuteTemplate(w, "editpb.templ", templ); err != nil {
					fmt.Println(err)
				}
			})
		})
	})

	r.With(mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
		templ := hydrateTemplate(r)
		templ.Title = "Log In"
		if err := templates.ExecuteTemplate(w, "login.templ", templ); err != nil {
			fmt.Println(err)
		}
	})
	r.With(mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
		templ := hydrateTemplate(r)
		templ.Title = "Sign Up"
		if err := templates.ExecuteTemplate(w, "signup.templ", templ); err != nil {
			fmt.Println(err)
		}
	})

	r.With(mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		common.RemoveSessionCookie(w)
		http.Redirect(w, r, "/", http.StatusFound)
	})

	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})
	return r
}
