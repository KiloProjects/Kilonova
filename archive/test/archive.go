package test

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
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
	props        *Properties
}

type Subtask struct {
	Score int
	Tests []int
}

type Properties struct {
	Subtasks map[int]Subtask
	// seconds
	TimeLimit float64
	// kbytes
	MemoryLimit int

	SubtaskedTests []int
}

func NewArchiveCtx() *ArchiveCtx {
	return &ArchiveCtx{tests: make(map[int]archiveTest), scoredTests: make([]int, 0, 10), hasScoreFile: false}
}

func ProcessScoreFile(ctx *ArchiveCtx, file *zip.File) error {
	f, err := file.Open()
	if err != nil {
		return errors.New("Unknown error")
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
			zap.S().Info(err)
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
		zap.S().Info(br.Err())
	}
	return br.Err()
}

func ProcessArchiveFile(ctx *ArchiveCtx, file *zip.File) error {
	name := path.Base(file.Name)
	if strings.HasSuffix(name, ".txt") { // test score file
		return ProcessScoreFile(ctx, file)
	}

	if strings.HasSuffix(name, ".properties") { // test properties file
		return ProcessPropertiesFile(ctx, file)
	}

	// if nothing else is detected, it should be a test file

	var tid int
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		// maybe it's problem_name.%d.{in,sol,out} format
		nm := strings.Split(strings.TrimSuffix(name, path.Ext(name)), ".")
		if len(nm) == 0 {
			zap.S().Info("Bad name:", name)
			return ErrBadArchive
		}
		val, err := strconv.Atoi(nm[len(nm)-1])
		if err != nil {
			zap.S().Info("Not number:", name)
			return ErrBadArchive
		}
		tid = val
	}

	if strings.HasSuffix(name, ".in") { // test input file
		tf := ctx.tests[tid]
		if tf.InFile != nil { // in file already exists
			return fmt.Errorf("Multiple input files for test %d", tid)
		}

		tf.InFile = file
		ctx.tests[tid] = tf
	}
	if strings.HasSuffix(name, ".out") || strings.HasSuffix(name, ".ok") || strings.HasSuffix(name, ".sol") { // test output file
		tf := ctx.tests[tid]
		if tf.OutFile != nil { // out file already exists
			return fmt.Errorf("Multiple output files for test %d", tid)
		}

		tf.OutFile = file
		ctx.tests[tid] = tf
	}
	return nil
}

func ProcessZipTestArchive(ctx context.Context, pb *kilonova.Problem, ar *zip.Reader, base *sudoapi.BaseAPI) error {
	aCtx := NewArchiveCtx()

	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := ProcessArchiveFile(aCtx, file); err != nil {
			return err
		}
	}

	if aCtx.hasScoreFile && len(aCtx.scoredTests) != len(aCtx.tests) {
		zap.S().Info(len(aCtx.scoredTests), len(aCtx.tests))
		return errors.New("Mismatched number of tests in archive and scored tests")
	}

	if aCtx.props != nil && aCtx.props.Subtasks != nil && len(aCtx.props.SubtaskedTests) != len(aCtx.tests) {
		zap.S().Info(len(aCtx.props.SubtaskedTests), len(aCtx.tests))
		return errors.New("Mismatched number of tests in archive and tests that correspond to at least one subtask")
	}

	for k, v := range aCtx.tests {
		if v.InFile == nil || v.OutFile == nil {
			return fmt.Errorf("Missing input or output file for test %d", k)
		}
	}

	if !aCtx.hasScoreFile {
		zap.S().Info("Automatically inserting scores...")
		n := len(aCtx.tests)
		perTest := 100/n + 1
		toSub := n - 100%n
		k := 0
		for i := range aCtx.tests {
			tst := aCtx.tests[i]
			tst.Score = perTest
			if k < toSub {
				tst.Score--
			}
			aCtx.tests[i] = tst
			k++
		}
	}

	// If we are loading an archive, the user might want to remove all tests first
	// So let's do it for them
	if err := base.OrphanTests(ctx, pb.ID); err != nil {
		zap.S().Warn(err)
		return err
	}

	createdTests := map[int]kilonova.Test{}

	for testID, v := range aCtx.tests {
		var test kilonova.Test
		test.ProblemID = pb.ID
		test.VisibleID = testID
		test.Score = v.Score
		if err := base.CreateTest(ctx, &test); err != nil {
			zap.S().Warn(err)
			return err
		}

		createdTests[testID] = test

		f, err := v.InFile.Open()
		if err != nil {
			return fmt.Errorf("Couldn't open() input file: %w", err)
		}
		if err := base.SaveTestInput(test.ID, f); err != nil {
			zap.S().Warn("Couldn't create test input", err)
			f.Close()
			return fmt.Errorf("Couldn't create test input: %w", err)
		}
		f.Close()
		f, err = v.OutFile.Open()
		if err != nil {
			return fmt.Errorf("Couldn't open() output file: %w", err)
		}
		if err := base.SaveTestOutput(test.ID, f); err != nil {
			zap.S().Warn("Couldn't create test output", err)
			f.Close()
			return fmt.Errorf("Couldn't create test output: %w", err)
		}
		f.Close()
	}

	if aCtx.props != nil {
		shouldUpd := false
		upd := kilonova.ProblemUpdate{}
		if aCtx.props.MemoryLimit != 0 {
			shouldUpd = true
			upd.MemoryLimit = &aCtx.props.MemoryLimit
		}
		if aCtx.props.TimeLimit != 0 {
			shouldUpd = true
			upd.TimeLimit = &aCtx.props.TimeLimit
		}

		if shouldUpd {
			if err := base.UpdateProblem(ctx, pb.ID, upd, nil); err != nil {
				zap.S().Warn(err)
				return fmt.Errorf("Couldn't update problem medatada: %w", err)
			}
		}

		if aCtx.props.Subtasks != nil {
			if err := base.DeleteSubTasks(ctx, pb.ID); err != nil {
				zap.S().Warn(err)
				return fmt.Errorf("Couldn't delete existing subtasks: %w", err)
			}
			for stkId, stk := range aCtx.props.Subtasks {
				outStk := kilonova.SubTask{
					ProblemID: pb.ID,
					VisibleID: stkId,
					Score:     stk.Score,
					Tests:     []int{},
				}
				for _, test := range stk.Tests {
					if tt, exists := createdTests[test]; !exists {
						return fmt.Errorf("Test %d not found in added tests. Aborting subtask creation", test)
					} else {
						outStk.Tests = append(outStk.Tests, tt.ID)
					}
				}

				if err := base.CreateSubTask(ctx, &outStk); err != nil {
					zap.S().Warn(err)
					return fmt.Errorf("Couldn't create subtask: %w", err)
				}
			}
		}
	}

	return nil
}

func eq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Ints(a)
	sort.Ints(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
