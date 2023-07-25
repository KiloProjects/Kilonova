package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/archive/test"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
	"github.com/gosimple/slug"
	"go.uber.org/zap"
)

type StatementEditorParams struct {
	Lang string
	Data string
	Att  *kilonova.Attachment

	APIPrefix string
}

type AttachmentEditorParams struct {
	Attachments []*kilonova.Attachment
	Problem     *kilonova.Problem
	BlogPost    *kilonova.BlogPost

	APIPrefix string
}

type ProblemEditParams struct {
	Ctx     *ReqContext
	Problem *kilonova.Problem
	Topbar  *ProblemTopbar

	Checklist *kilonova.ProblemChecklist

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
		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "general", -1),

			Checklist: chk,
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

		finalLang := rt.getFinalLang(r.FormValue("pref_lang"), variants)

		var statementData string
		var att *kilonova.Attachment
		if finalLang == "" {
			finalLang = config.Common.DefaultLang
		} else {
			att, err = rt.base.ProblemAttByName(r.Context(), util.Problem(r).ID, fmt.Sprintf("statement-%s.md", finalLang))
			if err != nil {
				zap.S().Warn(err)
				http.Error(w, "Couldn't get problem statement attachment", 500)
				return
			}
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				zap.S().Warn(err)
				http.Error(w, "Couldn't get problem statement", 500)
				return
			}
			statementData = string(val)
		}

		rt.runTempl(w, r, tmpl, &ProblemEditParams{
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "desc", -1),

			StatementEditor: &StatementEditorParams{
				Lang: finalLang,
				Data: statementData,
				Att:  att,

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
			Ctx:     GenContext(r),
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
			Ctx:     GenContext(r),
			Problem: util.Problem(r),
			Topbar:  rt.problemTopbar(r, "access", -1),
		})
	}
}

func (rt *Web) testIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testScores.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{
			GenContext(r), util.Problem(r), nil, rt.problemTopbar(r, "tests", -2), rt.base,
		})
	}
}

func (rt *Web) testArchive() func(w http.ResponseWriter, r *http.Request) {
	decoder := schema.NewDecoder()
	decoder.SetAliasTag("Json")
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct {
			Brief       bool `json:"brief"`
			Submissions bool `json:"submissions"`
			Editors     bool `json:"editors"`
			Name        bool `json:"name"`
		}
		if err := decoder.Decode(&args, r.Form); err != nil {
			http.Error(w, "Can't decode parameters", 400)
			return
		}

		args.Submissions = args.Submissions && rt.base.IsAdmin(util.UserBrief(r))
		args.Editors = args.Editors && rt.base.IsAdmin(util.UserBrief(r))

		w.Header().Add("Content-Type", "application/zip")
		w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.zip"`, slug.Make(util.Problem(r).Name)))
		w.WriteHeader(200)
		if err := test.GenerateArchive(r.Context(), util.Problem(r), w, rt.base, &test.ArchiveGenOptions{
			Brief:       args.Brief,
			Submissions: args.Submissions,
			Editors:     args.Editors,
			Name:        args.Name,
		}); err != nil {
			if !errors.Is(err, context.Canceled) {
				zap.S().Warn(err)
			}
			fmt.Fprint(w, err)
		}
	}
}

func (rt *Web) testAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testAdd.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{
			GenContext(r), util.Problem(r), nil, rt.problemTopbar(r, "tests", -1), rt.base,
		})
	}
}

func (rt *Web) testEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/testEdit.html", "problem/topbar.html", "problem/edit/testSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &TestEditParams{
			GenContext(r), util.Problem(r), util.Test(r), rt.problemTopbar(r, "tests", util.Test(r).VisibleID), rt.base,
		})
	}
}

func (rt *Web) subtaskIndex() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskIndex.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{
			GenContext(r), util.Problem(r), nil, rt.problemTopbar(r, "subtasks", -2), r.Context(), rt.base},
		)
	}
}

func (rt *Web) subtaskAdd() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskAdd.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{
			GenContext(r), util.Problem(r), nil, rt.problemTopbar(r, "subtasks", -1), r.Context(), rt.base},
		)
	}
}

func (rt *Web) subtaskEdit() func(w http.ResponseWriter, r *http.Request) {
	tmpl := rt.parse(nil, "problem/edit/subtaskEdit.html", "problem/topbar.html", "problem/edit/subtaskSidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, tmpl, &SubTaskEditParams{
			GenContext(r), util.Problem(r), util.SubTask(r), rt.problemTopbar(r, "subtasks", util.SubTask(r).VisibleID), r.Context(), rt.base},
		)
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
	r.Get("/test/archive", rt.testArchive())
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
			test, err1 := rt.base.Test(r.Context(), util.Problem(r).ID, testID)
			if err1 != nil {
				zap.S().Warn(err)
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
			subtask, err1 := rt.base.SubTask(r.Context(), util.Problem(r).ID, subtaskID)
			if err1 != nil {
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
