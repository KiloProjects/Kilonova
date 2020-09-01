package server

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/pkg/errors"
)

var (
	errUserNotFound = errors.New("User Not Found")
)

func (s *API) getSelfGravatar(w http.ResponseWriter, r *http.Request) {
	email := common.UserFromContext(r).Email
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	http.Redirect(w, r, getGravatarFromEmail(email)+"?s="+size, http.StatusTemporaryRedirect)
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
	user, err := s.db.GetUserByName(name)
	if err != nil {
		errorData(w, err, http.StatusNotFound)
		return
	}
	http.Redirect(w, r, getGravatarFromEmail(user.Email)+"?s="+size, http.StatusTemporaryRedirect)
}

func (s *API) getUserByName(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	user, err := s.db.GetUserByName(name)
	if err != nil {
		errorData(w, err, http.StatusNotFound)
		return
	}
	returnData(w, user)

}

func (s *API) getSelf(w http.ResponseWriter, r *http.Request) {
	returnData(w, common.UserFromContext(r))
}

func getGravatarFromEmail(email string) string {
	bSum := md5.Sum([]byte(email))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(bSum[:])
}

// ChangeEmail changes the e-mail of the saved user
func (s *API) changeEmail(w http.ResponseWriter, r *http.Request) {
	user := common.UserFromContext(r)
	email := r.FormValue("email")
	if email == "" {
		errorData(w, "You must provide a new email to change to", http.StatusBadRequest)
		return
	}
	if err := s.db.SetEmail(user.ID, email); err != nil {
		errorData(w, http.StatusText(500), 500)
	}
	returnData(w, "Successfully changed email")
}
