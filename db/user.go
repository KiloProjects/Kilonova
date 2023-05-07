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
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Name      string    `json:"name"`
	Admin     bool      `json:"admin"`
	Proposer  bool      `json:"proposer"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Bio       string    `json:"bio"`

	VerifiedEmail    bool       `json:"verified_email" db:"verified_email"`
	EmailVerifSentAt *time.Time `json:"-" db:"email_verif_sent_at"`

	PreferredLanguage string                  `json:"-" db:"preferred_language"`
	PreferredTheme    kilonova.PreferredTheme `json:"-" db:"preferred_theme"`

	Generated bool `json:"generated" db:"generated"`
}

func toUserBrief(user *User) *kilonova.UserBrief {
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

func (user *User) ToBrief() *kilonova.UserBrief {
	return toUserBrief(user)
}

func (user *User) ToFull() *kilonova.UserFull {
	if user == nil {
		return nil
	}
	t := time.Unix(0, 0)
	if user.EmailVerifSentAt != nil {
		t = *user.EmailVerifSentAt
	}
	return &kilonova.UserFull{
		UserBrief:         *user.ToBrief(),
		Email:             user.Email,
		VerifiedEmail:     user.VerifiedEmail,
		PreferredLanguage: user.PreferredLanguage,
		CreatedAt:         user.CreatedAt,
		EmailVerifResent:  t,
		Generated:         user.Generated,
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

// UserByName looks up a user by name.
func (s *DB) UserByName(ctx context.Context, name string) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT * FROM users WHERE lower(name) = lower($1) LIMIT 1", name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// UserByEmail looks up a user by email.
func (s *DB) UserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT * FROM users WHERE lower(email) = lower($1) LIMIT 1", email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// UserBySessionID looks up a user by an active session ID.
func (s *DB) UserBySessionID(ctx context.Context, sessionID string) (*User, error) {
	var user User
	err := s.conn.GetContext(ctx, &user, "SELECT users.* FROM users, active_sessions WHERE users.id = active_sessions.user_id AND active_sessions.id = $1 LIMIT 1", sessionID)
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
	err := s.conn.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE lower(name) = lower($1) OR lower(email) = lower($2)", username, email)
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
	if v := upd.PreferredTheme; v != kilonova.PreferredThemeNone {
		toUpd, args = append(toUpd, "preferred_theme = ?"), append(args, v)
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
	_, err := s.pgconn.Exec(ctx, "UPDATE users SET password = $1 WHERE id = $2", hash, userID)
	return err
}

// DeleteUser permanently deletes a user from the system.
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

// CreateUser creates a new user with the specified data.
func (s *DB) CreateUser(ctx context.Context, name, passwordHash, email, preferredLanguage string, theme kilonova.PreferredTheme, generated bool) (int, error) {
	if name == "" || passwordHash == "" || email == "" || preferredLanguage == "" {
		return -1, kilonova.ErrMissingRequired
	}

	var id = -1
	err := s.conn.GetContext(ctx, &id,
		"INSERT INTO users (name, email, password, preferred_language, preferred_theme, generated, verified_email) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		name, email, passwordHash, preferredLanguage, theme, generated, generated, // generated is for both generated and verified_email!
	)
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
		where, args = append(where, "id = ANY(?)"), append(args, v)
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

	if v := filter.ContestID; v != nil {
		where, args = append(where, "EXISTS (SELECT 1 FROM contest_registrations WHERE user_id = users.id AND contest_id = ?)"), append(args, v)
	}

	return where, args
}
