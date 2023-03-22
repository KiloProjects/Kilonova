package sudoapi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *BaseAPI) UserBrief(ctx context.Context, id int) (*UserBrief, *StatusError) {
	user, err := s.db.User(ctx, id)
	if err != nil || user == nil {
		if errors.Is(err, context.Canceled) {
			return nil, WrapError(err, "Context canceled")
		}
		if err != nil {
			zap.S().Warn(err)
		}
		return nil, WrapError(ErrNotFound, "User not found")
	}
	return user.ToBrief(), nil
}

func (s *BaseAPI) UserFull(ctx context.Context, id int) (*UserFull, *StatusError) {
	user, err := s.db.User(ctx, id)
	if err != nil || user == nil {
		return nil, WrapError(ErrNotFound, "User not found")
	}
	return user.ToFull(), nil
}

func (s *BaseAPI) UserBriefByName(ctx context.Context, name string) (*UserBrief, *StatusError) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, Statusf(400, "Username not specified")
	}
	user, err := s.db.UserByName(ctx, name)
	if err != nil || user == nil {
		return nil, WrapError(ErrNotFound, "User not found")
	}
	return user.ToBrief(), nil
}

func (s *BaseAPI) UserFullByName(ctx context.Context, name string) (*UserFull, *StatusError) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, Statusf(400, "Username not specified")
	}
	user, err := s.db.UserByName(ctx, name)
	if err != nil || user == nil {
		return nil, WrapError(ErrNotFound, "User not found")
	}
	return user.ToFull(), nil
}

func (s *BaseAPI) UserFullByEmail(ctx context.Context, email string) (*UserFull, *StatusError) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, Statusf(400, "Email not specified")
	}
	user, err := s.db.UserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, WrapError(ErrNotFound, "User not found")
	}
	return user.ToFull(), nil
}

func (s *BaseAPI) UsersBrief(ctx context.Context, filter kilonova.UserFilter) ([]*UserBrief, *StatusError) {
	users, err := s.db.Users(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	var usersBrief []*UserBrief
	for _, user := range users {
		usersBrief = append(usersBrief, user.ToBrief())
	}
	if len(usersBrief) == 0 {
		return []*UserBrief{}, nil
	}
	return usersBrief, nil
}

func (s *BaseAPI) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, *StatusError) {
	cnt, err := s.db.CountUsers(ctx, filter)
	if err != nil {
		return -1, WrapError(err, "Couldn't get user count")
	}
	return cnt, nil
}

func (s *BaseAPI) UpdateUser(ctx context.Context, userID int, upd kilonova.UserUpdate) *StatusError {
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{UserUpdate: upd})
}

func (s *BaseAPI) updateUser(ctx context.Context, userID int, upd kilonova.UserFullUpdate) *StatusError {
	if err := s.db.UpdateUser(ctx, userID, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update user")
	}
	return nil
}

func (s *BaseAPI) VerifyUserPassword(ctx context.Context, uid int, password string) *StatusError {
	user, err := s.db.User(ctx, uid)
	if err != nil || user == nil {
		return WrapError(ErrNotFound, "User not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return Statusf(400, "Invalid password")
	} else if err != nil {
		return ErrUnknownError
	}

	return nil
}

func (s *BaseAPI) DeleteUser(ctx context.Context, uid int) *StatusError {
	if err := s.db.DeleteUser(ctx, uid); err != nil {
		return WrapError(err, "Couldn't delete user")
	}
	s.LogUserAction(ctx, "Deleted user %d", uid)
	return nil
}

func (s *BaseAPI) UpdateUserPassword(ctx context.Context, uid int, password string) *StatusError {
	if err := s.CheckValidPassword(password); err != nil {
		return err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return WrapError(err, "Couldn't generate hash")
	}

	if err := s.db.UpdateUserPasswordHash(ctx, uid, hash); err != nil {
		return WrapError(err, "Couldn't update password")
	}
	return nil
}

func (s *BaseAPI) GenerateUser(ctx context.Context, uname, pwd, lang string, theme kilonova.PreferredTheme) (*kilonova.UserFull, *StatusError) {
	uname = strings.TrimSpace(uname)
	if !(len(uname) >= 3 && len(uname) <= 32 && usernameRegex.MatchString(uname)) {
		return nil, Statusf(400, "Username must be between 3 and 32 characters long and must contain only letters, digits, underlines and dashes.")
	}
	if err := s.CheckValidPassword(pwd); err != nil {
		return nil, err
	}
	if !(lang == "" || lang == "en" || lang == "ro") {
		return nil, Statusf(400, "Invalid language.")
	}
	if !(theme == kilonova.PreferredThemeNone || theme == kilonova.PreferredThemeLight || theme == kilonova.PreferredThemeDark) {
		return nil, Statusf(400, "Invalid theme.")
	}

	if exists, err := s.db.UserExists(ctx, uname, "INVALID_EMAIL"); err != nil || exists {
		return nil, Statusf(400, "User matching username already exists!")
	}

	if lang == "" {
		lang = config.Common.DefaultLang
	}
	if theme == kilonova.PreferredThemeNone {
		theme = kilonova.PreferredThemeDark
	}

	// Dummy email
	email := fmt.Sprintf("email_%s@kilonova.ro", uname)

	id, err := s.createUser(ctx, uname, email, pwd, lang, theme, true)
	if err != nil {
		zap.S().Warn(err)
		return nil, Statusf(500, "Couldn't create user")
	}

	user, err1 := s.UserFull(ctx, id)
	if err1 != nil {
		zap.S().Warn(err1)
	}

	return user, err1
}

func (s *BaseAPI) GetGravatarLink(user *kilonova.UserFull, size int) string {
	v := url.Values{}
	v.Add("s", strconv.Itoa(size))
	v.Add("d", "identicon")

	bSum := md5.Sum([]byte(user.Email))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%s?%s", hex.EncodeToString(bSum[:]), v.Encode())
}

func (s *BaseAPI) createUser(ctx context.Context, username, email, password, lang string, theme kilonova.PreferredTheme, generated bool) (int, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return -1, err
	}

	id, err := s.db.CreateUser(ctx, username, hash, email, lang, theme, generated)
	if err != nil {
		zap.S().Warn(err)
		return -1, err
	}

	if id == 1 {
		var True = true
		if err := s.db.UpdateUser(ctx, id, kilonova.UserFullUpdate{Admin: &True, Proposer: &True}); err != nil {
			zap.S().Warn(err)
			return id, err
		}
	}

	return id, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}
