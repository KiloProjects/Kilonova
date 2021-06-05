package api

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/microcosm-cc/bluemonday"
)

var (
	errUserNotFound = &kilonova.Error{Code: kilonova.ENOTFOUND, Message: "User not found"}
)

func (s *API) serveGravatar(w http.ResponseWriter, r *http.Request, user *kilonova.User, size int) {
	v := url.Values{}
	v.Add("s", strconv.Itoa(size))
	v.Add("d", "identicon")

	bSum := md5.Sum([]byte(user.Email))
	url := "https://www.gravatar.com/avatar/" + hex.EncodeToString(bSum[:]) + "?" + v.Encode()
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		errorData(w, err, 500)
		return
	}
	// get the name
	_, params, _ := mime.ParseMediaType(resp.Header.Get("content-disposition"))
	name := params["filename"]

	// get the modtime
	time, _ := http.ParseTime(resp.Header.Get("last-modified"))

	// get the image
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	reader := bytes.NewReader(buf)
	resp.Body.Close()

	w.Header().Add("ETag", fmt.Sprintf("\"kn-%s-%d\"", user.Name, time.Unix()))
	// Cache for 1 day
	w.Header().Add("cache-control", "public, max-age=86400, immutable")

	http.ServeContent(w, r, name, time, reader)
}

func (s *API) getSelfGravatar(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.Atoi(r.FormValue("s"))
	if err != nil || size == 0 {
		size = 128
	}
	s.serveGravatar(w, r, util.User(r), size)
}

func (s *API) getGravatar(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	size, err := strconv.Atoi(r.FormValue("s"))
	if err != nil || size == 0 {
		size = 128
	}
	users, err := s.db.Users(r.Context(), kilonova.UserFilter{Name: &name, Limit: 1})
	if err != nil || len(users) == 0 {
		errorData(w, err, http.StatusNotFound)
		return
	}
	s.serveGravatar(w, r, users[0], size)
}

func (s *API) setSubVisibility(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ Visibility bool }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	if err := s.db.UpdateUser(
		r.Context(),
		util.User(r).ID,
		kilonova.UserUpdate{DefaultVisible: &args.Visibility},
	); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Visibility {
		returnData(w, "Made visible")
	} else {
		returnData(w, "Made invisible")
	}
}

func (s *API) setBio() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct{ Bio string }
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, 400)
			return
		}

		safe := strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(args.Bio))

		if err := s.db.UpdateUser(
			r.Context(),
			util.User(r).ID,
			kilonova.UserUpdate{Bio: &safe},
		); err != nil {
			errorData(w, err, 500)
			return
		}

		returnData(w, "Updated bio")
	}
}

func (s *API) purgeBio(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID  int
		Bio string
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.db.UpdateUser(
		r.Context(),
		args.ID,
		kilonova.UserUpdate{Bio: &args.Bio},
	); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated bio")
}
func (s *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.db.User(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if user.Admin {
		errorData(w, "You can't erase a fellow admin! Unmod him first", 400)
		return
	}

	if err := s.db.DeleteUser(r.Context(), args.ID); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Deleted user")
}

func (s *API) getUserByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	users, err := s.db.Users(r.Context(), kilonova.UserFilter{Name: &name, Limit: 1})
	if err != nil || len(users) == 0 {
		errorData(w, "User not found", http.StatusNotFound)
		return
	}
	returnData(w, users[0])

}

func (s *API) getSelf(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.User(r))
}

func (s *API) getSelfSolvedProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := kilonova.SolvedProblems(r.Context(), util.User(r).ID, s.db)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, pbs)
}

func (s *API) getSolvedProblems(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		errorData(w, "Name not specified", http.StatusBadRequest)
		return
	}
	users, err := s.db.Users(r.Context(), kilonova.UserFilter{Name: &name, Limit: 1})
	if err != nil || len(users) == 0 {
		errorData(w, "User not found", http.StatusNotFound)
		return
	}
	pbs, err := kilonova.SolvedProblems(r.Context(), util.User(r).ID, s.db)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, util.FilterVisible(util.User(r), pbs))
}

// ChangeEmail changes the password of the saved user
// TODO: Check this is not a scam and the user actually wants to change password
func (s *API) changePassword(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	if password == "" {
		errorData(w, "You must provide a new password", http.StatusBadRequest)
		return
	}

	hash, err := kilonova.HashPassword(password)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.db.UpdateUser(
		r.Context(),
		util.User(r).ID,
		kilonova.UserUpdate{PwdHash: &hash},
	); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Successfully changed password")
}

// ChangeEmail changes the e-mail of the saved user
// TODO: Check this is not a scam and the user actually wants to change email
func (s *API) changeEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		errorData(w, "You must provide a new email to change to", 400)
		return
	}
	if err := validation.Validate(&email, is.Email); err != nil {
		errorData(w, "Invalid email", 400)
	}
	if err := s.db.UpdateUser(
		r.Context(),
		util.User(r).ID,
		kilonova.UserUpdate{Email: &email},
	); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Successfully changed email")
}

func (s *API) resendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	u := util.User(r)
	if u.VerifiedEmail {
		errorData(w, "You already have a verified email!", 400)
		return
	}

	if time.Now().Sub(u.EmailVerifSentAt.Time) < 5*time.Minute {
		errorData(w, "You must wait 5 minutes before sending another verification email!", 400)
		return
	}

	if err := kilonova.SendVerificationEmail(u.Email, u.Name, u.ID, s.db, s.mailer); err != nil {
		log.Println(err)
		errorData(w, "Couldn't send verification email", 500)
		return
	}

	now := time.Now()
	if err := s.db.UpdateUser(
		r.Context(),
		util.User(r).ID,
		kilonova.UserUpdate{EmailVerifSentAt: &now},
	); err != nil {
		log.Println("Couldn't update verification email timestamp:", err)
	}
	returnData(w, "Verification email resent")
}
