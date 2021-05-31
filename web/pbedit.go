package web

import (
	"html/template"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/go-chi/chi"
)

type ProblemEditPart struct {
	db kilonova.DB
}

func (p *ProblemEditPart) EditIndex() func(w http.ResponseWriter, r *http.Request) {
	// TODO
	tmpl := parse(template.FuncMap{}, "edit/index.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, nil)
	}
}

func (p *ProblemEditPart) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/", p.EditIndex())
	// TODO
	return r
}
