package web

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi"
)

type retData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ValidateProblemID makes sure the problem ID is a valid uint
func (rt *Web) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(chi.URLParam(r, "pbid"))
		if err != nil {
			rt.status(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
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
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemKey, problem)))
	})

}

// ValidateListID makes sure the list ID is a valid uint
func (rt *Web) ValidateListID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			rt.status(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		list, err := rt.db.ProblemList(r.Context(), listID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 404, "Lista nu a fost găsită")
				return
			}
			log.Println("ValidateListID:", err)
			rt.status(w, r, 500, "")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemListKey, list)))
	})
}

// ValidateAttachmentID makes sure the attachment ID is a valid uint
func (rt *Web) ValidateAttachmentID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attid, err := strconv.Atoi(chi.URLParam(r, "aid"))
		if err != nil {
			http.Error(w, "ID invalid", 400)
			return
		}
		att, err := rt.db.Attachment(r.Context(), attid)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Atașamentul nu a fost găsit", 404)
				return
			}
			log.Println("ValidateAttachmentID:", err)
			http.Error(w, http.StatusText(500), 500)
			rt.status(w, r, 500, "")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, att)))
	})
}

func (rt *Web) ValidateSubTaskID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subtaskID, err := strconv.Atoi(chi.URLParam(r, "stid"))
		if err != nil {
			rt.status(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		subtask, err := rt.db.SubTask(r.Context(), util.Problem(r).ID, subtaskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 404, "SubTask-ul nu există")
				return
			}
			log.Println("ValidateSubTaskID:", err)
			rt.status(w, r, 500, "")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubTaskKey, subtask)))
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
		subID, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			rt.status(w, r, 400, "ID submisie invalid")
			return
		}
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

		pb, err := rt.db.Problem(r.Context(), sub.ProblemID)
		if err != nil {
			log.Println(err)
			rt.status(w, r, 500, "")
			return
		}
		if !util.IsProblemVisible(util.User(r), pb) {
			rt.status(w, r, 403, "Nu poți accesa această submisie!")
			return
		}

		if !util.IsSubmissionVisible(sub, util.User(r), rt.db) {
			sub.Code = ""
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubKey, sub)))
	})
}

// ValidateTestID checks for the correctness of the test ID and adds it to context if ok
func (rt *Web) ValidateTestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testID, err := strconv.Atoi(chi.URLParam(r, "tid"))
		if err != nil {
			rt.status(w, r, 400, "Test invalid")
			return
		}
		test, err := rt.db.Test(r.Context(), util.Problem(r).ID, testID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				rt.status(w, r, 404, "Testul nu există")
				return
			}
			log.Println(err)
			rt.status(w, r, 500, "")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TestKey, test)))
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

func getSessCookie(r *http.Request) string {
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (rt *Web) getUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := rt.kn.GetSession(getSessCookie(r))
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := rt.db.User(r.Context(), sess)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user.Password = ""
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.UserKey, user)))
	})
}
