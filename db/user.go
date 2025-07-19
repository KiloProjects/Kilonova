package db

import (
	"context"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova"
)

// User looks up a user by ID.
// Deprecated: Use UserRepository instead
func (s *DB) User(ctx context.Context, filter kilonova.UserFilter) (*kilonova.UserFull, error) {
	return s.userRepo.User(ctx, filter)
}

func (s *DB) HashedPassword(ctx context.Context, userID int) (string, error) {
	return s.userRepo.HashedPassword(ctx, userID)
}

// Users retrieves users based on a filter.
// Deprecated: Use UserRepository instead
func (s *DB) Users(ctx context.Context, filter kilonova.UserFilter) ([]*kilonova.UserFull, error) {
	return s.userRepo.Users(ctx, filter)
}

// CountUsers retrieves the number of users matching a filter. It ignores the limit fields in `filter`.
// Deprecated: Use UserRepository instead
func (s *DB) CountUsers(ctx context.Context, filter kilonova.UserFilter) (int, error) {
	return s.userRepo.CountUsers(ctx, filter)
}

// UserExists says wether or not a user matches either a specific username (case-insensitive), either a specific email address.
// Deprecated: Use UserRepository instead
func (s *DB) UserExists(ctx context.Context, username string, email string) (bool, error) {
	return s.userRepo.UserExists(ctx, username, email)
}

// Deprecated: Use UserRepository instead
func (s *DB) LastUsernameChange(ctx context.Context, userID int) (time.Time, error) {
	return s.userRepo.LastUsernameChange(ctx, userID)
}

func (s *DB) NameUsedBefore(ctx context.Context, name string) (bool, error) {
	return s.userRepo.NameUsedBefore(ctx, name)
}

// Deprecated: Use UserRepository instead
func (s *DB) UsernameChangeHistory(ctx context.Context, userID int) ([]*kilonova.UsernameChange, error) {
	return s.userRepo.UsernameChangeHistory(ctx, userID)
}

// Deprecated: Use UserRepository instead
func (s *DB) HistoricalUsernameHolders(ctx context.Context, name string) ([]int, error) {
	return s.userRepo.HistoricalUsernameHolders(ctx, name)
}

// UpdateUser updates a user.
// Deprecated: Use UserRepository instead
func (s *DB) UpdateUser(ctx context.Context, id int, upd kilonova.UserFullUpdate) error {
	return s.userRepo.UpdateUser(ctx, id, upd)
}

// Deprecated: Use UserRepository instead
func (s *DB) UpdateUserPasswordHash(ctx context.Context, userID int, hash string) error {
	return s.userRepo.UpdateUserPasswordHash(ctx, userID, hash)
}

// DeleteUser permanently deletes a user from the system.
// Deprecated: Use UserRepository instead
func (s *DB) DeleteUser(ctx context.Context, id int) error {
	return s.userRepo.DeleteUser(ctx, id)
}

// CreateUser creates a new user with the specified data.
// Deprecated: Use UserRepository instead
func (s *DB) CreateUser(ctx context.Context, name, passwordHash, email, preferredLanguage string, theme kilonova.PreferredTheme, displayName string, bio string, generated bool) (int, error) {
	return s.userRepo.CreateUser(ctx, name, passwordHash, email, preferredLanguage, theme, displayName, bio, generated)
}

// Deprecated: Use UserRepository instead
func (s *DB) LogSignup(ctx context.Context, userID int, ip *netip.Addr, userAgent *string) error {
	return s.userRepo.LogSignup(ctx, userID, ip, userAgent)
}

// Deprecated: Use UserRepository instead
func (s *DB) CountSignups(ctx context.Context, ip netip.Addr, since time.Time) (int, error) {
	return s.userRepo.CountSignups(ctx, ip, since)
}
