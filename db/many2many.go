package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (s *DB) updateManyToMany(ctx context.Context, tableName, parentKey, childKey string, parentID int, children []int, position bool) error {
	return pgx.BeginFunc(ctx, s.pgconn, func(tx pgx.Tx) error {
		// Naively delete all associations, then add them back
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tableName, parentKey), parentID); err != nil {
			return err
		}

		for i, childID := range children {
			if position {
				q := fmt.Sprintf("INSERT INTO %s (%s, %s, position) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", tableName, parentKey, childKey)

				if _, err := tx.Exec(ctx, q, parentID, childID, i); err != nil {
					zap.S().Warn(err)
					return err
				}
			} else {
				q := fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES ($1, $2) ON CONFLICT DO NOTHING", tableName, parentKey, childKey)
				if _, err := tx.Exec(ctx, q, parentID, childID); err != nil {
					zap.S().Warn(err)
					return err
				}
			}
		}
		return nil
	})
}
