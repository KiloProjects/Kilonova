package test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/shopspring/decimal"
)

// TestID -> Score
type ScoreFileEntries = map[int]decimal.Decimal

func ParseScoreFile(ctx context.Context, r io.Reader) (ScoreFileEntries, error) {
	br := bufio.NewScanner(r)

	rez := make(ScoreFileEntries)

	for br.Scan() {
		line := br.Text()

		if line == "" || line[0] == '#' { // empty/comment line, skip
			continue
		}

		var testID int
		var score string
		if _, err := fmt.Sscanf(line, "%d %s\n", &testID, &score); err != nil {
			// Might just be a bad line
			continue
		}

		if _, ok := rez[testID]; ok {
			return nil, ErrBadTestFile
		}

		dec, err := decimal.NewFromString(score)
		if err != nil {
			// Bad line
			continue
		}
		rez[testID] = dec
	}
	if br.Err() != nil {
		slog.InfoContext(ctx, "Could not read score file", slog.Any("err", br.Err()))
		return nil, fmt.Errorf("Score file read error: %w", br.Err())
	}

	return rez, nil
}
