package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/microcosm-cc/bluemonday"
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
}

func (user *User) ToBrief() *kilonova.UserBrief {
	if user == nil {
		return nil
	}
	return &kilonova.UserBrief{
		ID:       user.ID,
		Name:     user.Name,
		Admin:    user.Admin,
		Proposer: user.Proposer,
		Bio:      user.Bio,
	}
}

func (user *User) ToFull() *kilonova.UserFull {
	if user == nil {
		return nil
	}
	t := time.Unix(0, 0)
	if user.EmailVerifSentAt.Valid {
		t = user.EmailVerifSentAt.Time
	}
	return &kilonova.UserFull{
		UserBrief:         *user.ToBrief(),
		Email:             user.Email,
		VerifiedEmail:     user.VerifiedEmail,
		PreferredLanguage: user.PreferredLanguage,
		CreatedAt:         user.CreatedAt,
		EmailVerifResent:  t,
	}
}

// User looks up a user by ID.
func (s *DB) User(ctx context.Context, id int) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// User looks up a user by name.
func (s *DB) UserByName(ctx context.Context, name string) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT * FROM users WHERE lower(name) = lower($1) LIMIT 1", name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// User looks up a user by email.
func (s *DB) UserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT * FROM users WHERE lower(email) = lower($1) LIMIT 1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// Users retrieves users based on a filter.
func (s *DB) Users(ctx context.Context, filter kilonova.UserFilter) ([]*User, error) {
	var users []*User
	where, args := userFilterQuery(&filter)
	query := s.conn.Rebind("SELECT * from users WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.conn.SelectContext(ctx, &users, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*User{}, nil
	}
	return users, err
}

// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
func (s *DB) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	var count int
	where, args := userFilterQuery(&filter)
	query := s.conn.Rebind("SELECT COUNT(*) FROM users WHERE " + strings.Join(where, " AND "))
	err := s.conn.GetContext(ctx, &count, query, args...)
	return count, err
}

// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
func (s *DB) UserExists(ctx context.Context, username string, email string) (bool, error) {
	var count int
	err := s.conn.GetContext(ctx, &count, s.conn.Rebind("SELECT COUNT(*) FROM users WHERE lower(name) = lower(?) OR lower(email) = lower(?)"), username, email)
	return count > 0, err
}

// UpdateUser updates a user.
// Returns ENOTFOUND if the user does not exist
func (s *DB) UpdateUser(ctx context.Context, id int, upd kilonova.UserFullUpdate) error {
	toUpd, args := []string{}, []any{}

	/*if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}*/
	if v := upd.Email; v != nil {
		toUpd, args = append(toUpd, "email = ?"), append(args, v)
	}

	if v := upd.Admin; v != nil {
		toUpd, args = append(toUpd, "admin = ?"), append(args, v)
	}
	if v := upd.Proposer; v != nil {
		toUpd, args = append(toUpd, "proposer = ?"), append(args, v)
	}
	if v := upd.Bio; v != nil {
		toUpd, args = append(toUpd, "bio = ?"), append(args, strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(*v)))
	}
	if v := upd.VerifiedEmail; v != nil {
		toUpd, args = append(toUpd, "verified_email = ?"), append(args, v)
	}
	if v := upd.EmailVerifSentAt; v != nil {
		toUpd, args = append(toUpd, "email_verif_sent_at = ?"), append(args, v)
	}
	if v := upd.PreferredLanguage; v != "" {
		toUpd, args = append(toUpd, "preferred_language = ?"), append(args, v)
	}
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	query := s.conn.Rebind("UPDATE users SET " + strings.Join(toUpd, ", ") + " WHERE id = ?;")
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) UpdateUserPasswordHash(ctx context.Context, userID int, hash string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("UPDATE users SET password = ? WHERE id = ?"), hash, userID)
	return err
}

// DeleteUser permanently deletes a user from the system.
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM users WHERE id = ?"), id)
	return err
}

// CreateUser creates a new user with the specified data.
func (s *DB) CreateUser(ctx context.Context, name, passwordHash, email, preferredLanguage string) (int, error) {
	if name == "" || passwordHash == "" || email == "" || preferredLanguage == "" {
		return -1, kilonova.ErrMissingRequired
	}

	var id = -1
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("INSERT INTO users (name, email, password, preferred_language) VALUES (?, ?, ?, ?) RETURNING id"), name, email, passwordHash, preferredLanguage)
	return id, err
}

func userFilterQuery(filter *kilonova.UserFilter) ([]string, []any) {
	where, args := []string{"1 = 1"}, []any{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		where = append(where, "id = -1")
	}
	if v := filter.IDs; len(v) > 0 {
		where = append(where, "id IN (?"+strings.Repeat(",?", len(v)-1)+")")
		for _, el := range v {
			args = append(args, el)
		}
	}
	if v := filter.Name; v != nil {
		where, args = append(where, "lower(name) = lower(?)"), append(args, v)
	}
	if v := filter.Email; v != nil {
		where, args = append(where, "lower(email) = lower(?)"), append(args, v)
	}
	if v := filter.Admin; v != nil {
		where, args = append(where, "admin = ?"), append(args, v)
	}
	if v := filter.Proposer; v != nil {
		where, args = append(where, "proposer = ?"), append(args, v)
	}
	return where, args
}
