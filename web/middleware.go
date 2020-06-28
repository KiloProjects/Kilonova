package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
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
		resp, _ := http.Get(fmt.Sprintf("http://localhost:8080/api/problem/getByID?id=%d", problemID))
		var ret retData
		json.NewDecoder(resp.Body).Decode(&ret)
		if ret.Status != "success" {
			fmt.Println("ValidateProblemID:", err)
			fmt.Println(err, ret.Data)
			http.Error(w, ret.Data.(string), http.StatusBadRequest)
			return
		}
		var problem common.Problem
		rt.remarshal(ret.Data, &problem)
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
		resp, _ := http.Get(fmt.Sprintf("http://localhost:8080/api/tasks/getByID?id=%d", taskID))
		var ret retData
		json.NewDecoder(resp.Body).Decode(&ret)
		if ret.Status != "success" {
			fmt.Println(ret.Data)
			return
		}
		ctx := context.WithValue(r.Context(), common.TaskID, uint(taskID))
		var task common.Task
		rt.remarshal(ret.Data, &task)
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
		if !common.UserFromContext(r).IsAdmin {
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
		req, err := http.NewRequest("GET", "http://localhost:8080/api/user/getSelf", nil)
		cookie, err := r.Cookie("kn-sessionid")
		if err != nil {
			ctx := context.WithValue(r.Context(), common.UserKey, nil)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		req.Header.Add("Authorization", "Bearer "+cookie.Value)
		resp, err := http.DefaultClient.Do(req)
		var user common.User
		var ret struct {
			Status string      `json:"status"`
			Data   common.User `json:"data"`
		}
		err = json.NewDecoder(resp.Body).
			Decode(&ret)
		user = ret.Data
		ctx := context.WithValue(r.Context(), common.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
