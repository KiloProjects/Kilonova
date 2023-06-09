package test

import (
	"archive/zip"
	"bufio"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func ProcessScoreFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	f, err := file.Open()
	if err != nil {
		return kilonova.Statusf(500, "Unknown error")
	}
	defer f.Close()

	br := bufio.NewScanner(f)

	for br.Scan() {
		line := br.Text()

		if line == "" { // empty line, skip
			continue
		}

		var testID int
		var score int
		if _, err := fmt.Sscanf(line, "%d %d\n", &testID, &score); err != nil {
			// Might just be a bad line
			continue
		}

		test := ctx.tests[testID]
		test.Score = score
		ctx.tests[testID] = test
		for _, ex := range ctx.scoredTests {
			if ex == testID {
				return ErrBadTestFile
			}
		}

		ctx.scoredTests = append(ctx.scoredTests, testID)
	}
	if br.Err() != nil {
		zap.S().Info(br.Err())
		return kilonova.WrapError(err, "Score file read error")
	}
	return nil
}
