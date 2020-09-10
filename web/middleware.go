package web

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
)

type retData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ValidateProblemID makes sure the problem ID is a valid uint
func (rt *Web) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			http.Error(w, "ID invalid", http.StatusBadRequest)
			return
		}
		// this is practically equivalent to /api/problem/getByID?id=problemID, but let's keep it fast
		problem, err := rt.db.GetProblemByID(uint(problemID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Problema nu a fost găsită", 404)
				return
			}
			rt.logger.Println("ValidateProblemID:", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		ctx := context.WithValue(r.Context(), util.ProblemKey, problem)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func (rt *Web) ValidateVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemVisible(r) {
			http.Error(w, "Problema nu a fost găsită", 404)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateSubmissionID puts the ID and the Submission in the router context
func (rt *Web) ValidateSubmissionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			http.Error(w, "Invalid submission ID", http.StatusBadRequest)
			return
		}
		// this is equivalent to /api/submissions/getByID but it's faster to directly access
		sub, err := rt.db.GetSubmissionByID(uint(subID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Submisia nu există", http.StatusBadRequest)
				return
			}
			rt.logger.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		if !util.IsSubmissionVisible(*sub, util.UserFromContext(r)) {
			sub.SourceCode = ""
		}

		ctx := context.WithValue(r.Context(), util.SubID, uint(subID))
		ctx = context.WithValue(ctx, util.SubKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (rt *Web) ValidateTestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testID, err := strconv.ParseUint(chi.URLParam(r, "tid"), 10, 32)
		if err != nil {
			http.Error(w, "Invalid test ID", http.StatusBadRequest)
		}
		test, err := rt.db.GetTestByVisibleID(util.ProblemFromContext(r).ID, uint(testID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Testul nu există", http.StatusBadRequest)
				return
			}
			rt.logger.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		ctx := context.WithValue(r.Context(), util.TestID, uint(testID))
		ctx = context.WithValue(ctx, util.TestKey, test)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (rt *Web) mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAuthed(r) {
			http.Error(w, "You must be logged in", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProposer(r) {
			http.Error(w, "You must be a proposer", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeAdmin(next http.Handler) http.Handler {
	return rt.mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAdmin(r) {
			http.Error(w, "You must be an admin", 401)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (rt *Web) mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.IsRAuthed(r) {
			http.Error(w, "You must not be logged in", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemEditor(r) {
			http.Error(w, "You must be the problem author", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) getUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this is analogous to doing a web request to /api/user/getSelf, but it's faster (and easier) to directly interact with the DB
		sess := common.GetSession(r)
		if sess == nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := rt.db.GetUserByID(sess.UserID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), util.UserID, user.ID)
		ctx = context.WithValue(ctx, util.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
