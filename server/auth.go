package server

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/bcrypt"
)

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := strings.ToLower(r.FormValue("email"))
	username := r.FormValue("username")
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		errorData(w, "You must specify an email address, username and password", http.StatusBadRequest)
		return
	}

	if strings.IndexFunc(username, unicode.IsSpace) != -1 {
		errorData(w, "Username must not contain spaces", http.StatusBadRequest)
		return
	}

	if strings.IndexFunc(email, unicode.IsSpace) != -1 || len(email) > 254 || len(email) < 3 {
		errorData(w, "Invalid e-mail address", http.StatusBadRequest)
		return
	}

	if s.db.UserExists(email, username) {
		errorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	user, err := s.db.RegisterUser(email, username, password)
	if err != nil {
		errorData(w, "Couldn't register user", 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		fmt.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "success", encoded)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if s.config.Debug {
		fmt.Println(username, password)
	}

	if password == "" || username == "" {
		errorData(w, "You must specify an username and a password", http.StatusBadRequest)
		return
	}

	var user *common.User
	quser, err := s.db.GetUserByName(username)
	if err != nil {
		errorData(w, "user not found", http.StatusBadRequest)
		return
	}
	user = quser
	spew.Config.Dump(user)
	fmt.Println(user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		errorData(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		fmt.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		fmt.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, "success", encoded)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	common.RemoveSessionCookie(w)
}
