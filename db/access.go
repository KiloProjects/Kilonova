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

type userAccess struct {
	UserID int        `db:"user_id"`
	Access accessType `db:"access"`
}

func (s *DB) getAccess(ctx context.Context, tableName, colName string, colValue int) ([]*userAccess, error) {
	var rights []*userAccess
	q := fmt.Sprintf("SELECT user_id, access FROM %s WHERE %s = $1", tableName, colName)
	err := s.conn.SelectContext(ctx, &rights, q, colValue)
	if errors.Is(err, sql.ErrNoRows) {
		return []*userAccess{}, nil
	}
	if err != nil {
		return nil, err
	}
	return rights, nil
}

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
