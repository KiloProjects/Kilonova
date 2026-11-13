package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui/editors"
	"github.com/KiloProjects/kilonova/web/ui/pbedit"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/go-chi/chi/v5"
	g "maragu.dev/gomponents"
)

func (rt *Web) editIndex() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		pb := util.Problem(r)
		chk, err := rt.base.ProblemChecklist(ctx, pb.ID)
		if err != nil || chk == nil {
			chk = &kilonova.ProblemChecklist{}
		}

		diagnostics, err := rt.base.ProblemDiagnostics(ctx, pb)
		if err != nil {
			slog.WarnContext(ctx, "Error getting diagnostics", slog.Any("err", err))
		}
		diagLines := make([]string, 0, len(diagnostics))
		for _, d := range diagnostics {
			diagLines = append(diagLines, d.Level.String()+": "+d.Message)
		}

		settings, err := rt.base.ProblemSettings(ctx, pb)
		if err != nil {
			slog.WarnContext(ctx, "Could not get problem settings", slog.Any("err", err))
			settings = nil
		}

		tags, err := rt.base.ProblemTags(ctx, pb.ID)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't get problem tags", slog.Any("err", err))
			tags = nil
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.edit.index", pb.ID, pb.Name),
			Content: pbedit.General(ctx, pbedit.GeneralParams{
				Problem:     pb,
				Topbar:      rt.problemTopbar(r, "general", -1),
				Checklist:   chk,
				Diagnostics: diagLines,
				Settings:    settings,
				Tags:        tags,
				MaxMemMB:    float64(flags.TestMaxMemKB.Get(r.Context())) / 1024.0,
				LLMEnabled:  rt.llmEnabled,
			}),
		})
	}
}

func (rt *Web) editDesc() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		variants, err := rt.base.ProblemDescVariants(r.Context(), util.Problem(r).ID, true)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get statement variants", slog.Any("err", err))
			http.Error(w, "Couldn't get statement variants", 500)
			return
		}

		finalVariant := rt.getFinalVariant(r.FormValue("pref_lang"), r.FormValue("pref_type"), variants)

		var statementData string
		var att *kilonova.Attachment
		att, err = rt.base.ProblemAttByName(r.Context(), util.Problem(r).ID, rt.base.FormatDescName(finalVariant))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Couldn't get problem statement attachment", slog.Any("err", err))
			http.Error(w, "Couldn't get problem statement attachment", 500)
			return
		}
		if att != nil {
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get problem statement", slog.Any("err", err))
				http.Error(w, "Couldn't get problem statement", 500)
				return
			}
			statementData = string(val)
		}

		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.edit.description", pb.ID, pb.Name),
			Content: g.Group{
				problems.Topbar(r.Context(), rt.problemTopbar(r, "desc", -1)),
				editors.StatementEditor(r.Context(), editors.StatementEditorParams{
					Variants: variants,
					Variant:  finalVariant,
					Data:     statementData,
					Att:      att,

					APIPrefix: fmt.Sprintf("/problem/%d", pb.ID),
				}),
			},
		})
	}
}

func (rt *Web) editAttachments() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.edit.attachments", pb.ID, pb.Name),
			Content: g.Group{
				problems.Topbar(r.Context(), rt.problemTopbar(r, "attachments", -1)),
				editors.AttachmentEditor(r.Context(), editors.AttachmentEditorParams{
					Attachments: atts,
					Problem:     pb,
					Updaters:    rt.attUpdaters(r.Context(), atts),
					APIPrefix:   fmt.Sprintf("/problem/%d", pb.ID),
				}),
			},
		})
	}
}

func (rt *Web) editAccessControl() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.access_control", pb.ID, pb.Name),
			Content: pbedit.Access(r.Context(), pb, rt.problemTopbar(r, "access", -1)),
		})
	}
}

func (rt *Web) testsParams(r *http.Request, pageID int) pbedit.TestsParams {
	pb := util.Problem(r)
	tests, err := rt.base.Tests(r.Context(), pb.ID)
	if err != nil {
		tests = nil
	}
	return pbedit.TestsParams{Problem: pb, Topbar: rt.problemTopbar(r, "tests", pageID), Tests: tests}
}

func (rt *Web) testIndex() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.tests", pb.ID, pb.Name),
			Content: pbedit.TestScores(r.Context(), rt.testsParams(r, -2)),
		})
	}
}

