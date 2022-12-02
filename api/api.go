package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
)

var decoder *schema.Decoder

// API is the base struct for the project's API
type API struct {
	base *sudoapi.BaseAPI

	sudoHandlers *sudoapi.WebHandler

	testArchiveLock *sync.Mutex
}

// New declares a new API instance
func New(base *sudoapi.BaseAPI) *API {
	return &API{base, sudoapi.NewWebHandler(base), &sync.Mutex{}}
}

// Handler is the magic behind the API
func (s *API) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	r.With(s.MustBeAdmin).Route("/admin", func(r chi.Router) {

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)

		r.Post("/updateConfig", webMessageWrapper("Updated config. Some changes may only apply after a restart", s.base.UpdateConfig))

		r.Route("/maintenance", func(r chi.Router) {
			r.Post("/resetWaitingSubs", webMessageWrapper("Reset waiting subs", func(ctx context.Context, args struct{}) *kilonova.StatusError {
				return s.base.ResetWaitingSubmissions(ctx)
			}))
			r.Post("/reevaluateSubmission", webMessageWrapper("Reset submission", func(ctx context.Context, args struct {
				ID int `json:"id"`
			}) *kilonova.StatusError {
				return s.base.ResetSubmission(ctx, args.ID)
			}))
		})

		r.Get("/getAllUsers", s.getAllUsers)
	})

	r.Route("/auth", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/logout", s.logout)
		r.With(s.MustBeVisitor).Post("/signup", s.signup)
		r.With(s.MustBeVisitor).Post("/login", s.login)

		r.With(s.MustBeVisitor).Post("/forgotPassword", s.sendForgotPwdMail)
		r.Post("/resetPassword", s.resetPassword)
	})
	r.Route("/problem", func(r chi.Router) {
		r.Get("/get", s.getProblems)

		r.With(s.MustBeProposer).Post("/create", s.initProblem)
		r.Get("/maxScore", s.maxScore)

		r.Route("/{problemID}", func(r chi.Router) {
			r.Use(s.validateProblemID)
			r.Use(s.validateProblemEditor)

			r.Route("/update", func(r chi.Router) {
				r.Post("/", s.updateProblem)

				r.Post("/addTest", s.createTest)
				r.Route("/test/{tID}", func(r chi.Router) {
					r.Use(s.validateTestID)
					r.Post("/data", s.saveTestData)
					r.Post("/info", s.updateTestInfo)
					r.Post("/orphan", s.orphanTest)
				})

				r.Post("/addEditor", s.addProblemEditor)
				r.Post("/addViewer", s.addProblemViewer)
				r.Post("/stripAccess", s.stripProblemAccess)

				r.Post("/addAttachment", s.createAttachment)
				//r.With(s.validateAttachmentID).Post("/attachment/{aID}/", s.updateAttachmentMetadata)
				r.Post("/bulkDeleteAttachments", s.bulkDeleteAttachments)
				r.Post("/bulkUpdateAttachmentData", s.bulkUpdateAttachmentData)

				r.Post("/bulkDeleteTests", s.bulkDeleteTests)
				r.Post("/bulkUpdateTestScores", s.bulkUpdateTestScores)
				r.Post("/orphanTests", s.purgeTests)
				r.Post("/processTestArchive", s.processTestArchive)

				r.Post("/addSubTask", s.createSubTask)
				r.Post("/updateSubTask", s.updateSubTask)
				r.Post("/bulkUpdateSubTaskScores", s.bulkUpdateSubTaskScores)
				r.Post("/bulkDeleteSubTasks", s.bulkDeleteSubTasks)

			})

			r.Post("/reevaluateSubs", webMessageWrapper("Reevaluating submissions", func(ctx context.Context, args struct{}) *kilonova.StatusError {
				return s.base.ResetProblemSubmissions(ctx, util.ProblemContext(ctx).ID)
			}))

			r.Route("/get", func(r chi.Router) {
				r.Get("/attachments", webWrapper(func(ctx context.Context, args struct{}) ([]*kilonova.Attachment, *kilonova.StatusError) {
					return s.base.ProblemAttachments(ctx, util.ProblemContext(ctx).ID)
				}))
				// The one from /web/web.go is good enough
				// r.Get("/attachmentData", s.getAttachment)

				r.Get("/accessControl", s.getProblemAccessControl)

				r.Get("/tests", s.getTests)
				r.Get("/test", s.getTest)
			})
			r.Post("/delete", s.deleteProblem)
		})
	})
	r.Route("/submissions", func(r chi.Router) {
		r.Get("/get", s.filterSubs())
		r.Get("/getByID", s.getSubmissionByID())

		r.With(s.MustBeAuthed).Post("/createPaste", s.createPaste)

		r.With(s.MustBeAuthed).With(s.withProblem("problemID", true)).Post("/submit", webWrapper(func(ctx context.Context, args struct {
			Code      string `json:"code"`
			Lang      string `json:"language"`
			ProblemID int    `json:"problemID"`
		}) (int, *kilonova.StatusError) {
			lang, ok := eval.Langs[args.Lang]
			if !ok {
				return -1, kilonova.Statusf(400, "Invalid language")
			}
			return s.base.CreateSubmission(ctx, util.UserBriefContext(ctx), util.ProblemContext(ctx), args.Code, lang)
		}))
		r.With(s.MustBeAdmin).Post("/delete", webWrapper(s.sudoHandlers.DeleteSubmission))
	})
	r.Route("/paste/{pasteID}", func(r chi.Router) {
		r.Get("/", s.getPaste)
		r.With(s.MustBeAuthed).Post("/delete", s.deletePaste)
	})
	r.Route("/user", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/setBio", s.setBio())
		r.With(s.MustBeAuthed).Post("/setPreferredLanguage", s.setPreferredLanguage())

		r.With(s.MustBeAuthed).Post("/resendEmail", s.resendVerificationEmail)

		r.Get("/get", webWrapper(func(ctx context.Context, args struct {
			ID int `json:"id"`
		}) (*kilonova.UserBrief, *sudoapi.StatusError) {
			return s.base.UserBrief(ctx, args.ID)
		}))
		r.Get("/getByName", webWrapper(func(ctx context.Context, args struct {
			Name string `json:"name"`
		}) (*kilonova.UserBrief, *sudoapi.StatusError) {
			return s.base.UserBriefByName(ctx, args.Name)
		}))
		r.Get("/getSelf", func(w http.ResponseWriter, r *http.Request) { returnData(w, util.UserFull(r)) })
		r.With(s.MustBeAuthed).Get("/getSelfSolvedProblems", s.getSelfSolvedProblems)
		r.With(s.MustBeAuthed).Get("/getSolvedProblems", s.getSolvedProblems)

		r.With(s.MustBeAdmin).Route("/moderation", func(r chi.Router) {
			r.Post("/purgeBio", s.purgeBio)
			// TODO
			// r.Post("/nukeUser", s.nukeUser)
			// r.Post("/banUser", s.banUser)
			r.Post("/deleteUser", s.deleteUser)
		})

		r.Get("/getGravatar", s.getGravatar)
		r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.getSelfGravatar)

		// TODO: Make this secure and maybe with email stuff
		r.With(s.MustBeAuthed).Post("/changeEmail", s.changeEmail)
		r.With(s.MustBeAuthed).Post("/changePassword", s.changePassword)
	})
	r.Route("/problemList", func(r chi.Router) {
		r.Get("/get", s.getProblemList)
		r.Get("/getComplex", s.getComplexProblemList)
		r.Get("/filter", s.problemLists)
		r.With(s.MustBeProposer).Post("/create", s.initProblemList)
		r.With(s.MustBeAuthed).Post("/update", s.updateProblemList)
		r.With(s.MustBeAuthed).Post("/delete", s.deleteProblemList)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Method not allowed", 405)
	})

	return r
}

