package web

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi"
)

// ValidateProblemID makes sure the problem ID is a valid uint
func (rt *Web) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(chi.URLParam(r, "pbid"))
		if err != nil {
			statusPage(w, r, http.StatusBadRequest, "ID invalid", false)
			return
		}
		problem, err := rt.db.Problem(r.Context(), problemID)
		if err != nil {
			log.Println("ValidateProblemID:", err)
			statusPage(w, r, 500, "", false)
			return
		}
		if problem == nil {
			statusPage(w, r, 404, "Problema nu a fost găsită", false)
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
			statusPage(w, r, http.StatusBadRequest, "ID invalid", false)
			return
		}
		list, err := rt.db.ProblemList(r.Context(), listID)
		if err != nil {
			log.Println("ValidateListID:", err)
			statusPage(w, r, 500, "", false)
			return
		}
		if list == nil {
			statusPage(w, r, 404, "Lista nu a fost găsită", false)
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
			log.Println("ValidateAttachmentID:", err)
			statusPage(w, r, 500, "", false)
			return
		}
		if att == nil {
			http.Error(w, "Atașamentul nu a fost găsit", 404)
			return
		}
		if att.Private && !util.IsRProblemEditor(r) {
			statusPage(w, r, 403, "Nu ai voie să accesezi acest atașament", true)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, att)))
	})
}

// ValidateVisible checks if the problem from context is visible from the logged in user
func (rt *Web) ValidateVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemVisible(r) {
			statusPage(w, r, 403, "Nu ai voie să accesezi problema!", true)
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
			statusPage(w, r, 400, "ID submisie invalid", false)
			return
		}
		sub, err := rt.db.Submission(r.Context(), subID)
		if err != nil {
			log.Println(err)
			statusPage(w, r, 500, "", false)
			return
		}
		if sub == nil {
			statusPage(w, r, 400, "Submisia nu există", false)
			return
		}

		pb, err := rt.db.Problem(r.Context(), sub.ProblemID)
		if err != nil {
			log.Println(err)
			statusPage(w, r, 500, "", false)
			return
		}
		if !util.IsProblemVisible(util.User(r), pb) {
			statusPage(w, r, 403, "Nu poți accesa această submisie!", true)
			return
		}

		if !util.IsSubmissionVisible(sub, util.User(r), rt.db) {
			sub.Code = ""
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubKey, sub)))
	})
}

func mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAuthed(r) {
			statusPage(w, r, 401, "Trebuie să fii logat", true)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProposer(r) {
			statusPage(w, r, 401, "Trebuie să fii propunător", true)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustBeAdmin(next http.Handler) http.Handler {
	return mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRAdmin(r) {
			statusPage(w, r, 401, "Trebuie să fii admin", true)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.IsRAuthed(r) {
			statusPage(w, r, 401, "Trebuie să fii delogat", false)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustBeEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.IsRProblemEditor(r) {
			statusPage(w, r, 401, "Trebuie să fii autorul problemei", true)
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
		sess, err := rt.db.GetSession(r.Context(), getSessCookie(r))
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
