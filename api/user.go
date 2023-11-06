package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
)

var (
	bm = bluemonday.StrictPolicy()
)

func (s *API) serveGravatar(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull, size int) {
	url := s.base.GetGravatarLink(user, size)
	resp, err := http.Get(url)
	if err != nil {
		zap.S().Warn(err)
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
		zap.S().Warn(err)
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
	s.serveGravatar(w, r, util.UserFull(r), size)
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
	user, err1 := s.base.UserFullByName(r.Context(), name)
	if err1 != nil {
		errorData(w, err, http.StatusNotFound)
		return
	}
	s.serveGravatar(w, r, user, size)
}

func (s *API) setPreferredLanguage() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct{ Language string }
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, 400)
			return
		}

		safe := strings.TrimSpace(bm.Sanitize(args.Language))
		if !(safe == "en" || safe == "ro") {
			errorData(w, "Invalid language", 400)
			return
		}

		if err := s.base.UpdateUser(
			r.Context(),
			util.UserBrief(r).ID,
			kilonova.UserUpdate{PreferredLanguage: safe},
		); err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, "Updated preferred default language")
	}
}

func (s *API) setPreferredTheme() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct{ Theme string }
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, 400)
			return
		}

		safe := strings.TrimSpace(bm.Sanitize(args.Theme))
		if !(safe == "light" || safe == "dark") {
			errorData(w, "Invalid language", 400)
			return
		}

		if err := s.base.UpdateUser(
			r.Context(),
			util.UserBrief(r).ID,
			kilonova.UserUpdate{PreferredTheme: kilonova.PreferredTheme(safe)},
		); err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, "Updated preferred default theme")
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

		safe := strings.TrimSpace(args.Bio)

		if err := s.base.UpdateUser(
			r.Context(),
			util.UserBrief(r).ID,
			kilonova.UserUpdate{Bio: &safe},
		); err != nil {
			err.WriteError(w)
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

	if err := s.base.UpdateUser(
		r.Context(),
		args.ID,
		kilonova.UserUpdate{Bio: &args.Bio},
	); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Purged bio")
}

func (s *API) manageUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID      int     `json:"id"`
		Lockout *bool   `json:"lockout"`
		NewName *string `json:"new_name"`

		ForceUsernameChange *bool `json:"force_username_change"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.base.UserBrief(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if args.NewName != nil && len(*args.NewName) > 2 && user.Name != *args.NewName {
		// Admins can change to formerly existing names
		if err := s.base.UpdateUsername(r.Context(), user.ID, *args.NewName, false, true); err != nil {
			err.WriteError(w)
			return
		}
	}

	if args.ForceUsernameChange != nil {
		if err := s.base.SetForceUsernameChange(r.Context(), user.ID, *args.ForceUsernameChange); err != nil {
			err.WriteError(w)
			return
		}
	}

	if args.Lockout != nil {
		if err := s.base.SetUserLockout(r.Context(), user.ID, *args.Lockout); err != nil {
			err.WriteError(w)
			return
		}
	}

	returnData(w, "Updated user")
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

	user, err := s.base.UserBrief(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if user.Admin {
		errorData(w, "You can't delete an admin account!", 400)
		return
	}

	if err := s.base.DeleteUser(r.Context(), user); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Deleted user")
}

func (s *API) getSelfSolvedProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := s.base.SolvedProblems(r.Context(), util.UserBrief(r), util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, pbs)
}

func (s *API) getSolvedProblems(w http.ResponseWriter, r *http.Request) {
	user, err := s.base.UserBriefByName(r.Context(), r.FormValue("name"))
	if err != nil {
		errorData(w, "User not found", http.StatusNotFound)
		return
	}
	pbs, err := s.base.SolvedProblems(r.Context(), user, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, pbs)
}

func (s *API) updateUsername(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID  *int   `json:"userID"`
		NewName string `json:"newName"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	user := util.UserBrief(r)
	if args.UserID != nil && util.UserBrief(r).Admin {
		newUser, err := s.base.UserBrief(r.Context(), *args.UserID)
		if err != nil {
			err.WriteError(w)
			return
		}
		user = newUser
	}

	if err := s.base.UpdateUsername(r.Context(), user.ID, args.NewName, !util.UserBrief(r).Admin, util.UserBrief(r).Admin || util.UserFull(r).NameChangeForced); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Username updated")
}

