package api

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"

	"github.com/KiloProjects/Kilonova/models"
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
)

var (
	errUserNotFound = errors.New("User Not Found")
)

// RegisterUserRoutes mounts the user-related routes at /api/user
func (s *API) RegisterUserRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getByName", s.GetUserByName)
	r.With(s.MustBeAuthed).Get("/getSelf", s.GetSelf)
	r.Get("/getGravatar", s.GetGravatar)
	r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.GetSelfGravatar)
	return r
}

// GetSelfGravatar returns the gravatar of the authenticated user
func (s *API) GetSelfGravatar(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(models.KNContextType("user")).(models.User)
	size := r.FormValue("s")
	if size == "" {
		size = "128"
	}
	http.Redirect(w, r, s.getGravatarURLFromEmail(user.Email)+"?s="+size, http.StatusMovedPermanently)
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
	user, err := s.getUserByName(name)
	if err != nil {
		s.ErrorData(w, err.Error(), http.StatusNotFound)
		return
	}
	http.Redirect(w, r, s.getGravatarURLFromEmail(user.Email)+"?s="+size, http.StatusMovedPermanently)
}

// GetUserByName returns a user based on a specified name
func (s *API) GetUserByName(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name == "" {
		s.ErrorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	user, err := s.getUserByName(name)
	if err != nil {
		s.ErrorData(w, err.Error(), http.StatusNotFound)
		return
	}
	s.ReturnData(w, "success", user)

}

// GetSelf returns the authenticated user
func (s *API) GetSelf(w http.ResponseWriter, r *http.Request) {
	s.ReturnData(w, "success", s.getContextValue(r, "user"))
}

func (s *API) getGravatarURLFromEmail(email string) string {
	bSum := md5.Sum([]byte(email))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(bSum[:])
}

func (s *API) getUserByName(name string) (*models.User, error) {
	var user models.User
	s.db.First(&user, "lower(name) = lower(?)", name)
	if user.ID == 0 {
		return nil, errUserNotFound
	}
	return &user, nil
}
