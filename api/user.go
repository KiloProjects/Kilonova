package api

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/pkg/errors"
)

var (
	errUserNotFound = errors.New("User Not Found")
)

func (s *API) getSelfGravatar(w http.ResponseWriter, r *http.Request) {
	email := util.User(r).Email
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	url := getGravatarFromEmail(email) + "?s=" + size
	etag := `KN-Gravatar "` + util.User(r).Name + `"`
	w.Header().Set("Etag", etag)
	w.Header().Add("Cache-Control", "max-age=1440")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	http.Redirect(w, r, url, http.StatusPermanentRedirect)
}

func (s *API) getGravatar(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	user, err := s.db.UserByName(r.Context(), name)
	if err != nil {
		errorData(w, err, http.StatusNotFound)
		return
	}
	email := user.Email

	url := getGravatarFromEmail(email) + "?s=" + size
	etag := `KN-Gravatar "` + user.Name + `"`
	w.Header().Set("Etag", etag)
	w.Header().Add("Cache-Control", "max-age=1440")

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	http.Redirect(w, r, url, http.StatusPermanentRedirect)
}

func (s *API) setSubVisibility(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Visibility bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if err := util.User(r).SetDefaultVisibility(args.Visibility); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Visibility {
		returnData(w, "Made visible")
	} else {
		returnData(w, "Made invisible")
	}
}

func (s *API) setBio(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Bio string }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if err := util.User(r).SetBio(args.Bio); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated bio")
}

func (s *API) purgeBio(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int64 }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.db.User(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := user.SetBio(""); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Removed bio")
}

func (s *API) getUserByName(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	user, err := s.db.UserByName(r.Context(), name)
	if err != nil || user.ID == 0 {
		errorData(w, "User not found", http.StatusNotFound)
		return
	}
	returnData(w, user)

}

func (s *API) getSelf(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.User(r))
}

func (s *API) getSelfSolvedProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := util.User(r).SolvedProblems()
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, pbs)
}

func getGravatarFromEmail(email string) string {
	bSum := md5.Sum([]byte(email))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(bSum[:])
}

func (s *API) changePassword(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		errorData(w, "You must provide a new password", http.StatusBadRequest)
		return
	}

	hash, err := s.kn.GenHash(password)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := util.User(r).SetPasswordHash(hash); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Successfully changed password")
}

// ChangeEmail changes the e-mail of the saved user
// TODO: Check this is not a scam and the user actually wants to change email
func (s *API) changeEmail(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	if email == "" {
		errorData(w, "You must provide a new email to change to", http.StatusBadRequest)
		return
	}
	if err := util.User(r).SetEmail(email); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Successfully changed email")
}
