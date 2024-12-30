package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type StatementEditorParams struct {
	Variants []*kilonova.StatementVariant
	Variant  *kilonova.StatementVariant
	Data     string
	Att      *kilonova.Attachment

	APIPrefix string
}

type AttachmentEditorParams struct {
	Attachments []*kilonova.Attachment
	Problem     *kilonova.Problem
	BlogPost    *kilonova.BlogPost

	APIPrefix string
}

type ProblemEditParams struct {
	Problem *kilonova.Problem
	Topbar  *ProblemTopbar

	Diagnostics []*sudoapi.ProblemDiagnostic
	Checklist   *kilonova.ProblemChecklist

	AttachmentEditor *AttachmentEditorParams
	StatementEditor  *StatementEditorParams
}

func (rt *Web) editIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/index.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		chk, err := rt.base.ProblemChecklist(r.Context(), util.Problem(r).ID)
		if err != nil {
			chk = nil
		}

		diagnostics, err := rt.base.ProblemDiagnostics(r.Context(), util.Problem(r))
		if err != nil {
			slog.WarnContext(r.Context(), "Error getting diagnostics", slog.Any("err", err))
			diagnostics = nil
		}

		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "general", -1),

			Checklist:   chk,
			Diagnostics: diagnostics,
		})
	}
}

func (rt *Web) editDesc() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/desc.html", "modals/md_att_editor.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		variants, err := rt.base.ProblemDescVariants(r.Context(), util.Problem(r).ID, true)
		if err != nil {
			zap.S().Warn(err)
			http.Error(w, "Couldn't get statement variants", 500)
			return
		}

		finalVariant := rt.getFinalVariant(r.FormValue("pref_lang"), r.FormValue("pref_type"), variants)

		var statementData string
		var att *kilonova.Attachment
		att, err = rt.base.ProblemAttByName(r.Context(), util.Problem(r).ID, rt.base.FormatDescName(finalVariant))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			zap.S().Warn(err)
			http.Error(w, "Couldn't get problem statement attachment", 500)
			return
		}
		if att != nil {
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				zap.S().Warn(err)
				http.Error(w, "Couldn't get problem statement", 500)
				return
			}
			statementData = string(val)
		}

		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "desc", -1),

			StatementEditor: &StatementEditorParams{
				Variants: variants,
				Variant:  finalVariant,
				Data:     statementData,
				Att:      att,

				APIPrefix: fmt.Sprintf("/problem/%d", util.Problem(r).ID),
			},
		})
	}
}

func (rt *Web) editAttachments() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/attachments.html", "modals/att_manager.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "attachments", -1),

			AttachmentEditor: &AttachmentEditorParams{
				Attachments: atts,
				Problem:     util.Problem(r),
				APIPrefix:   fmt.Sprintf("/problem/%d", util.Problem(r).ID),
			},
		})
	}
}

func (rt *Web) editAccessControl() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/access.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "access", -1),
		})
	}
}

func (rt *Web) testIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testScores.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{util.Problem(r), nil, rt.problemTopbar(r, "tests", -2), rt.base, r.Context()})
	}
}

func (rt *Web) testAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testAdd.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{util.Problem(r), nil, rt.problemTopbar(r, "tests", -1), rt.base, r.Context()})
	}
}

func (rt *Web) testEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testEdit.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{util.Problem(r), util.Test(r), rt.problemTopbar(r, "tests", util.Test(r).VisibleID), rt.base, r.Context()})
	}
}

func (rt *Web) subtaskIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskIndex.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{util.Problem(r), nil, rt.problemTopbar(r, "subtasks", -2), r.Context(), rt.base})
	}
}

func (rt *Web) subtaskAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskAdd.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{util.Problem(r), nil, rt.problemTopbar(r, "subtasks", -1), r.Context(), rt.base})
	}
}

func (rt *Web) subtaskEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskEdit.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{util.Problem(r), util.SubTask(r), rt.problemTopbar(r, "subtasks", util.SubTask(r).VisibleID), r.Context(), rt.base})
	}
}

// Handler is the http handler to be attached
// The caller should ensure a User and a Problem are attached to the context
func (rt *Web) ProblemEditRouter(r chi.Router) {
	r.Get("/", rt.editIndex())
	r.Get("/desc", rt.editDesc())
	r.Get("/attachments", rt.editAttachments())
	r.Get("/access", rt.editAccessControl())

	r.Get("/test", rt.testIndex())
	r.Get("/test/add", rt.testAdd())
	r.With(rt.TestIDValidator()).Get("/test/{tid}", rt.testEdit())

	r.Get("/subtasks", rt.subtaskIndex())
	r.Get("/subtasks/add", rt.subtaskAdd())
	r.With(rt.SubTaskValidator()).Get("/subtasks/{stid}", rt.subtaskEdit())
}

func (rt *Web) TestIDValidator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			testID, err := strconv.Atoi(chi.URLParam(r, "tid"))
			if err != nil {
				rt.statusPage(w, r, 400, "Test invalid")
				return
			}
			test, err := rt.base.Test(r.Context(), util.Problem(r).ID, testID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get test", slog.Any("err", err))
				rt.statusPage(w, r, 500, "")
				return
			}
			if test == nil {
				rt.statusPage(w, r, 404, "Testul nu există")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
		})
	}
}

func (rt *Web) SubTaskValidator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subtaskID, err := strconv.Atoi(chi.URLParam(r, "stid"))
			if err != nil {
				rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
				return
			}
			subtask, err := rt.base.SubTask(r.Context(), util.Problem(r).ID, subtaskID)
			if err != nil {
				rt.statusPage(w, r, 500, "")
				return
			}
			if subtask == nil {
				rt.statusPage(w, r, 404, "SubTask-ul nu există")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubTaskKey, subtask)))
		})
	}
}