func (s *API) withProblem(fieldName string, required bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			problem_id, err := strconv.Atoi(r.FormValue(fieldName))
			if err != nil || problem_id <= 0 {
				if required {
					errorData(w, "Invalid problem ID", 400)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			problem, err1 := s.base.Problem(r.Context(), problem_id)
			if err1 != nil {
				if required {
					err1.WriteError(w)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemKey, problem)))
		})
	}
}

func webWrapper[T1, T2 any](handler func(context.Context, T1) (T2, *sudoapi.StatusError)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var query T1
		if err := decoder.Decode(&query, r.Form); err != nil {
			errorData(w, "Invalid request parameters", 400)
			return
		}
		rez, err := handler(r.Context(), query)
		if err != nil {
			err.WriteError(w)
			return
		}
		returnData(w, rez)
	}
}

func webMessageWrapper[T1 any](successString string, handler func(context.Context, T1) *sudoapi.StatusError) http.HandlerFunc {
	return webWrapper(func(ctx context.Context, args T1) (string, *kilonova.StatusError) {
		if err := handler(ctx, args); err != nil {
			return "", err
		}
		return successString, nil
	})
}

func init() {
	decoder = schema.NewDecoder()
	decoder.SetAliasTag("json")
}

func returnData(w http.ResponseWriter, retData interface{}) {
	kilonova.StatusData(w, "success", retData, 200)
}

func errorData(w http.ResponseWriter, retData interface{}, errCode int) {
	kilonova.StatusData(w, "error", retData, errCode)
}

func parseJsonBody[T any](r *http.Request, output *T) *kilonova.StatusError {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(output); err != nil {
		return kilonova.Statusf(400, "Invalid JSON input.")
	}
	return nil
}
