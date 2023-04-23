package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type accessType string

const (
	accessEditor accessType = "editor"
	accessViewer accessType = "viewer"
)

func (s *DB) addAccess(ctx context.Context, tableName, colName string, colValue, userID int, rank accessType) error {
	q := fmt.Sprintf("INSERT INTO %s (%s, user_id, access) VALUES ($1, $2, $3)", tableName, colName)
	_, err := s.conn.ExecContext(ctx, q, colValue, userID, rank)
	return err
}

func (s *DB) removeAccess(ctx context.Context, tableName, colName string, colValue, userID int) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE %s = $1 AND user_id = $2", tableName, colName)
	_, err := s.conn.ExecContext(ctx, q, colValue, userID)
	return err
}

func (s *DB) getAccessUsers(ctx context.Context, tableName, colName string, colValue int, rank accessType) ([]*User, error) {
	var users []*User
	q := fmt.Sprintf("SELECT users.* FROM users INNER JOIN %s tb ON tb.user_id = users.id WHERE tb.%s = $1 AND tb.access = $2", tableName, colName)
	err := s.conn.SelectContext(ctx, &users, q, colValue, rank)
	if errors.Is(err, sql.ErrNoRows) {
		return []*User{}, nil
	}
	if err != nil {
		return nil, err
	}
	return users, nil
}
