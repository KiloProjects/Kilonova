package web

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/internal/cookie"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/go-chi/chi"
)

type retData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ValidateProblemID makes sure the problem ID is a valid uint
func (rt *Web) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			rt.status(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		// this is practically equivalent to /api/problem/getByID?id=problemID, but let's keep it fast
		problem, err := rt.db.Problem(r.Context(), problemID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 404, "Problema nu a fost găsită")
				return
			}
			log.Println("ValidateProblemID:", err)
			rt.status(w, r, 500, "")
			return
		}
		ctx := context.WithValue(r.Context(), util.ProblemKey, problem)
		ctx = context.WithValue(ctx, util.PbID, problem.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

// ValidateVisible checks if the problem from context is visible from the logged in user
func (rt *Web) ValidateVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemVisible(r) {
			rt.status(w, r, 404, "Problema nu a fost găsită")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateSubmissionID puts the ID and the Submission in the router context
func (rt *Web) ValidateSubmissionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			rt.status(w, r, 400, "ID submisie invalid")
			return
		}
		// this is equivalent to /api/submissions/getByID but it's faster to directly access
		sub, err := rt.db.Submission(r.Context(), subID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 400, "Submisia nu există")
				return
			}
			log.Println(err)
			rt.status(w, r, 500, "")
			return
		}

		if !util.IsSubmissionVisible(sub, util.User(r)) {
			sub.Code = ""
		}

		ctx := context.WithValue(r.Context(), util.SubID, uint(subID))
		ctx = context.WithValue(ctx, util.SubKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateTestID checks for the correctness of the test ID and adds it to context if ok
func (rt *Web) ValidateTestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testID, err := strconv.ParseInt(chi.URLParam(r, "tid"), 10, 32)
		if err != nil {
			rt.status(w, r, 400, "Test invalid")
			return
		}
		test, err := rt.db.TestVisibleID(r.Context(), db.TestVisibleIDParams{ProblemID: util.ID(r, util.PbID), VisibleID: testID})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 404, "Testul nu există")
				return
			}
			log.Println(err)
			rt.status(w, r, 500, "")
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
			rt.status(w, r, 401, "Trebuie să fii logat")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProposer(r) {
			rt.status(w, r, 401, "Trebuie să fii propunător")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeAdmin(next http.Handler) http.Handler {
	return rt.mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAdmin(r) {
			rt.status(w, r, 401, "Trebuie să fii admin")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (rt *Web) mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.IsRAuthed(r) {
			rt.status(w, r, 401, "Trebuie să fii delogat")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemEditor(r) {
			rt.status(w, r, 401, "Trebuie să fii autorul problemei")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) getUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this is analogous to doing a web request to /api/user/getSelf, but it's faster (and easier) to directly interact with the DB
		sess := cookie.GetSession(r)
		if sess == nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := rt.db.User(r.Context(), sess.UserID)
		user.Password = ""
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), util.UserID, user.ID)
		ctx = context.WithValue(ctx, util.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
