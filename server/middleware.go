package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"gorm.io/gorm"
)

func (s *API) isAuthed(r *http.Request) bool {
	user := common.UserFromContext(r)
	if user.ID == 0 {
		return false
	}
	return true
}

func (s *API) isAdmin(r *http.Request) bool {
	if !s.isAuthed(r) {
		return false
	}
	user := common.UserFromContext(r)
	if user.ID == 1 {
		return true
	}
	return user.IsAdmin
}

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.isAuthed(r) {
			s.ErrorData(w, "You must not be logged in to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return s.MustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isAdmin(r) {
			s.ErrorData(w, "You must be an admin to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.isAuthed(r) {
			s.ErrorData(w, "You must be authenticated to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupSession adds the user with the specified user ID to context
func (s *API) SetupSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := common.GetSession(r)
		if session == nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := s.db.GetUserByID(session.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				next.ServeHTTP(w, r)
				return
			}
			s.ErrorData(w, http.StatusText(500), 500)
			return
		}
		ctx := context.WithValue(r.Context(), common.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
