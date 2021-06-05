package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi"
)

// MustBeVisitor is middleware to make sure the user creating the request is not authenticated
func (s *API) MustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.IsRAuthed(r) {
			errorData(w, "You must not be logged in to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeAdmin is middleware to make sure the user creating the request is an admin
func (s *API) MustBeAdmin(next http.Handler) http.Handler {
	return s.MustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAdmin(r) {
			errorData(w, "You must be an admin to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// MustBeAuthed is middleware to make sure the user creating the request is authenticated
func (s *API) MustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAuthed(r) {
			errorData(w, "You must be authenticated to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MustBeProposer is middleware to make sure the user creating the request is a proposer
func (s *API) MustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProposer(r) {
			errorData(w, "You must be a proposer to do this", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SetupSession adds the user with the specified user ID to context
func (s *API) SetupSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetRSession(r, s.db)
		if session == -1 {
			next.ServeHTTP(w, r)
			return
		}
		user, err := s.db.User(r.Context(), session)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				next.ServeHTTP(w, r)
				return
			}
			fmt.Println(err)
			errorData(w, http.StatusText(500), 500)
			return
		}
		user.Password = ""
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.UserKey, user)))
	})
}

func (s *API) validateProblemEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemEditor(r) {
			errorData(w, "You must be authorized to edit the problem", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// validateTestID pre-emptively returns if there isnt a valid test ID in the URL params
// Also, it fetches the test from the DB and makes sure it exists
// NOTE: This does not fetch the test data from disk
func (s *API) validateTestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testID, err := strconv.Atoi(chi.URLParam(r, "tID"))
		if err != nil {
			errorData(w, "invalid test ID", http.StatusBadRequest)
			return
		}
		test, err := s.db.Test(r.Context(), util.Problem(r).ID, testID)
		if err != nil {
			errorData(w, "test does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
	})
}

func (s *API) validateAttachmentID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attID, err := strconv.Atoi(chi.URLParam(r, "aID"))
		if err != nil {
			errorData(w, "invalid attachment ID", http.StatusBadRequest)
			return
		}
		att, err := s.db.Attachment(r.Context(), attID)
		if err != nil {
			errorData(w, "attachment does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, att)))
	})
}

// validateProblemID pre-emptively returns if there isnt a valid problem ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) validateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			errorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		problem, err := s.db.Problem(r.Context(), problemID)
		if err != nil {
			errorData(w, "problem does not exist", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemKey, problem)))
	})
}
func GetRSession(r *http.Request, db kilonova.DB) int {
	authToken := getAuthHeader(r)
	if authToken != "" { // use Auth tokens by default
		id, err := db.GetSession(r.Context(), authToken)
		if err == nil {
			return id
		}
	}
	return -1
}

func getAuthHeader(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "guest" {
		header = ""
	}
	return header
}
