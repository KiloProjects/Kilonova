package test

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var (
	ErrBadTestFile = kilonova.Statusf(400, "Bad test score file")
	ErrBadArchive  = kilonova.Statusf(400, "Bad archive")
)

type archiveTest struct {
	InFile  *zip.File
	OutFile *zip.File
	Score   int
}

type ArchiveCtx struct {
	hasScoreFile bool
	tests        map[int]archiveTest
	scoredTests  []int
}

func NewArchiveCtx() *ArchiveCtx {
	return &ArchiveCtx{tests: make(map[int]archiveTest), scoredTests: make([]int, 0, 10), hasScoreFile: false}
}

func ProcessArchiveFile(ctx *ArchiveCtx, name string, file *zip.File) error {
	if strings.HasSuffix(name, ".txt") { // test score file

		f, err := file.Open()
		if err != nil {
			return kilonova.Statusf(500, "unknown error")
		}
		defer f.Close()

		// If there's multiple score files, quit
		if ctx.hasScoreFile {
			return ErrBadArchive
		}
		ctx.hasScoreFile = true

		br := bufio.NewScanner(f)

		for br.Scan() {
			line := br.Text()

			if line == "" { // empty line, skip
				continue
			}

			var testID int
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

	var tid int
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		// maybe it's problem_name.%d.{in,sol,out} format
		nm := strings.Split(strings.TrimSuffix(name, path.Ext(name)), ".")
		if len(nm) == 0 {
			log.Println("Bad name:", name)
			return ErrBadArchive
		}
		val, err := strconv.Atoi(nm[len(nm)-1])
		if err != nil {
			log.Println("Not number:", name)
			return ErrBadArchive
		}
		tid = val
	}

	if strings.HasSuffix(name, ".in") { // test input file
		tf := ctx.tests[tid]
		if tf.InFile != nil { // in file already exists
			return fmt.Errorf("multiple input files for test %d", tid)
		}

		tf.InFile = file
		ctx.tests[tid] = tf
	}
	if strings.HasSuffix(name, ".out") || strings.HasSuffix(name, ".ok") || strings.HasSuffix(name, ".sol") { // test output file
		tf := ctx.tests[tid]
		if tf.OutFile != nil { // out file already exists
			return fmt.Errorf("multiple output files for test %d", tid)
		}

		tf.OutFile = file
		ctx.tests[tid] = tf
	}
	return nil
}

func ProcessZipTestArchive(pb *kilonova.Problem, ar *zip.Reader, base *sudoapi.BaseAPI) error {
	ctx := NewArchiveCtx()

	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := ProcessArchiveFile(ctx, path.Base(file.Name), file); err != nil {
			return err
		}
	}

	if ctx.hasScoreFile && len(ctx.scoredTests) != len(ctx.tests) {
		log.Println(len(ctx.scoredTests), len(ctx.tests))
		return kilonova.Statusf(400, "mismatched number of tests in archive and scored tests")
	}

	for k, v := range ctx.tests {
		if v.InFile == nil || v.OutFile == nil {
			return kilonova.Statusf(400, "missing input or output file for test %d", k)
		}
	}

	if !ctx.hasScoreFile {
		log.Println("Automatically inserting scores...")
		n := len(ctx.tests)
		perTest := 100/n + 1
		toSub := n - 100%n
		k := 0
		for i := range ctx.tests {
			tst := ctx.tests[i]
			tst.Score = perTest
			if k < toSub {
				tst.Score--
			}
			ctx.tests[i] = tst
			k++
		}
	}

	// If we are loading an archive, the user might want to remove all tests first
	// So let's do it for them
	if err := base.OrphanTests(context.Background(), pb.ID); err != nil {
		log.Println(err)
		return err
	}

	for testID, v := range ctx.tests {
		var test kilonova.Test
		test.ProblemID = pb.ID
		test.VisibleID = testID
		test.Score = v.Score
		if err := base.CreateTest(context.Background(), &test); err != nil {
			log.Println(err)
			return err
		}

		{
			f, err := v.InFile.Open()
			if err != nil {
				return kilonova.WrapError(err, "Couldn't open() input file")
			}
			defer f.Close()
			if err := base.SaveTestInput(test.ID, f); err != nil {
				log.Println("Couldn't create test input", err)
				return kilonova.WrapError(err, "Couldn't create test input")
			}
		}
		{
			f, err := v.OutFile.Open()
			if err != nil {
				return kilonova.WrapError(err, "Couldn't open() output file")
			}
			defer f.Close()
			if err := base.SaveTestOutput(test.ID, f); err != nil {
				log.Println("Couldn't create test output", err)
				return kilonova.WrapError(err, "Couldn't create test output")
			}
		}
	}

	return nil
}