// changePassword changes the password of the saved user
// TODO: Check this is not a scam and the user actually wants to change password
func (s *API) changePassword(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Password    string `json:"password"`
		PasswordOld string `json:"old_password"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Password == "" {
		errorData(w, "You must provide a new password", http.StatusBadRequest)
		return
	}
	if err := s.base.VerifyUserPassword(r.Context(), util.UserBrief(r).ID, args.PasswordOld); err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.UpdateUserPassword(
		r.Context(),
		util.UserBrief(r).ID,
		args.Password,
	); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Successfully changed password")
}

func (s *API) sendForgotPwdMail(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Email string `json:"email"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	if args.Email == "" {
		errorData(w, "No email address specified", 400)
		return
	}
	if !s.base.MailerEnabled() {
		errorData(w, "Mailing subsystem was disabled by administrator", 400)
		return
	}
	user, err := s.base.UserFullByEmail(r.Context(), args.Email)
	if err != nil {
		if errors.Is(err, kilonova.ErrNotFound) {
			returnData(w, "If the provided email address is correct, an email should arrive shortly.")
			return
		}
		err.WriteError(w)
		return
	}
	go func(user *kilonova.UserFull) {
		if err := s.base.SendPasswordResetEmail(context.Background(), user.ID, user.Name, user.Email); err != nil {
			zap.S().Info(err)
		}
	}(user)

	returnData(w, "If the provided email address is correct, an email should arrive shortly.")
}

func (s *API) resetPassword(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		RequestID   string `json:"req_id"`
		NewPassword string `json:"password"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	if err := s.base.FinalizePasswordReset(r.Context(), args.RequestID, args.NewPassword); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Password reset. You may now log in with the updated credentials")
}

// ChangeEmail changes the email of the saved user
func (s *API) changeEmail(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	if email == "" {
		errorData(w, "You must provide the password and a new email to change to", 400)
		return
	}
	if err := validation.Validate(&email, is.Email); err != nil {
		errorData(w, "Invalid email", 400)
		return
	}

	if err := s.base.VerifyUserPassword(r.Context(), util.UserBrief(r).ID, password); err != nil {
		err.WriteError(w)
		return
	}

	user, err := s.base.UserFullByEmail(r.Context(), email)
	if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
		err.WriteError(w)
		return
	}
	if user != nil {
		if user.ID == util.UserBrief(r).ID {
			errorData(w, "You already use this email!", 400)
			return
		}
		errorData(w, "Email is already in use.", 400)
		return
	}

	if err := s.base.SendVerificationEmail(r.Context(), util.UserBrief(r).ID, util.UserBrief(r).Name, email); err != nil {
		if err.Code != 400 {
			zap.S().Warn(err)
		}
		err.WriteError(w)
		return
	}
	returnData(w, "Successfully changed email")
}

func (s *API) resendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	u := util.UserFull(r)
	if u.VerifiedEmail {
		errorData(w, "You already have a verified email!", 400)
		return
	}

	if time.Since(u.EmailVerifResent) < 5*time.Minute {
		errorData(w, "You must wait 5 minutes before sending another verification email!", 400)
		return
	}

	if err := s.base.SendVerificationEmail(r.Context(), u.ID, u.Name, u.Email); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Verification email resent")
}

func (s *API) generateUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name     string `json:"username"`
		Password string `json:"password"`
		Lang     string `json:"language"`

		Email       *string `json:"email"`
		DisplayName *string `json:"display_name"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Password == "" {
		args.Password = kilonova.RandomString(7)
	}

	user, err := s.base.GenerateUser(r.Context(), args.Name, args.Password, args.Lang, kilonova.PreferredThemeDark, args.DisplayName, args.Email)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Password string             `json:"password"`
		User     *kilonova.UserFull `json:"user"`
	}{
		Password: args.Password,
		User:     user,
	})
}
