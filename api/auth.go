package api

import (
	"fmt"
	"net/http"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

func (s *API) registerAuth() chi.Router {
	r := chi.NewRouter()
	r.Post("/signup", s.signup)
	r.Post("/login", s.login)

	return r
}

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	var user models.User
	r.ParseForm()
	var foundUser models.User
	s.db.Find(&foundUser, "email = ? OR name = ?", r.Form.Get("email"), r.Form.Get("username"))
	user.Name = r.Form.Get("username")
	user.Email = r.Form.Get("email")
	hashed, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	user.Password = string(hashed)
	s.db.Create(&user)

	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	spew.Dump(r.PostForm)
	w.Header().Set("location", "http://localhost:3000/")
	w.WriteHeader(http.StatusMovedPermanently)
}
