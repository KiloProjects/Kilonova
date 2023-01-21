package db

import (
	"context"
	"database/sql"
	"errors"
)

func (s *DB) ContestRegistrations(ctx context.Context, contestID int, limit, offset int) ([]*User, error) {
	var users []*User
	query := "SELECT * FROM users, contest_registrations rgs WHERE rgs.user_id = users.id AND rgs.contest_id = $1 ORDER BY id ASC" + FormatLimitOffset(limit, offset)
	err := s.conn.SelectContext(ctx, &users, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*User{}, nil
	}
	return users, err
}
