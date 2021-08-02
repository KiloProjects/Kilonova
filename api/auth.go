package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"golang.org/x/crypto/bcrypt"
)

var unameValidation = []validation.Rule{validation.Required, validation.Length(3, 32), is.PrintableASCII}
var pwdValidation = []validation.Rule{validation.Required, validation.Length(6, 64)}
var langValidation = []validation.Rule{validation.In("", "en", "ro")}

type signupForm struct {
	Username string
	Email    string
	Password string
	Language string
}

func (s signupForm) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.Username, unameValidation...),
		validation.Field(&s.Email, validation.Required, is.Email),
		validation.Field(&s.Password, pwdValidation...),
		validation.Field(&s.Language, validation.In("", "en", "ro")),
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

	if exists, err := s.db.UserExists(r.Context(), auth.Username, auth.Email); err != nil || exists {
		errorData(w, "User matching email or username already exists", http.StatusBadRequest)
		return
	}

	lang := auth.Language
	if lang == "" {
		lang = config.Common.DefaultLang
	}

	user, err := s.addUser(r.Context(), auth.Username, auth.Email, auth.Password, lang)
	if err != nil {
		fmt.Println(err)
		errorData(w, "Couldn't create user", 500)
		return
	}

	if err := kilonova.SendVerificationEmail(user, s.db, s.mailer); err != nil {
		log.Println("Couldn't send user verification email:", err)
		return
	}

	sid, err := s.db.CreateSession(r.Context(), user.ID)
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

	users, err := s.db.Users(r.Context(), kilonova.UserFilter{Name: &auth.Username, Limit: 1})
	if err != nil || len(users) == 0 {
		errorData(w, "User not found", http.StatusBadRequest)
		return
	}
	user := users[0]

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

	sid, err := s.db.CreateSession(r.Context(), user.ID)
	if err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}
	returnData(w, sid)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	h := getAuthHeader(r)
	if h == "" {
		errorData(w, "You are already logged out!", 400)
		return
	}
	s.db.RemoveSession(r.Context(), h)
	returnData(w, "Logged out")
}

func (s *API) addUser(ctx context.Context, username, email, password, lang string) (*kilonova.User, error) {
	hash, err := kilonova.HashPassword(password)
	if err != nil {
		return nil, err
	}

	var user kilonova.User
	user.Name = username
	user.Email = email
	user.Password = hash
	user.PreferredLanguage = lang

	err = s.db.CreateUser(ctx, &user)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if user.ID == 1 {
		var True = true
		if err := s.db.UpdateUser(ctx, user.ID, kilonova.UserUpdate{Admin: &True, Proposer: &True}); err != nil {
			log.Println(err)
			return &user, err
		}
	}

	return &user, nil
}
