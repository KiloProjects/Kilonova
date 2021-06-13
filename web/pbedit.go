package web

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi"
)

type ProblemEditParams struct {
	User    *kilonova.User
	Problem *kilonova.Problem
	Topbar  *EditTopbar

	Attachments []*kilonova.Attachment
}

type ProblemEditPart struct {
	db kilonova.DB
	dm kilonova.DataStore
}

func (p *ProblemEditPart) EditIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/index.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &ProblemEditParams{
			User:    util.User(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"general", -1},
		})
	}
}

func (p *ProblemEditPart) EditDesc() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/desc.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &ProblemEditParams{
			User:    util.User(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"desc", -1},
		})
	}
}

func (p *ProblemEditPart) EditChecker() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/checker.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &ProblemEditParams{
			User:    util.User(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"checker", -1},
		})
	}
}

func (p *ProblemEditPart) EditAttachments() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/attachments.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := p.db.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		tmpl.Execute(w, &ProblemEditParams{
			User:    util.User(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"attachments", -1},

			Attachments: atts,
		})
	}
}

func (p *ProblemEditPart) TestIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/testScores.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &TestEditParams{util.User(r), util.Problem(r), nil, &EditTopbar{"tests", -2}, p.db, p.dm})
	}
}

func (p *ProblemEditPart) TestAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/testAdd.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &TestEditParams{util.User(r), util.Problem(r), nil, &EditTopbar{"tests", -1}, p.db, p.dm})
	}
}

func (p *ProblemEditPart) TestEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/testEdit.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &TestEditParams{util.User(r), util.Problem(r), util.Test(r), &EditTopbar{"tests", util.Test(r).VisibleID}, p.db, p.dm})
	}
}

func (p *ProblemEditPart) SubtaskIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/subtaskIndex.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), nil, &EditTopbar{"subtasks", -2}, r.Context(), p.db})
	}
}

func (p *ProblemEditPart) SubtaskAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/subtaskAdd.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), nil, &EditTopbar{"subtasks", -1}, r.Context(), p.db})
	}
}

func (p *ProblemEditPart) SubtaskEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := parse(nil, "edit/subtaskEdit.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), util.SubTask(r), &EditTopbar{"subtasks", util.SubTask(r).VisibleID}, r.Context(), p.db})
	}
}

// Handler is the http handler to be attached
// The caller should ensure a User and a Problem are attached to the context
func (p *ProblemEditPart) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/", p.EditIndex())
	r.Get("/desc", p.EditDesc())
	r.Get("/checker", p.EditChecker())
	r.Get("/attachments", p.EditAttachments())

	r.Get("/test", p.TestIndex())
	r.Get("/test/add", p.TestAdd())
	r.With(TestIDValidator(p.db)).Get("/test/{tid}", p.TestEdit())

	r.Get("/subtasks", p.SubtaskIndex())
	r.Get("/subtasks/add", p.SubtaskAdd())
	r.With(SubTaskValidator(p.db)).Get("/subtasks/{stid}", p.SubtaskEdit())
	return r
}

func TestIDValidator(db kilonova.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			testID, err := strconv.Atoi(chi.URLParam(r, "tid"))
			if err != nil {
				statusPage(w, r, 400, "Test invalid", false)
				return
			}
			test, err := db.Test(r.Context(), util.Problem(r).ID, testID)
			if err != nil {
				log.Println(err)
				statusPage(w, r, 500, "", false)
				return
			}
			if test == nil {
				statusPage(w, r, 404, "Testul nu există", false)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
		})
	}
}

func SubTaskValidator(db kilonova.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subtaskID, err := strconv.Atoi(chi.URLParam(r, "stid"))
			if err != nil {
				statusPage(w, r, http.StatusBadRequest, "ID invalid", false)
				return
			}
			subtask, err := db.SubTask(r.Context(), util.Problem(r).ID, subtaskID)
			if err != nil {
				log.Println("ValidateSubTaskID:", err)
				statusPage(w, r, 500, "", false)
				return
			}
			if subtask == nil {
				statusPage(w, r, 404, "SubTask-ul nu există", false)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubTaskKey, subtask)))
		})
	}
}
