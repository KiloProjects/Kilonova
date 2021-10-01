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
	Ctx     *ReqContext
	Problem *kilonova.Problem
	Topbar  *EditTopbar

	Attachments []*kilonova.Attachment
}

func (rt *Web) editIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/index.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &ProblemEditParams{
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"general", -1},
		})
	}
}

func (rt *Web) editDesc() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/desc.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &ProblemEditParams{
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"desc", -1},
		})
	}
}

func (rt *Web) editAttachments() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/attachments.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.db.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		runTempl(w, r, tmpl, &ProblemEditParams{
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  &EditTopbar{"attachments", -1},

			Attachments: atts,
		})
	}
}

func (rt *Web) testIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/testScores.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &TestEditParams{GenContext(r), util.Problem(r), nil, &EditTopbar{"tests", -2}, rt.db, rt.dm})
	}
}

func (rt *Web) testAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/testAdd.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &TestEditParams{GenContext(r), util.Problem(r), nil, &EditTopbar{"tests", -1}, rt.db, rt.dm})
	}
}

func (rt *Web) testEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/testEdit.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &TestEditParams{GenContext(r), util.Problem(r), util.Test(r), &EditTopbar{"tests", util.Test(r).VisibleID}, rt.db, rt.dm})
	}
}

func (rt *Web) subtaskIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/subtaskIndex.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &SubTaskEditParams{GenContext(r), util.Problem(r), nil, &EditTopbar{"subtasks", -2}, r.Context(), rt.db})
	}
}

func (rt *Web) subtaskAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/subtaskAdd.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, tmpl, &SubTaskEditParams{GenContext(r), util.Problem(r), nil, &EditTopbar{"subtasks", -1}, r.Context(), rt.db})
	}
}

func (rt *Web) subtaskEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "edit/subtaskEdit.html", "edit/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Execute(w, &SubTaskEditParams{GenContext(r), util.Problem(r), util.SubTask(r), &EditTopbar{"subtasks", util.SubTask(r).VisibleID}, r.Context(), rt.db})
	}
}

// Handler is the http handler to be attached
// The caller should ensure a User and a Problem are attached to the context
func (rt *Web) ProblemEditRouter(r chi.Router) {
	r.Get("/", rt.editIndex())
	r.Get("/desc", rt.editDesc())
	r.Get("/attachments", rt.editAttachments())

	r.Get("/test", rt.testIndex())
	r.Get("/test/add", rt.testAdd())
	r.With(TestIDValidator(rt.db)).Get("/test/{tid}", rt.testEdit())

	r.Get("/subtasks", rt.subtaskIndex())
	r.Get("/subtasks/add", rt.subtaskAdd())
	r.With(SubTaskValidator(rt.db)).Get("/subtasks/{stid}", rt.subtaskEdit())
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
