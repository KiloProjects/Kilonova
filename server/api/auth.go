package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

// RegisterAuthRoutes registers the Authentication routes on /api/auth
func (s *API) RegisterAuthRoutes() chi.Router {
	r := chi.NewRouter()
	r.With(s.MustBeAuthed).Post("/logout", s.Logout)
	r.With(s.MustBeVisitor).Post("/signup", s.Signup)
	r.With(s.MustBeVisitor).Post("/login", s.Login)
	return r
}

// middleware

// Signup creates a new user based on the request data
func (s *API) Signup(w http.ResponseWriter, r *http.Request) {
	var user models.User
	r.ParseForm()
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		http.Error(w, "You must specify an email address, username and password", http.StatusBadRequest)
		return
	}
	email = strings.ToLower(email)
	var foundUser models.User

	s.db.Find(&foundUser, "email = ? OR lower(name) = lower(?)", email, username)
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

	encoded, err := s.SetSession(w, models.Session{IsAdmin: user.IsAdmin, UserID: user.ID})
	if err != nil {

		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", encoded)
}

// Login creates a new Session while checking that the user credentials are correct
func (s *API) Login(w http.ResponseWriter, r *http.Request) {
	spew.Dump(r.Form)
	username := r.FormValue("username")
	password := r.FormValue("password")

	fmt.Println(username, password)

	if password == "" || username == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var user models.User
	s.db.First(&user, "lower(name) = lower(?)", username)
	if user.ID == 0 {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	spew.Dump(user)
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	encoded, err := s.SetSession(w, models.Session{IsAdmin: user.IsAdmin, UserID: user.ID})
	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", encoded)
}

// Logout removes the session cookie
func (s *API) Logout(w http.ResponseWriter, r *http.Request) {
	s.RemoveSessionCookie(w)
}

// IsAuthed reads the session and says if the requester is authenticated
func (s *API) IsAuthed(r *http.Request) bool {
	session := s.GetSession(r)
	if session == nil {
		return false
	}
	return session.UserID != 0
}

// IsAdmin reads the session and says if the requester is an admin
func (s *API) IsAdmin(r *http.Request) bool {
	session := s.GetSession(r)
	if session == nil {
		return false
	}
	return session.IsAdmin
}
