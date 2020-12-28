package logic

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/KiloProjects/Kilonova/internal/db"
)

var (
	ErrBadTestFile = errors.New("Bad test score file")
	ErrBadArchive  = errors.New("Bad archive")
)

type archiveTest struct {
	InFile  io.Reader
	OutFile io.Reader
	Score   int
}

type ArchiveCtx struct {
	hasScoreFile bool
	tests        map[int64]archiveTest
	scoredTests  []int64
}

func NewArchiveCtx() *ArchiveCtx {
	return &ArchiveCtx{tests: make(map[int64]archiveTest), scoredTests: make([]int64, 0, 10), hasScoreFile: false}
}

func ProcessArchiveFile(ctx *ArchiveCtx, name string, file io.Reader) error {
	if strings.HasSuffix(name, ".txt") { // test score file

		// If there's multiple score files, quit
		if ctx.hasScoreFile {
			return ErrBadArchive
		}
		ctx.hasScoreFile = true

		br := bufio.NewScanner(file)

		for br.Scan() {
			line := br.Text()

			if line == "" { // empty line, skip
				continue
			}

			var testID int64
			var score int
			if _, err := fmt.Sscanf(line, "%d %d\n", &testID, &score); err != nil {
				log.Println(err)
				return ErrBadTestFile
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
			log.Println(br.Err())
		}
		return br.Err()
	}

	var tid int64
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		log.Println("Bad name:", name)
		return ErrBadArchive
	}

	if strings.HasSuffix(name, ".in") { // test input file
		tf := ctx.tests[tid]
		if tf.InFile != nil { // in file already exists
			log.Println("In file already declared")
			return ErrBadArchive
		}

		tf.InFile = file
		ctx.tests[tid] = tf
	}
	if strings.HasSuffix(name, ".out") || strings.HasSuffix(name, ".ok") { // test output file
		tf := ctx.tests[tid]
		if tf.OutFile != nil { // out file already exists
			log.Println("Out file already declared")
			return ErrBadArchive
		}

		tf.OutFile = file
		ctx.tests[tid] = tf
	}
	return nil
}

func (kn *Kilonova) ProcessZipTestArchive(pb *db.Problem, ar *zip.Reader) error {
	ctx := NewArchiveCtx()

	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}

		f, err := file.Open()
		if err != nil {
			log.Println(err)
			return errors.New("Unknown error")
		}
		defer f.Close() // This will always close all files, regardless of when the program leaves
		if err := ProcessArchiveFile(ctx, path.Base(file.Name), f); err != nil {
			return err
		}
	}

	if len(ctx.scoredTests) != len(ctx.tests) {
		log.Println("idK")
		return ErrBadArchive
	}

	for _, v := range ctx.tests {
		if v.InFile == nil || v.OutFile == nil {
			return ErrBadArchive
		}
	}

	if !ctx.hasScoreFile {
		return errors.New("Missing test score file")
	}

	// If we are loading an archive, the user might want to remove all tests first
	// So let's do it for them
	if err := pb.ClearTests(); err != nil {
		log.Println(err)
		return err
	}

	for testID, v := range ctx.tests {
		test, err := pb.CreateTest(testID, int32(v.Score))
		if err != nil {
			log.Println(err)
			return err
		}

		if err := kn.DM.SaveTestInput(test.ID, v.InFile); err != nil {
			log.Println("Couldn't create test input", err)
			return fmt.Errorf("Couldn't create test input: %w", err)
		}
		if err := kn.DM.SaveTestOutput(test.ID, v.OutFile); err != nil {
			log.Println("Couldn't create test output", err)
			return fmt.Errorf("Couldn't create test output: %w", err)
		}
	}

	return nil
}
