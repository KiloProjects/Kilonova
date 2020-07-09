package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

// RegisterAuthRoutes registers the Authentication routes on /api/auth
func (s *API) RegisterAuthRoutes() chi.Router {
	r := chi.NewRouter()
	// /auth/logout
	r.With(s.MustBeAuthed).Post("/logout", s.Logout)
	// /auth/signup
	r.With(s.MustBeVisitor).Post("/signup", s.Signup)
	// /auth/login
	r.With(s.MustBeVisitor).Post("/login", s.Login)
	return r
}

// Signup creates a new user based on the request data
func (s *API) Signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := strings.ToLower(r.FormValue("email"))
	username := r.FormValue("username")
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		s.ErrorData(w, "You must specify an email address, username and password", http.StatusBadRequest)
		return
	}

	if s.db.UserExists(email, username) {
		s.ErrorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	user, err := s.db.RegisterUser(email, username, password)
	if err != nil {
		s.errlog("Couldn't register user: %s", err)
		s.ErrorData(w, "Couldn't register user", 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		fmt.Println(err)
		s.ErrorData(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", encoded)
}

// Login creates a new Session while checking that the user credentials are correct
func (s *API) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if s.config.Debug {
		fmt.Println(username, password)
	}

	if password == "" || username == "" {
		s.ErrorData(w, "You must specify an username and a password", http.StatusBadRequest)
		return
	}

	var user *common.User
	quser, err := s.db.GetUserByName(username)
	if err != nil {
		s.ErrorData(w, "user not found", http.StatusBadRequest)
		return
	}
	user = quser
	spew.Config.Dump(user)
	fmt.Println(user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		s.ErrorData(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		fmt.Println(err)
		s.ErrorData(w, http.StatusText(500), 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		fmt.Println(err)
		s.ErrorData(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", encoded)
}

// Logout removes the session cookie
func (s *API) Logout(w http.ResponseWriter, r *http.Request) {
	common.RemoveSessionCookie(w)
}
