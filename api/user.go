package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/microcosm-cc/bluemonday"
	"go.uber.org/zap"
)

var (
	bm = bluemonday.StrictPolicy()
)

func (s *API) serveGravatar(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull, size int) {
	// Read from cache
	rd, lastmod, valid, err := s.base.GetGravatar(user.Email, size, time.Now().Add(-12*time.Hour))
	if !valid || err != nil {
		zap.S().Warn("BaseAPI GetGravatar is not valid or returned error", valid, err)
		http.Error(w, "", 500)
		return
	}
	defer rd.Close()
	w.Header().Add("ETag", fmt.Sprintf("\"kn-%s-%d\"", user.Name, lastmod.Unix()))
	// Cache for 1 day
	w.Header().Add("Cache-Control", "public, max-age=86400, immutable")

	http.ServeContent(w, r, "gravatar.png", lastmod, rd)
}

func (s *API) getGravatar(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.Atoi(r.FormValue("s"))
	if err != nil || size == 0 {
		size = 128
	}
	s.serveGravatar(w, r, util.ContentUserFull(r), size)
}

func (s *API) serveDiscordAvatar(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull, size int) {
	// Read from cache
	rd, lastmod, valid, err := s.base.GetDiscordAvatar(r.Context(), user, size, time.Now().Add(-12*time.Hour))
	if !valid || err != nil {
		// zap.S().Warn("BaseAPI GetDiscordAvatar is not valid or returned error", valid, err)
		http.Error(w, "", 500)
		return
	}
	defer rd.Close()
	w.Header().Add("ETag", fmt.Sprintf("\"kn-discord-%s-%d\"", user.Name, lastmod.Unix()))
	// Cache for 1 day
	w.Header().Add("Cache-Control", "public, max-age=86400, immutable")

	http.ServeContent(w, r, "discordAvatar.png", lastmod, rd)
}

func (s *API) getDiscordAvatar(w http.ResponseWriter, r *http.Request) {
	size, err := strconv.Atoi(r.FormValue("s"))
	if err != nil || size == 0 {
		size = 128
	}
	s.serveDiscordAvatar(w, r, util.ContentUserFull(r), size)
}

func (s *API) getAvatar(w http.ResponseWriter, r *http.Request) {
	user := util.ContentUserFull(r)
	if user.AvatarType == "discord" && user.DiscordID != nil {
		s.getDiscordAvatar(w, r)
		return
	}
	s.getGravatar(w, r)
}

func (s *API) deauthAllSessions(w http.ResponseWriter, r *http.Request) {
	if err := s.base.RemoveUserSessions(r.Context(), util.ContentUserBrief(r).ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Force logged out")
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
			util.ContentUserBrief(r).ID,
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
			util.ContentUserBrief(r).ID,
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

		if len(safe) > 10000 { // 10k characters
			errorData(w, "Bio is too long", 400)
			return
		}

		if err := s.base.UpdateUser(
			r.Context(),
			util.ContentUserBrief(r).ID,
			kilonova.UserUpdate{Bio: &safe},
		); err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, "Updated bio")
	}
}

func (s *API) setAvatarType() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args struct{ AvatarType string }
		if err := decoder.Decode(&args, r.Form); err != nil {
			errorData(w, err, 400)
			return
		}

		safe := strings.TrimSpace(args.AvatarType)
		if safe != "discord" {
			safe = "gravatar"
		}

		if err := s.base.UpdateUser(
			r.Context(),
			util.ContentUserBrief(r).ID,
			kilonova.UserUpdate{AvatarType: &safe},
		); err != nil {
			err.WriteError(w)
			return
		}

		returnData(w, "Updated avatar type. It may take a while for changes to propagate")
	}
}

func (s *API) manageUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Lockout *bool   `json:"lockout"`
		NewName *string `json:"new_name"`

		ForceUsernameChange *bool `json:"force_username_change"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	user := util.ContentUserFull(r)
	if args.NewName != nil && len(*args.NewName) > 2 && user.Name != *args.NewName {
		// Admins can change to formerly existing names
		if err := s.base.UpdateUsername(r.Context(), user, *args.NewName, false, true); err != nil {
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
	user := util.ContentUserBrief(r)

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

func (s *API) getSolvedProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := s.base.SolvedProblems(r.Context(), util.ContentUserBrief(r), util.UserBrief(r))
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

		Password string `json:"password"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}
	user := util.UserFull(r)
	if args.UserID != nil && util.UserBrief(r).Admin {
		newUser, err := s.base.UserFull(r.Context(), *args.UserID)
		if err != nil {
			err.WriteError(w)
			return
		}
		user = newUser
	}

	fromAdmin := util.UserBrief(r).Admin || util.UserFull(r).NameChangeForced
	if !fromAdmin {
		if err := s.base.VerifyUserPassword(r.Context(), user.ID, args.Password); err != nil {
			err.WriteError(w)
			return
		}
	}

	if err := s.base.UpdateUsername(r.Context(), user, args.NewName, !util.UserBrief(r).Admin, fromAdmin); err != nil {
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
		if err := s.base.SendPasswordResetEmail(context.Background(), user.ID, user.Name, user.Email, user.PreferredLanguage); err != nil {
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

	if err := s.base.SendVerificationEmail(r.Context(), util.UserBrief(r).ID, util.UserBrief(r).Name, email, util.UserFull(r).PreferredLanguage); err != nil {
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

	if err := s.base.SendVerificationEmail(r.Context(), u.ID, u.Name, u.Email, u.PreferredLanguage); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Verification email resent")
}

func (s *API) generateUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args sudoapi.UserGenerationRequest
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	pwd, user, err := s.base.GenerateUserFlow(r.Context(), args)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Password string             `json:"password"`
		User     *kilonova.UserFull `json:"user"`
	}{
		Password: pwd,
		User:     user,
	})
}
