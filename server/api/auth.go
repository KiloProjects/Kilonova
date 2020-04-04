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
	r.HandleFunc("/logout", s.logout)
	r.With(s.mustBeVisitor).HandleFunc("/signup", s.signup)
	r.With(s.mustBeVisitor).HandleFunc("/login", s.login)
	return r
}

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	var user models.User
	r.ParseForm()
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")
	if email == "" || username == "" || password == "" {
		http.Error(w, "You must specify an email address, username and password", http.StatusBadRequest)
		return
	}

	var foundUser models.User
	s.db.Find(&foundUser, "email = ? OR name = ?", email, username)
	if foundUser.ID > 0 {
		http.Error(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}
	user.Name = username
	user.Email = email
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Could not hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashed)
	s.db.Create(&user)

	s.SetSessionCookie(w, models.Session{IsAdmin: user.IsAdmin, UserID: user.ID})
	w.WriteHeader(http.StatusOK)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	spew.Dump(r.Form)
	username := r.FormValue("username")
	password := r.FormValue("password")

	var user models.User
	s.db.First(&user, "name = ?", username)
	spew.Dump(user)
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	fmt.Println(err)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	s.SetSessionCookie(w, models.Session{IsAdmin: user.IsAdmin, UserID: user.ID})
	w.WriteHeader(http.StatusOK)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	s.RemoveSessionCookie(w)
	w.WriteHeader(http.StatusOK)
}
