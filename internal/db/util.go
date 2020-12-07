package db

import (
	"context"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
)

func (db *DB) MaxScore(ctx context.Context, userID int64, pbID int64) int {
	score, err := db.raw.MaxScore(ctx, rawdb.MaxScoreParams{UserID: userID, ProblemID: pbID})
	if err != nil {
		score = 0
	}
	return int(score)
}