func (rt *Web) testAdd() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb := util.Problem(r)
		params := rt.testsParams(r, -1)
		params.NextVID = rt.base.NextVID(r.Context(), pb.ID)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.test_add", pb.ID, pb.Name),
			Content: pbedit.TestAdd(r.Context(), params),
		})
	}
}

func (rt *Web) testEdit() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb, test := util.Problem(r), util.Test(r)
		params := rt.testsParams(r, test.ID)
		params.Test = test
		params.Data = rt.testData(r.Context(), test.ID)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.test_edit", test.VisibleID, pb.ID, pb.Name),
			Content: pbedit.TestEdit(r.Context(), params),
		})
	}
}

// testData reads a test's input and output, replacing either with a message if it is too large or unreadable.
func (rt *Web) testData(ctx context.Context, testID int) pbedit.TestData {
	in, err := rt.base.TestInput(testID)
	if err != nil {
		return pbedit.TestData{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := rt.base.TestOutput(testID)
	if err != nil {
		return pbedit.TestData{In: "err", Out: "err"}
	}
	defer out.Close()

	inData, okIn := readOrTruncate(ctx, in)
	outData, okOut := readOrTruncate(ctx, out)
	return pbedit.TestData{In: string(inData), OkIn: okIn, Out: string(outData), OkOut: okOut}
}

const readLimit = 1024 * 1024 // 1MB

func readOrTruncate(ctx context.Context, r io.Reader) ([]byte, bool) {
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, readLimit); err != nil {
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), true
		}
		slog.WarnContext(ctx, "Could not read until limit", slog.Any("err", err))
		return []byte("err"), false
	}

	return []byte("Files larger than 1MB cannot be displayed"), false
}

func (rt *Web) subtasksParams(r *http.Request, pageID int) pbedit.SubtasksParams {
	pb := util.Problem(r)
	sts, err := rt.base.SubTasks(r.Context(), pb.ID)
	if err != nil {
		sts = nil
	}
	tests, err := rt.base.Tests(r.Context(), pb.ID)
	if err != nil {
		tests = nil
	}
	return pbedit.SubtasksParams{Problem: pb, Topbar: rt.problemTopbar(r, "subtasks", pageID), Subtasks: sts, Tests: tests}
}

func (rt *Web) subtaskIndex() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.subtask", pb.ID, pb.Name),
			Content: pbedit.SubtaskIndex(r.Context(), rt.subtasksParams(r, -2)),
		})
	}
}

func (rt *Web) subtaskAdd() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb := util.Problem(r)
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.subtask_add", pb.ID, pb.Name),
			Content: pbedit.SubtaskAdd(r.Context(), rt.subtasksParams(r, -1)),
		})
	}
}

func (rt *Web) subtaskEdit() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pb, st := util.Problem(r), util.SubTask(r)
		params := rt.subtasksParams(r, st.ID)
		params.SubTask = st
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.edit.subtask_edit", st.VisibleID, pb.ID, pb.Name),
			Content: pbedit.SubtaskEdit(r.Context(), params),
		})
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
			testID, err := strconv.Atoi(r.PathValue("tid"))
			if err != nil {
				rt.statusPage(w, r, 400, "Test invalid")
				return
			}
			test, err := rt.base.Test(r.Context(), testID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get test", slog.Any("err", err))
				rt.statusPage(w, r, 500, "")
				return
			}
			if test == nil {
				rt.statusPage(w, r, 404, "Testul nu există")
				return
			}
			if test.ProblemID != util.Problem(r).ID {
				rt.statusPage(w, r, 404, "Testul nu aparține problemei")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
		})
	}
}

func (rt *Web) SubTaskValidator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subtaskID, err := strconv.Atoi(r.PathValue("stid"))
			if err != nil {
				rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
				return
			}
			subtask, err := rt.base.SubTask(r.Context(), subtaskID)
			if err != nil {
				rt.statusPage(w, r, 500, "")
				return
			}
			if subtask == nil {
				rt.statusPage(w, r, 404, "SubTask-ul nu există")
				return
			}
			if subtask.ProblemID != util.Problem(r).ID {
				rt.statusPage(w, r, 404, "SubTask-ul nu aparține problemei")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubTaskKey, subtask)))
		})
	}
}
