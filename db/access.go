package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type accessType string

const (
	accessEditor accessType = "editor"
	accessViewer accessType = "viewer"
)

func (s *DB) addAccess(ctx context.Context, tableName, colName string, colValue, userID int, rank accessType) error {
	q := fmt.Sprintf("INSERT INTO %s (%s, user_id, access) VALUES ($1, $2, $3)", tableName, colName)
	_, err := s.conn.Exec(ctx, q, colValue, userID, rank)
	return err
}

func (s *DB) removeAccess(ctx context.Context, tableName, colName string, colValue, userID int) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = $1 AND user_id = $2", tableName, colName)
	_, err := s.conn.Exec(ctx, q, colValue, userID)
	return err
}

func (s *DB) getAccessUsers(ctx context.Context, tableName, colName string, colValue int, rank accessType) ([]*User, error) {
	q := fmt.Sprintf("SELECT users.* FROM users INNER JOIN %s tb ON tb.user_id = users.id WHERE tb.%s = $1 AND tb.access = $2", tableName, colName)
	rows, err := s.conn.Query(ctx, q, colValue, rank)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*User{}, nil
	}
	if err != nil {
		return nil, err
	}
	users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[User])
	if err != nil {
		return nil, err
	}
	return users, nil
}
