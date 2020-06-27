package server

import (
	"context"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
)

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if common.IsAuthed(r) {
			s.ErrorData(w, "You must not be logged in to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return s.MustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsAdmin(r) {
			s.ErrorData(w, "You must be an admin to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsAuthed(r) {
			s.ErrorData(w, "You must be authenticated to view this", http.StatusUnauthorized)
			return
		}
		var user common.User
		session := common.GetSession(r)
		s.db.First(&user, session.UserID)
		ctx := context.WithValue(r.Context(), common.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
