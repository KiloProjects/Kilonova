package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
)

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if common.IsAuthed(r) {
			errorData(w, "You must not be logged in to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return s.MustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsAdmin(r) {
			errorData(w, "You must be an admin to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsAuthed(r) {
			errorData(w, "You must be authenticated to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeProposer is middleware to make sure the user creating the request is a proposer
func (s *API) MustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsProposer(r) {
			errorData(w, "You must be a proposer to do this", http.StatusUnauthorized)
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
			errorData(w, http.StatusText(500), 500)
			return
		}
		ctx := context.WithValue(r.Context(), common.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *API) validateProblemEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !common.IsProblemEditor(r) {
			errorData(w, "You must be authorized to edit the problem", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateProblemID pre-emptively returns if there isnt a valid problem ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) validateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			errorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		problem, err := s.db.GetProblemByID(uint(problemID))
		if err != nil {
			errorData(w, "problem does not exist", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), common.PbID, uint(problemID))
		ctx = context.WithValue(ctx, common.ProblemKey, problem)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
