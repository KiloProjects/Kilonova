package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

func (s *DB) updateManyToMany(ctx context.Context, tableName, parentKey, childKey string, parentID int, children []int, position bool) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		// Naively delete all associations, then add them back
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", tableName, parentKey), parentID); err != nil {
			return err
		}

		rows := [][]any{}
		colNames := []string{parentKey, childKey}
		if position {
			colNames = append(colNames, "position")
			for i, childID := range children {
				rows = append(rows, []any{parentID, childID, i})
			}
		} else {
			for _, childID := range children {
				rows = append(rows, []any{parentID, childID})
			}
		}

		_, err := tx.CopyFrom(ctx, pgx.Identifier{tableName}, colNames, pgx.CopyFromRows(rows))
		if err != nil {
			slog.WarnContext(ctx, "Could not update many2many", slog.Any("err", err))
			return err
		}

		return nil
	})
}
