package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/KiloProjects/kilonova"
)

// User looks up a user by ID. Returns ENOTFOUND if the ID doesn't exist
func (s *DB) User(ctx context.Context, id int) (*kilonova.User, error) {
	var user kilonova.User
	err := s.conn.GetContext(ctx, &user, s.conn.Rebind("SELECT * FROM users WHERE id = ? LIMIT 1"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &user, err
}

// Users retrieves users based on a filter.
func (s *DB) Users(ctx context.Context, filter kilonova.UserFilter) ([]*kilonova.User, error) {
	var users []*kilonova.User
	where, args := userFilterQuery(&filter)
	query := s.conn.Rebind("SELECT * from users WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.conn.SelectContext(ctx, &users, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.User{}, nil
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
func (s *DB) UpdateUser(ctx context.Context, id int, upd kilonova.UserUpdate) error {
	toUpd, args := []string{}, []interface{}{}

	/*if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}*/
	if v := upd.Email; v != nil {
		toUpd, args = append(toUpd, "email = ?"), append(args, v)
	}
	if v := upd.PwdHash; v != nil {
		toUpd, args = append(toUpd, "password = ?"), append(args, v)
	}

	if v := upd.Admin; v != nil {
		toUpd, args = append(toUpd, "admin = ?"), append(args, v)
	}
	if v := upd.Proposer; v != nil {
		toUpd, args = append(toUpd, "proposer = ?"), append(args, v)
	}
	if v := upd.Bio; v != nil {
		toUpd, args = append(toUpd, "bio = ?"), append(args, v)
	}
	if v := upd.DefaultVisible; v != nil {
		toUpd, args = append(toUpd, "default_visible = ?"), append(args, v)
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
	if v := upd.Banned; v != nil {
		toUpd, args = append(toUpd, "banned = ?"), append(args, v)
	}
	if v := upd.Disabled; v != nil {
		toUpd, args = append(toUpd, "disabled = ?"), append(args, v)
	}
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)

	query := s.conn.Rebind("UPDATE users SET " + strings.Join(toUpd, ", ") + " WHERE id = ?;")
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

// DeleteUser permanently deletes a user from the system.
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind("DELETE FROM users WHERE id = ?"), id)
	return err
}

// CreateUser creates a new user with the info specified in the object.
// On success, user.ID is set to the new user ID.
func (s *DB) CreateUser(ctx context.Context, user *kilonova.User) error {
	if user.Name == "" || user.Password == "" || user.Email == "" || user.PreferredLanguage == "" {
		return kilonova.ErrMissingRequired
	}

	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind("INSERT INTO users (name, email, password, bio, default_visible, verified_email, admin, proposer, preferred_language) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id"), user.Name, user.Email, user.Password, user.Bio, user.DefaultVisible, user.VerifiedEmail, user.Admin, user.Proposer, user.PreferredLanguage)
	if err == nil {
		user.ID = id
	}
	return err
}

func userFilterQuery(filter *kilonova.UserFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
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
	if v := filter.Verified; v != nil {
		where, args = append(where, "verified_email = ?"), append(args, v)
	}
	if v := filter.Banned; v != nil {
		where, args = append(where, "banned = ?"), append(args, v)
	}
	if v := filter.Disabled; v != nil {
		where, args = append(where, "disabled = ?"), append(args, v)
	}
	return where, args
}
