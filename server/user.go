package server

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
)

var (
	errUserNotFound = errors.New("User Not Found")
)

// RegisterUserRoutes mounts the user-related routes at /api/user
func (s *API) RegisterUserRoutes() chi.Router {
	r := chi.NewRouter()
	// /user/getByName
	r.Get("/getByName", s.GetUserByName)
	// /user/getSelf
	r.With(s.MustBeAuthed).Get("/getSelf", s.GetSelf)
	// /user/getGravatar
	r.Get("/getGravatar", s.GetGravatar)
	// /user/getSelfGravatar
	r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.GetSelfGravatar)
	// /user/changeEmail
	r.With(s.MustBeAuthed).Post("/changeEmail", s.ChangeEmail)
	return r
}

// GetSelfGravatar returns the gravatar of the authenticated user
func (s *API) GetSelfGravatar(w http.ResponseWriter, r *http.Request) {
	email := common.UserFromContext(r).Email
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	http.Redirect(w, r, s.getGravatarFromEmail(email)+"?s="+size, http.StatusTemporaryRedirect)
}

// GetGravatar returns a gravatar from a specified user
func (s *API) GetGravatar(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		s.ErrorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	user, err := s.db.GetUserByName(name)
	if err != nil {
		s.ErrorData(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Redirect(w, r, s.getGravatarFromEmail(user.Email)+"?s="+size, http.StatusTemporaryRedirect)
}

// GetUserByName returns a user based on a specified name
func (s *API) GetUserByName(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		s.ErrorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	user, err := s.db.GetUserByName(name)
	if err != nil {
		s.ErrorData(w, err.Error(), http.StatusNotFound)
		return
	}
	s.ReturnData(w, "success", user)

}

// GetSelf returns the authenticated user
func (s *API) GetSelf(w http.ResponseWriter, r *http.Request) {
	spew.Dump(common.UserFromContext(r))
	s.ReturnData(w, "success", common.UserFromContext(r))
}

func (s *API) getGravatarFromEmail(email string) string {
	bSum := md5.Sum([]byte(email))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(bSum[:])
}

// ChangeEmail changes the e-mail of the saved user
func (s *API) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	user := common.UserFromContext(r)
	email := r.FormValue("email")
	if email == "" {
		s.ErrorData(w, "You must provide a new email to change to", http.StatusBadRequest)
		return
	}
	if err := s.db.SetEmail(user.ID, email); err != nil {
		// shouldn't happen, so log it
		s.errlog("/user/changeEmail Couldn't change email of user with id %d: %s", user.ID, email)
		s.ReturnData(w, http.StatusText(500), 500)
	}
}
