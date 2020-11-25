package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/cookie"
	"github.com/KiloProjects/Kilonova/internal/db"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"golang.org/x/crypto/bcrypt"
)

var unameValidation = []validation.Rule{validation.Required, validation.Length(3, 32), is.PrintableASCII}
var pwdValidation = []validation.Rule{validation.Required, validation.Length(6, 64)}

type signupForm struct {
	Username string
	Email    string
	Password string
}

func (s signupForm) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Username, unameValidation...),
		validation.Field(&s.Email, validation.Required, is.Email),
		validation.Field(&s.Password, pwdValidation...),
	)
}

func userExists(number int64, err error) bool {
	if err != nil {
		log.Println(err)
	}
	if number > 0 {
		return true
	}
	return false
}

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth signupForm
	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := auth.Validate(); err != nil {
		errorData(w, err.Error(), http.StatusBadRequest)
		return
	}
	cnt, err := s.db.CountUsers(r.Context(), db.CountUsersParams{Username: auth.Username, Email: auth.Email})
	fmt.Println(cnt, err)
	if userExists(cnt, err) {
		errorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(auth.Password), bcrypt.DefaultCost)
	if err != nil {
		errorData(w, "Couldn't create password hash", 500)
		return
	}

	user, err := s.db.CreateUser(r.Context(), db.CreateUserParams{Email: auth.Email, Name: auth.Username, Password: string(hashed)})
	if err != nil {
		errorData(w, "Couldn't create user", 500)
		return
	}

	if user == 1 {
		if err := s.db.SetAdmin(r.Context(), db.SetAdminParams{ID: user, Admin: true}); err != nil {
			fmt.Println(err)
		}
		if err := s.db.SetProposer(r.Context(), db.SetProposerParams{ID: user, Proposer: true}); err != nil {
			fmt.Println(err)
		}
	}

	encoded, err := cookie.SetSession(w, cookie.Session{UserID: user})
	if err != nil {
		s.logger.Println(err)
		fmt.Println(err)
		errorData(w, "Could not set session", 500)
		return
	}
	returnData(w, encoded)
}

type loginForm struct {
	Username string
	Password string
}

func (l loginForm) Validate() error {
	return validation.ValidateStruct(&l,
		validation.Field(&l.Username, unameValidation...),
		validation.Field(&l.Password, pwdValidation...),
	)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth loginForm

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := auth.Validate(); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	user, err := s.db.UserByName(r.Context(), auth.Username)
	if err != nil {
		errorData(w, "user not found", http.StatusBadRequest)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auth.Password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		errorData(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		s.logger.Println(err)
		errorData(w, err, 500)
		return
	}

	encoded, err := cookie.SetSession(w, cookie.Session{UserID: user.ID})
	if err != nil {
		s.logger.Println(err)
		errorData(w, err, 500)
		return
	}
	returnData(w, encoded)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	cookie.RemoveSessionCookie(w)
}
