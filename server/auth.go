package server

import (
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
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

	user, err := s.db.GetUserByName(auth.Username)
	if err != nil {
		errorData(w, "user not found", http.StatusBadRequest)
		return
	}

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
