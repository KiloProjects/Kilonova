package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
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
			fmt.Println("ValidateProblemID:", err)
			http.Error(w, "Invalid problem ID", http.StatusBadRequest)
			return
		}
		// this is practically equivalent to /api/problem/getByID?id=problemID, but let's keep it fast
		problem, err := rt.db.GetProblemByID(uint(problemID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Problema nu există", http.StatusBadRequest)
				return
			}
			fmt.Println("ValidateProblemID:", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		ctx := context.WithValue(r.Context(), common.ProblemKey, problem)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

// ValidateTaskID puts the ID and the Task in the router context
func (rt *Web) ValidateTaskID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		taskID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			fmt.Println("ValidateTaskID:", err)
			http.Error(w, "Invalid task ID", http.StatusBadRequest)
			return
		}
		// this is equivalent to /api/tasks/getByID but it's faster to directly access
		task, err := rt.db.GetTaskByID(uint(taskID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				http.Error(w, "Task-ul nu există", http.StatusBadRequest)
				return
			}
			fmt.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		ctx := context.WithValue(r.Context(), common.TaskID, uint(taskID))
		ctx = context.WithValue(ctx, common.TaskKey, task)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (rt *Web) mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var defUser common.User
		if common.UserFromContext(r) == defUser {
			http.Error(w, "You must be logged in", 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeAdmin(next http.Handler) http.Handler {
	return rt.mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.UserFromContext(r).IsAdmin && common.UserFromContext(r).ID != 1 {
			http.Error(w, "You must be an admin", 403)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (rt *Web) mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var defUser common.User
		if common.UserFromContext(r) != defUser {
			http.Error(w, "You must not be logged in", 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) getUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// this is analogous to doing a web request to /api/user/getSelf, but it's faster to directly interact with the DB
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
		ctx := context.WithValue(r.Context(), common.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
