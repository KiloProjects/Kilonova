package repository

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	conn *pgxpool.Pool
}

func NewUserRepository(conn *pgxpool.Pool) *UserRepository {
	return &UserRepository{conn: conn}
}

// User looks up a user by ID.
func (s *UserRepository) User(ctx context.Context, filter kilonova.UserFilter) (*kilonova.UserFull, error) {
	filter.Limit = 1
	users, err := s.Users(ctx, filter)
	if err != nil || len(users) == 0 {
		return nil, err
	}
	return users[0], nil
}

func (s *UserRepository) HashedPassword(ctx context.Context, userID int) (string, error) {
	var password string
	err := s.conn.QueryRow(ctx, "SELECT password FROM users WHERE id = $1", userID).Scan(&password)
	if err != nil {
		return "", err
	}
	return password, nil
}

// Users retrieves users based on a filter.
func (s *UserRepository) Users(ctx context.Context, filter kilonova.UserFilter) ([]*kilonova.UserFull, error) {
	sb := sq.Select("*").From("users")
	sb = userFilterQuery(&filter, sb)
	query, args, err := sb.OrderBy("id ASC").ToSql()
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, query, args...)
	users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[User])
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.UserFull{}, nil
	}

	usersFull := make([]*kilonova.UserFull, len(users))
	for i := range users {
		usersFull[i] = users[i].Full()
	}
	return usersFull, err
}

// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
func (s *UserRepository) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	sb := sq.Select("COUNT(*)").From("users")
	sb = userFilterQuery(&filter, sb).RemoveLimit().RemoveOffset()
	query, args, err := sb.ToSql()
	if err != nil {
		return -1, err
	}

	var count int
	err = s.conn.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
func (s *UserRepository) UserExists(ctx context.Context, username string, email string) (bool, error) {
	count, err := s.CountUsers(ctx, kilonova.UserFilter{Name: &username})
	if err == nil && count > 0 {
		return true, nil
	}
	count, err = s.CountUsers(ctx, kilonova.UserFilter{Email: &email})
	return count > 0, err
}

func (s *UserRepository) LastUsernameChange(ctx context.Context, userID int) (time.Time, error) {
	var changedAt time.Time
	err := s.conn.QueryRow(ctx, "SELECT MAX(changed_at) FROM username_change_history WHERE user_id = $1", userID).Scan(&changedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Now(), nil
		}
		return time.Now(), err
	}
	return changedAt, nil
}

func (s *UserRepository) NameUsedBefore(ctx context.Context, name string) (bool, error) {
	var cnt int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(name) FROM username_change_history WHERE lower(name) = lower($1)", name).Scan(&cnt)
	if err != nil {
		return true, err
	}
	return cnt > 0, nil
}

func (s *UserRepository) UsernameChangeHistory(ctx context.Context, userID int) ([]*kilonova.UsernameChange, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM username_change_history WHERE user_id = $1 ORDER BY changed_at DESC", userID)
	changes, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[kilonova.UsernameChange])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*kilonova.UsernameChange{}, nil
		}
		return nil, err
	}
	return changes, nil
}

func (s *UserRepository) HistoricalUsernameHolders(ctx context.Context, name string) ([]int, error) {
	rows, _ := s.conn.Query(ctx, "SELECT user_id, MAX(changed_at) FROM username_change_history WHERE lower(name) = lower($1) GROUP BY user_id ORDER BY MAX(changed_at) DESC", name)
	entries, err := pgx.CollectRows(rows, pgx.RowToStructByPos[struct {
		UserID    int
		ChangedAt time.Time
	}])
	if errors.Is(err, pgx.ErrNoRows) {
		return []int{}, nil
	}
	ids := make([]int, len(entries))
	for i := range ids {
		ids[i] = entries[i].UserID
	}
	return ids, err
}

// UpdateUser updates a user.
// Returns ENOTFOUND if the user does not exist
func (s *UserRepository) UpdateUser(ctx context.Context, id int, upd kilonova.UserFullUpdate) error {
	updQuery := sq.Update("users").Where(sq.Eq{"id": id})

	if v := upd.Name; v != nil {
		updQuery = updQuery.Set("name", v)
	}
	if v := upd.Email; v != nil {
		updQuery = updQuery.Set("email", v)
	}

	if v := upd.DisplayName; v != nil {
		updQuery = updQuery.Set("display_name", strings.TrimSpace(*v))
	}

	if v := upd.Admin; v != nil {
		updQuery = updQuery.Set("admin", v)
	}
	if v := upd.Proposer; v != nil {
		updQuery = updQuery.Set("proposer", v)
	}
	if v := upd.Bio; v != nil {
		updQuery = updQuery.Set("bio", strings.TrimSpace(*v))
	}
	if v := upd.VerifiedEmail; v != nil {
		updQuery = updQuery.Set("verified_email", v)
	}
	if v := upd.EmailVerifSentAt; v != nil {
		updQuery = updQuery.Set("email_verif_sent_at", v)
	}

	if v := upd.DiscordID; upd.SetDiscordID {
		updQuery = updQuery.Set("discord_id", v)
	}
	if v := upd.AvatarType; v != nil {
		updQuery = updQuery.Set("avatar_type", v)
	}

	if v := upd.LockedLogin; v != nil {
		updQuery = updQuery.Set("locked_login", v)
	}
	if v := upd.NameChangeRequired; v != nil {
		updQuery = updQuery.Set("name_change_required", v)
	}

	if v := upd.PreferredLanguage; v != "" {
		updQuery = updQuery.Set("preferred_language", v)
	}
	if v := upd.PreferredTheme; v != kilonova.PreferredThemeNone {
		updQuery = updQuery.Set("preferred_theme", v)
	}

	query, args, err := updQuery.ToSql()
	if err != nil {
		return err
	}

	_, err = s.conn.Exec(ctx, query, args...)
	return err
}

