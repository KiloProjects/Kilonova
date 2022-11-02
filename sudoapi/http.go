package sudoapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

var decoder *schema.Decoder

type WebHandler struct {
	base *BaseAPI
}

func (s *WebHandler) GetHandler() http.Handler {
	r := chi.NewRouter()

	// Commented routes represent unimplemented endpoints exposed by functions
	// in the BaseAPI, but with no use yet found. When (if) the time comes, they should be implemented.

	// Please note that login only verifies the credentials and returns the user id matching the creds.
	// It is the caller's responsibility to actually create and manage the session.
	// On a similar note, signup only validates and creates the user!
	r.Post("/login", webWrapper(func(ctx context.Context, auth struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}) (int, *StatusError) {
		return s.base.Login(ctx, auth.Username, auth.Password)
	}))
	r.Post("/signup", webWrapper(func(ctx context.Context, auth struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Language string `json:"language"`
	}) (int, *StatusError) {
		return s.base.Signup(ctx, auth.Email, auth.Username, auth.Password, auth.Language)
	}))

	r.Get("/user/brief", webWrapper(func(ctx context.Context, args struct {
		UserID int `json:"id"`
	}) (*UserBrief, *StatusError) {
		return s.base.UserBrief(ctx, args.UserID)
	}))
	r.Get("/user/full", webWrapper(func(ctx context.Context, args struct {
		UserID int `json:"id"`
	}) (*UserFull, *StatusError) {
		return s.base.UserFull(ctx, args.UserID)
	}))

	r.Get("/user/briefByName", webWrapper(func(ctx context.Context, args struct {
		Name string `json:"name"`
	}) (*UserBrief, *StatusError) {
		return s.base.UserBriefByName(ctx, args.Name)
	}))
	r.Get("/user/fullByName", webWrapper(func(ctx context.Context, args struct {
		Name string `json:"name"`
	}) (*UserFull, *StatusError) {
		return s.base.UserFullByName(ctx, args.Name)
	}))

	// TODO
	r.Group(func(r chi.Router) {
		r.Use(s.withUser("user_id", true))
		r.Post("/user/update", webWrapper(func(ctx context.Context, args kilonova.UserUpdate) (string, *StatusError) {
			err := s.base.UpdateUser(ctx, util.UserBriefContext(ctx).ID, args)
			if err != nil {
				return "", err
			}
			return "Updated user", nil
		}))
		// r.Post("/user/verifyEmail", s.VerifyUserEmail)
		// r.Post("/user/resendEmail", s.ResendUserEmail)
		// r.Post("/user/delete", s.DeleteUser)
	})

	r.Get("/problem", webWrapper(func(ctx context.Context, args struct {
		ID int `json:"problem_id"`
	}) (*kilonova.Problem, *StatusError) {
		return s.base.Problem(ctx, args.ID)
	}))

	r.With(s.withUser("user_id", true)).Post("/problem", webWrapper(func(ctx context.Context, args struct {
		Title        string `json:"title"`
		ConsoleInput bool   `json:"console_input"`
	}) (int, *StatusError) {
		return s.base.CreateProblem(ctx, args.Title, util.UserBriefContext(ctx), args.ConsoleInput)
	}))
	r.With(s.withUser("user_id", false)).With(s.withProblem("problem_id", true)).Post("/problem/update", webWrapper(func(ctx context.Context, upd kilonova.ProblemUpdate) (string, *StatusError) {
		if err := s.base.UpdateProblem(ctx, util.ProblemContext(ctx).ID, upd, util.UserBriefContext(ctx)); err != nil {
			return "", err
		}
		return "Updated problem", nil
	}))
	r.With(s.withUser("looking_user", false)).Get("/problems", webWrapper(func(ctx context.Context, args struct{}) ([]*kilonova.Problem, *StatusError) {
		return s.base.Problems(ctx, kilonova.ProblemFilter{LookingUser: util.UserBriefContext(ctx), Look: true})
	}))
	r.Get("/problem/score", webWrapper(func(ctx context.Context, args struct {
		UserID    int `json:"user_id"`
		ProblemID int `json:"problem_id"`
	}) (int, *StatusError) {
		return s.base.MaxScore(ctx, args.UserID, args.ProblemID), nil
	}))
	r.Post("/problem/delete", webWrapper(func(ctx context.Context, args struct {
		ProblemID int `json:"problem_id"`
	}) (string, *StatusError) {
		if err := s.base.DeleteProblem(ctx, args.ProblemID); err != nil {
			return "", err
		}
		return "Deleted problem", nil
	}))

	r.With(s.withUser("looking_user", false)).Get("/submissions", webWrapper(func(ctx context.Context, args kilonova.SubmissionFilter) (*Submissions, *StatusError) {
		return s.base.Submissions(ctx, args, util.UserBriefContext(ctx))
	}))
	r.With(s.withUser("looking_user", false)).Get("/submission", webWrapper(func(ctx context.Context, args struct {
		SubmissionID int `json:"submission_id"`
	}) (*FullSubmission, *StatusError) {
		return s.base.Submission(ctx, args.SubmissionID, util.UserBriefContext(ctx))
	}))
	r.With(s.withUser("user_id", true)).With(s.withProblem("problem_id", true)).Post("/submission/create", webWrapper(s.CreateSubmission))
	r.Post("/submission/delete", webWrapper(s.DeleteSubmission))

	// r.Post("/attachment", s.CreateAttachment)
	// r.Get("/attachments", s.ProblemAttachments)
	// r.Get("/attachment/byID", s.Attachment)
	// r.Get("/attachment/byName", s.ProblemAttachment)
	// r.Post("/attachment/update", s.UpdateAttachment)
	// r.Post("/attachment/delete", s.DeleteAttachment)

	// subtasks

	// contests

	// tests

	// problem lists (old)

	// lists (pblist v2)
	// TODO

	// session
	r.Post("/session/create", webWrapper(func(ctx context.Context, args struct {
		ID int `json:"uid"`
	}) (string, *StatusError) {
		return s.base.CreateSession(ctx, args.ID)
	}))
	r.Get("/session/get", webWrapper(func(ctx context.Context, args struct {
		SID string `json:"session_id"`
	}) (*kilonova.UserFull, *StatusError) {
		uid, err := s.base.GetSession(ctx, args.SID)
		if err != nil {
			return nil, err
		}
		return s.base.UserFull(ctx, uid)
	}))
	r.Post("/session/remove", webWrapper(func(ctx context.Context, args struct {
		SID string `json:"session_id"`
	}) (string, *StatusError) {
		return "Removed.", s.base.RemoveSession(ctx, args.SID)
	}))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		zap.S().Warn("Tried to call endpoint that was not found")
		errorData(w, "404 not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		zap.S().Warn("Tried to use unknown method")
		errorData(w, "Method not allowed", 405)
	})

	return r
}

func NewWebHandler(base *BaseAPI) *WebHandler {
	return &WebHandler{base}
}

// middleware and wrappers

func (s *WebHandler) withUser(fieldName string, required bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user_id, err := strconv.Atoi(r.FormValue(fieldName))
			if err != nil || user_id <= 0 {
				if required {
					errorData(w, "Invalid user ID", 400)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			user, err1 := s.base.UserFull(r.Context(), user_id)
			if err1 != nil {
				if required {
					err1.WriteError(w)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.UserKey, user)))
		})
	}
}

func (s *WebHandler) withProblem(fieldName string, required bool) func(next http.Handler) http.Handler {
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

func webWrapper[T1, T2 any](handler func(context.Context, T1) (T2, *StatusError)) http.HandlerFunc {
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

// from api/util.go

func returnData(w http.ResponseWriter, retData any) {
	kilonova.StatusData(w, "success", retData, 200)
}

func errorData(w http.ResponseWriter, retData any, errCode int) {
	kilonova.StatusData(w, "error", retData, errCode)
}

func init() {
	decoder = schema.NewDecoder()
	decoder.SetAliasTag("json")
}
