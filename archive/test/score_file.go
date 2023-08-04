package test

import (
	"archive/zip"
	"bufio"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// TestID -> Score
type ScoreFileEntries = map[int]decimal.Decimal

func ParseScoreFile(file *zip.File) (ScoreFileEntries, *kilonova.StatusError) {
	f, err := file.Open()
	if err != nil {
		return nil, kilonova.Statusf(500, "Unknown error")
	}
	defer f.Close()

	br := bufio.NewScanner(f)

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
		zap.S().Info(br.Err())
		return nil, kilonova.WrapError(err, "Score file read error")
	}

	return rez, nil
}
