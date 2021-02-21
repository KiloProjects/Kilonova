package kilonova

import (
	"context"
	"database/sql"
	"errors"
	"log"
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

	VerifiedEmail    bool         `json:"verified_email" db:"verified_email"`
	EmailVerifSentAt sql.NullTime `json:"-" db:"email_verif_sent_at"`

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
	Name    *string `json:"name"`
	Email   *string `json:"email"`
	PwdHash *string `json:"pwd_hash"`

	Admin    *bool `json:"admin"`
	Proposer *bool `json:"proposer"`

	Bio            *string `json:"bio"`
	DefaultVisible *bool   `json:"default_visible"`

	VerifiedEmail    *bool      `json:"verified_email"`
	EmailVerifSentAt *time.Time `json:"-"`

	Banned   *bool `json:"banned"`
	Disabled *bool `json:"disabled"`
}

// UserService represents a service for managing users
type UserService interface {
	// UserByID looks up a user by ID. Returns ENOTFOUND if the ID doesn't exist
	UserByID(ctx context.Context, id int) (*User, error)

	// Users retrieves users based on a filter.
	Users(ctx context.Context, filter UserFilter) ([]*User, error)

	// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
	CountUsers(ctx context.Context, filter UserFilter) (int, error)

	// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
	UserExists(ctx context.Context, username, email string) (bool, error)

	// CreateUser creates a new user with the info specified in the object.
	// On success, user.ID is set to the new user ID.
	CreateUser(ctx context.Context, user *User) error

	// UpdateUser updates a user.
	// Returns ENOTFOUND if the user does not exist
	UpdateUser(ctx context.Context, id int, upd UserUpdate) error

	// DeleteUser permanently deletes a user from the system.
	DeleteUser(ctx context.Context, id int) error
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
		log.Println(err)
		return false
	}
	return true
}
