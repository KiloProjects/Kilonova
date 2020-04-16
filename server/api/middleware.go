package api

import (
	"context"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
)

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.IsAuthed(r) {
			s.ErrorData(w, "You must not be logged in to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.IsAdmin(r) {
			s.ErrorData(w, "You must be an admin to view this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if !s.IsAuthed(r) {
			s.ErrorData(w, "You must be authenticated to view this", http.StatusUnauthorized)
			return
		}
		var user models.User
		session := s.GetSession(r)
		s.db.First(&user, "id = ?", session.UserID)
		ctx = context.WithValue(ctx, models.KNContextType("user"), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
