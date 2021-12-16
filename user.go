package kilonova

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int       `json:"id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	Name           string    `json:"name"`
	Admin          bool      `json:"admin"`
	Proposer       bool      `json:"proposer"`
	Email          string    `json:"email"`
	Password       string    `json:"-"`
	Bio            string    `json:"bio"`
	DefaultVisible bool      `json:"default_visible" db:"default_visible"`

	VerifiedEmail     bool         `json:"verified_email" db:"verified_email"`
	EmailVerifSentAt  sql.NullTime `json:"-" db:"email_verif_sent_at"`
	PreferredLanguage string       `json:"-" db:"preferred_language"`

	Banned   bool `json:"banned,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
}

// UserFilter is the struct with all filterable fields on the user
// It also provides a Limit and Offset field, for pagination
type UserFilter struct {
	ID *int `json:"id"`

	// Name is case insensitive
	Name  *string `json:"name"`
	Email *string `json:"email"`

	Admin    *bool `json:"admin"`
	Proposer *bool `json:"proposer"`

	Verified *bool `json:"verified"`
	Banned   *bool `json:"banned"`
	Disabled *bool `json:"disabled"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// UserUpdate is the struct with all updatable fields on the user
type UserUpdate struct {
	//Name    *string `json:"name"`
	Email   *string `json:"email"`
	PwdHash *string `json:"pwd_hash"`

	Admin    *bool `json:"admin"`
	Proposer *bool `json:"proposer"`

	Bio            *string `json:"bio"`
	DefaultVisible *bool   `json:"default_visible"`

	VerifiedEmail     *bool      `json:"verified_email"`
	EmailVerifSentAt  *time.Time `json:"-"`
	PreferredLanguage string     `json:"-"`

	Banned   *bool `json:"banned"`
	Disabled *bool `json:"disabled"`
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}
