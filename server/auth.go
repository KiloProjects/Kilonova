package server

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string
		Email    string
		Password string
	}
	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if auth.Email == "" || auth.Username == "" || auth.Password == "" {
		errorData(w, "You must specify an email address, username and password", http.StatusBadRequest)
		return
	}

	if strings.IndexFunc(auth.Username, unicode.IsSpace) != -1 {
		errorData(w, "Username must not contain spaces", http.StatusBadRequest)
		return
	}

	if strings.IndexFunc(auth.Email, unicode.IsSpace) != -1 || len(auth.Email) > 254 || len(auth.Email) < 3 {
		errorData(w, "Invalid e-mail address", http.StatusBadRequest)
		return
	}

	if s.db.UserExists(auth.Email, auth.Username) {
		errorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	user, err := s.db.RegisterUser(auth.Email, auth.Username, auth.Password)
	if err != nil {
		errorData(w, "Couldn't register user", 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		s.logger.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, encoded)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string
		Password string
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
	}

	if auth.Password == "" || auth.Username == "" {
		errorData(w, "You must specify an username and a password", http.StatusBadRequest)
		return
	}

	var user *models.User
	quser, err := s.db.GetUserByName(auth.Username)
	if err != nil {
		errorData(w, "user not found", http.StatusBadRequest)
		return
	}
	user = quser
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auth.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		errorData(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		s.logger.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}

	encoded, err := common.SetSession(w, common.Session{UserID: user.ID})
	if err != nil {
		s.logger.Println(err)
		errorData(w, http.StatusText(500), 500)
		return
	}
	returnData(w, encoded)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	common.RemoveSessionCookie(w)
}
