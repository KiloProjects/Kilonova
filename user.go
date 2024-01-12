package kilonova

import (
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type PreferredTheme string

const (
	PreferredThemeNone  = ""
	PreferredThemeLight = "light"
	PreferredThemeDark  = "dark"
)

type UserBrief struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Admin    bool   `json:"admin"`
	Proposer bool   `json:"proposer"`

	DisplayName string `json:"display_name"`
}

func (u *UserBrief) IsAuthed() bool {
	return u != nil && u.ID != 0
}

func (u *UserBrief) IsAdmin() bool {
	if !u.IsAuthed() {
		return false
	}
	return u.Admin
}

func (u *UserBrief) IsProposer() bool {
	if !u.IsAuthed() {
		return false
	}
	return u.Admin || u.Proposer
}

type UserFull struct {
	UserBrief
	Bio               string         `json:"bio,omitempty"`
	Email             string         `json:"email,omitempty"`
	VerifiedEmail     bool           `json:"verified_email"`
	PreferredLanguage string         `json:"preferred_language"`
	PreferredTheme    PreferredTheme `json:"preferred_theme"`
	EmailVerifResent  time.Time      `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	Generated         bool           `json:"generated"`
	LockedLogin       bool           `json:"locked_login"`
	NameChangeForced  bool           `json:"name_change_forced"`
}

func (uf *UserFull) Brief() *UserBrief {
	if uf == nil {
		return nil
	}
	return &uf.UserBrief
}

// UserFilter is the struct with all filterable fields on the user
// It also provides a Limit and Offset field, for pagination
type UserFilter struct {
	ID  *int  `json:"id"`
	IDs []int `json:"ids"`

	// Name is case insensitive
	Name  *string `json:"name"`
	Email *string `json:"email"`

	// For user filtering
	FuzzyName *string `json:"name_fuzzy"`

	Admin    *bool `json:"admin"`
	Proposer *bool `json:"proposer"`

	// For registrations
	ContestID *int `json:"contest_id"`

	// For session recognition
	SessionID *string `json:"session_id"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// UserUpdate is the struct with updatable fields that can be easily changed.
// Stuff like admin and proposer should be updated through their dedicated
// SudoAPI methods
type UserUpdate struct {
	//Name    *string `json:"name"`
	DisplayName *string `json:"display_name"`

	Bio *string `json:"bio"`

	PreferredLanguage string         `json:"-"`
	PreferredTheme    PreferredTheme `json:"-"`
}

// UserFullUpdate is the struct with all updatable fields on the user. Internal use only
type UserFullUpdate struct {
	UserUpdate

	Name *string `json:"name"`

	Email *string `json:"email"`

	Admin    *bool `json:"admin"`
	Proposer *bool `json:"proposer"`

	LockedLogin        *bool `json:"locked_login"`
	NameChangeRequired *bool `json:"name_change_required"`

	VerifiedEmail    *bool      `json:"verified_email"`
	EmailVerifSentAt *time.Time `json:"-"`
}

type UsernameChange struct {
	UserID    int       `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	ChangedAt time.Time `json:"changed_at" db:"changed_at"`
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}

func CheckPwdHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false
	}
	if err != nil {
		zap.S().Warn(err)
		return false
	}
	return true
}
