package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

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

	if s.db.UserExists(r.Context(), auth.Username, auth.Email) {
		errorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	user, err := s.kn.AddUser(r.Context(), auth.Username, auth.Email, auth.Password)
	if err != nil {
		fmt.Println(err)
		errorData(w, "Couldn't create user", 500)
		return
	}

	sid, err := s.kn.CreateSession(user.ID)
	if err != nil {
		log.Println(err)
		errorData(w, "Could not set session", 500)
		return
	}
	returnData(w, sid)
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

// TODO: Move login logic to internal/logic
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
		errorData(w, "User not found", http.StatusBadRequest)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auth.Password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		errorData(w, "Invalid username or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}

	/*
		NOTE: This is how a cookie should look like (set by the frontend)
			cookie := &http.Cookie{
				Name:     "kn-sessionid",
				Value:    sid,
				Path:     "/",
				HttpOnly: false,
				SameSite: http.SameSiteDefaultMode,
				Expires:  time.Now().Add(time.Hour * 24 * 30),
			}
	*/

	sid, err := s.kn.CreateSession(user.ID)
	if err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}
	returnData(w, sid)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	s.kn.RemoveSessionCookie(w, r)
}
