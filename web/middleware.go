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

// ValidateProblemID makes sure the problem ID is a valid uint
func ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			fmt.Println("ValidateProblemID:", err)
			http.Error(w, "Invalid problem ID", http.StatusBadRequest)
			return
		}
		resp, _ := http.Get(fmt.Sprintf("http://localhost:8080/api/problem/getByID?id=%d", problemID))
		var retData struct {
			Status string      `json:"status"`
			Data   interface{} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&retData)
		if retData.Status != "success" {
			fmt.Println("ValidateProblemID:", err)
			fmt.Println(err, retData.Data)
			http.Error(w, retData.Data.(string), http.StatusBadRequest)
			return
		}
		var problem common.Problem
		remarshal(retData.Data, &problem)
		ctx := context.WithValue(r.Context(), common.ProblemKey, problem)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var defUser common.User
		if common.UserFromContext(r) == defUser {
			http.Error(w, "You must be logged in", 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustBeAdmin(next http.Handler) http.Handler {
	return mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.UserFromContext(r).IsAdmin {
			http.Error(w, "You must be an admin", 403)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var defUser common.User
		if common.UserFromContext(r) != defUser {
			http.Error(w, "You must not be logged in", 403)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getUser(next http.Handler) http.Handler {
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
