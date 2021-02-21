package psql

import (
	"context"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.UserService = &UserService{}

type UserService struct {
	db *sqlx.DB
}

// UserByID looks up a user by ID. Returns ENOTFOUND if the ID doesn't exist
func (s *UserService) UserByID(ctx context.Context, id int) (*kilonova.User, error) {
	// TODO: Wrap error in a kilonova error
	var user kilonova.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1 LIMIT 1", id)
	return &user, err
}

// Users retrieves users based on a filter.
func (s *UserService) Users(ctx context.Context, filter kilonova.UserFilter) ([]*kilonova.User, error) {
	// TODO: Wrap error in a kilonova error
	var users []*kilonova.User
	where, args := s.filterQueryMaker(&filter)
	query := s.db.Rebind("SELECT * from users WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := s.db.SelectContext(ctx, &users, query, args...)
	return users, err
}

// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
func (s *UserService) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	// TODO: Wrap error in a kilonova error
	where, args := s.filterQueryMaker(&filter)
	var count int
	query := s.db.Rebind("SELECT COUNT(*) FROM users WHERE " + strings.Join(where, " AND "))
	err := s.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
func (s *UserService) UserExists(ctx context.Context, username string, email string) (bool, error) {
	// TODO: Wrap error in a kilonova error
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users WHERE lower(name) = lower($1) OR lower(email) = lower($2)", username, email)
	return count > 0, err
}

// CreateUser creates a new user with the info specified in the object.
// On success, user.ID is set to the new user ID.
func (s *UserService) CreateUser(ctx context.Context, user *kilonova.User) error {
	// TODO: Wrap error in a kilonova error
	return s.createUser(ctx, user)
}

// UpdateUser updates a user.
// Returns ENOTFOUND if the user does not exist
func (s *UserService) UpdateUser(ctx context.Context, id int, upd kilonova.UserUpdate) error {
	// TODO: Wrap error in a kilonova error
	return s.updateUser(ctx, id, &upd)
}

// DeleteUser permanently deletes a user from the system.
func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	// TODO: Wrap error in a kilonova error
	_, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

func (s *UserService) updateUser(ctx context.Context, id int, upd *kilonova.UserUpdate) error {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}
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

	query := s.db.Rebind("UPDATE users SET " + strings.Join(toUpd, ", ") + " WHERE id = ?;")
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *UserService) createUser(ctx context.Context, user *kilonova.User) error {
	if user.Name == "" || user.Password == "" || user.Email == "" {
		return kilonova.ErrMissingRequired
	}

	var id int
	err := s.db.GetContext(ctx, &id, "INSERT INTO users (name, email, password, bio, default_visible, verified_email, admin, proposer) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id", user.Name, user.Email, user.Password, user.Bio, user.DefaultVisible, user.VerifiedEmail, user.Admin, user.Proposer)
	if err == nil {
		user.ID = id
	}
	return err
}

func (s *UserService) filterQueryMaker(filter *kilonova.UserFilter) ([]string, []interface{}) {
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
