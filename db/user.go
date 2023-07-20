package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
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
		Bio:               user.Bio,
		Email:             user.Email,
		VerifiedEmail:     user.VerifiedEmail,
		PreferredLanguage: user.PreferredLanguage,
		CreatedAt:         user.CreatedAt,
		EmailVerifResent:  t,
		Generated:         user.Generated,
	}
}

// User looks up a user by ID.
func (s *DB) User(ctx context.Context, filter kilonova.UserFilter) (*User, error) {
	filter.Limit = 1
	users, err := s.Users(ctx, filter)
	if err != nil || len(users) == 0 {
		return nil, err
	}
	return users[0], nil
}

// Users retrieves users based on a filter.
func (s *DB) Users(ctx context.Context, filter kilonova.UserFilter) ([]*User, error) {
	fb := newFilterBuilder()
	userFilterQuery(&filter, fb)
	rows, _ := s.pgconn.Query(ctx, "SELECT * from users WHERE "+fb.Where()+" ORDER BY id ASC "+FormatLimitOffset(filter.Limit, filter.Offset), fb.Args()...)
	users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[User])
	if errors.Is(err, sql.ErrNoRows) {
		return []*User{}, nil
	}
	return users, err
}

// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
func (s *DB) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	var count int
	fb := newFilterBuilder()
	userFilterQuery(&filter, fb)
	err := s.pgconn.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE "+fb.Where(), fb.Args()...).Scan(&count)
	return count, err
}

// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
func (s *DB) UserExists(ctx context.Context, username string, email string) (bool, error) {
	count, err := s.CountUsers(ctx, kilonova.UserFilter{Name: &username})
	if err == nil && count > 0 {
		return true, nil
	}
	count, err = s.CountUsers(ctx, kilonova.UserFilter{Email: &email})
	return count > 0, err
}

// UpdateUser updates a user.
// Returns ENOTFOUND if the user does not exist
func (s *DB) UpdateUser(ctx context.Context, id int, upd kilonova.UserFullUpdate) error {
	ub := newUpdateBuilder()

	if v := upd.Email; v != nil {
		ub.AddUpdate("email = %s", v)
	}

	if v := upd.Admin; v != nil {
		ub.AddUpdate("admin = %s", v)
	}
	if v := upd.Proposer; v != nil {
		ub.AddUpdate("proposer = %s", v)
	}
	if v := upd.Bio; v != nil {
		ub.AddUpdate("bio = %s", strings.TrimSpace(*v))
	}
	if v := upd.VerifiedEmail; v != nil {
		ub.AddUpdate("verified_email = %s", v)
	}
	if v := upd.EmailVerifSentAt; v != nil {
		ub.AddUpdate("email_verif_sent_at = %s", v)
	}
	if v := upd.PreferredLanguage; v != "" {
		ub.AddUpdate("preferred_language = %s", v)
	}
	if v := upd.PreferredTheme; v != kilonova.PreferredThemeNone {
		ub.AddUpdate("preferred_theme = %s", v)
	}
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}

	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)

	_, err := s.pgconn.Exec(ctx, "UPDATE users SET "+fb.WithUpdate(), fb.Args()...)
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
	err := s.pgconn.QueryRow(ctx,
		"INSERT INTO users (name, email, password, preferred_language, preferred_theme, generated, verified_email) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		name, email, passwordHash, preferredLanguage, theme, generated, generated, // generated is for both generated and verified_email!
	).Scan(&id)
	return id, err
}

func userFilterQuery(filter *kilonova.UserFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		fb.AddConstraint("id = -1")
	}
	if v := filter.IDs; len(v) > 0 {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.Name; v != nil {
		fb.AddConstraint("lower(name) = lower(%s)", v)
	}
	if v := filter.FuzzyName; v != nil {
		fb.AddConstraint("position(lower(unaccent(%s)) in format('#%%s %%s', id, lower(unaccent(name)))) > 0", v)
	}
	if v := filter.Email; v != nil {
		fb.AddConstraint("lower(email) = lower(%s)", v)
	}
	if v := filter.Admin; v != nil {
		fb.AddConstraint("admin = %s", v)
	}
	if v := filter.Proposer; v != nil {
		fb.AddConstraint("proposer = %s", v)
	}

	if v := filter.ContestID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM contest_registrations WHERE user_id = users.id AND contest_id = %s)", v)
	}

	if v := filter.SessionID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM active_sessions WHERE user_id = users.id AND id = %s)", v)
	}
}