func (s *UserRepository) UpdateUserPasswordHash(ctx context.Context, userID int, hash string) error {
	_, err := s.conn.Exec(ctx, "UPDATE users SET password = $1 WHERE id = $2", hash, userID)
	return err
}

// DeleteUser permanently deletes a user from the system.
func (s *UserRepository) DeleteUser(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

// CreateUser creates a new user with the specified data.
func (s *UserRepository) CreateUser(ctx context.Context, name, passwordHash, email, preferredLanguage string, theme kilonova.PreferredTheme, displayName string, bio string, generated bool) (int, error) {
	if name == "" || passwordHash == "" || email == "" || preferredLanguage == "" {
		return -1, kilonova.ErrMissingRequired
	}

	var id = -1
	err := s.conn.QueryRow(ctx,
		"INSERT INTO users (name, email, password, preferred_language, preferred_theme, display_name, generated, verified_email, bio) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		name, email, passwordHash, preferredLanguage, theme, displayName, generated, generated, bio, // generated is for both generated and verified_email!
	).Scan(&id)
	return id, err
}

func (s *UserRepository) LogSignup(ctx context.Context, userID int, ip *netip.Addr, userAgent *string) error {
	_, err := s.conn.Exec(ctx, `INSERT INTO signup_logs (user_id, ip_addr, user_agent) VALUES ($1, $2, $3)`, userID, ip, userAgent)
	return err
}

func (s *UserRepository) CountSignups(ctx context.Context, ip netip.Addr, since time.Time) (int, error) {
	var cnt int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM signup_logs WHERE ip_addr = $1 AND created_at >= $2", ip, since).Scan(&cnt)
	if err != nil {
		return -1, err
	}
	return cnt, nil
}

func userFilterQuery(filter *kilonova.UserFilter, sb sq.SelectBuilder) sq.SelectBuilder {
	where := sq.And{}
	if v := filter.ID; v != nil {
		where = append(where, sq.Eq{"id": v})
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		where = append(where, sq.Expr("0 = 1"))
	}
	if v := filter.IDs; len(v) > 0 {
		where = append(where, sq.Expr("id = ANY(?)", v))
	}
	if v := filter.Name; v != nil {
		where = append(where, sq.Expr("lower(name) = lower(?)", v))
	}
	if v := filter.FuzzyName; v != nil {
		where = append(where, sq.Expr("position(lower(unaccent(?)) in format('#%s %s %s', id, lower(unaccent(name)), lower(unaccent(COALESCE(display_name, ''))))) > 0", v))
	}
	if v := filter.Email; v != nil {
		where = append(where, sq.Expr("lower(email) = lower(?)", v))
	}
	if v := filter.Admin; v != nil {
		where = append(where, sq.Eq{"admin": v})
	}
	if v := filter.Proposer; v != nil {
		where = append(where, sq.Eq{"proposer": v})
	}

	if v := filter.ContestID; v != nil {
		where = append(where, sq.Expr("EXISTS (SELECT 1 FROM contest_registrations WHERE user_id = users.id AND contest_id = ?)", v))
	}

	if v := filter.Generated; v != nil {
		where = append(where, sq.Eq{"generated": v})
	}

	if v := filter.SessionID; v != nil {
		where = append(where, sq.Expr("EXISTS (SELECT 1 FROM active_sessions WHERE user_id = users.id AND id = ?)", v))
	}

	if v := filter.Limit; v > 0 {
		sb = sb.Limit(v)
	}
	if v := filter.Offset; v > 0 {
		sb = sb.Offset(v)
	}

	return sb.Where(where)
}

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

	NameChangeRequired bool `json:"name_change_required" db:"name_change_required"`

	DiscordID  *string `json:"discord_id" db:"discord_id"`
	AvatarType string  `json:"avatar_type" db:"avatar_type"`

	LockedLogin bool `json:"locked_login" db:"locked_login"`
	Generated   bool `json:"generated" db:"generated"`

	DisplayName string `json:"display_name" db:"display_name"`
}

func (user *User) Brief() *kilonova.UserBrief {
	if user == nil {
		return nil
	}
	return &kilonova.UserBrief{
		ID:       user.ID,
		Name:     user.Name,
		Admin:    user.Admin,
		Proposer: user.Proposer,

		DisplayName: user.DisplayName,

		Generated: user.Generated,

		DiscordID: user.DiscordID,
	}
}

func (user *User) Full() *kilonova.UserFull {
	if user == nil {
		return nil
	}
	t := time.Unix(0, 0)
	if user.EmailVerifSentAt != nil {
		t = *user.EmailVerifSentAt
	}
	return &kilonova.UserFull{
		UserBrief:         *user.Brief(),
		Bio:               user.Bio,
		Email:             user.Email,
		VerifiedEmail:     user.VerifiedEmail,
		PreferredLanguage: user.PreferredLanguage,
		CreatedAt:         user.CreatedAt,
		EmailVerifResent:  t,
		LockedLogin:       user.LockedLogin,
		NameChangeForced:  user.NameChangeRequired,

		DiscordID:  user.DiscordID,
		AvatarType: user.AvatarType,
	}
}
