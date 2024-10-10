package test

import (
	"archive/zip"
	"fmt"
	"log/slog"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/shopspring/decimal"
)

type testIDMode int

const (
	idModeParse testIDMode = iota
	idModeSort
	idModeParseSort
)

var (
	ErrInvalidTestID = kilonova.Statusf(400, "Invalid test ID")
)

type archiveTest struct {
	InFile  *zip.File
	OutFile *zip.File

	VisibleID int
	Key       string
	Score     decimal.Decimal
}

func (t archiveTest) Matches(re *regexp.Regexp) bool {
	places := re.FindStringIndex(t.Key)
	if places == nil {
		return false
	}
	return places[0] == 0
}

func ProcessTestInputFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	testName := path.Base(file.Name)
	if slices.Contains(testInputSuffixes, path.Ext(testName)) {
		testName = strings.TrimSuffix(testName, path.Ext(testName))
	} else {
		testName = strings.TrimPrefix(testName, "input")
	}
	tf := ctx.tests[testName]
	if tf.InFile != nil { // in file already exists
		return kilonova.Statusf(400, "Multiple input files for test %q", testName)
	}

	tf.InFile = file
	tf.Key = testName
	ctx.tests[testName] = tf
	return nil
}

func ProcessTestOutputFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	testName := path.Base(file.Name)
	if slices.Contains(testOutputSuffixes, path.Ext(testName)) {
		testName = strings.TrimSuffix(testName, path.Ext(testName))
	} else {
		testName = strings.TrimPrefix(testName, "output")
	}
	tf := ctx.tests[testName]
	if tf.OutFile != nil { // out file already exists
		return kilonova.Statusf(400, "Multiple output files for test %q", testName)
	}

	tf.OutFile = file
	tf.Key = testName
	ctx.tests[testName] = tf
	return nil
}

func getTestID(name string) (int, *kilonova.StatusError) {
	var tid int
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		// check grader_test%d format
		if _, err = fmt.Sscanf(name, "grader_test%d", &tid); err == nil {
			return tid, nil
		}

		// maybe it's problem_name[.-_]%d.{in,sol,out} format
		if ext := path.Ext(name); slices.Contains(testInputSuffixes, ext) || slices.Contains(testOutputSuffixes, ext) {
			name = strings.TrimSuffix(name, ext)
		}
		nm := strings.FieldsFunc(name, func(r rune) bool {
			return r == '-' || r == '.' || r == '_'
		})
		if len(nm) == 0 {
			return -1, ErrInvalidTestID
		}
		val, err := strconv.Atoi(nm[len(nm)-1])
		if err != nil {
			return -1, ErrInvalidTestID
		}
		return val, nil
	}
	return tid, nil
}

func deduceTestIDMode(ctx *ArchiveCtx) testIDMode {
	// Get initial mode by analyzing test IDs
	mode := idModeParse
	for key := range ctx.tests {
		if _, err := getTestID(key); err != nil {
			mode = idModeSort
			slog.Debug("Using `sort` ID mode")
			break
		}
	}

	// Rationale for the following `if` statement:
	// Score parameters that are based on regexes should just take that initial string, so just return the sorting mode.
	// If they are based on the test count, however, that's where things get interesting
	// If test IDs are not ok, then just sort
	// However, if test IDs are ok, they should be a better source of truth, they should be parsed,
	// automatically padded and then sorted. For example, if the single-digit ID tests are not prefixed
	// like `01` (it simply performs string comparisons), that would mess up the sorting process and produce incorrect results
	// In theory this should never happen since most uses of score parameters come from CMS that has a stricter format
	// but we want to ensure the most ergonomic proposer experience
	if len(ctx.scoreParameters) > 0 {
		if ctx.scoreParameters[0].Match != nil {
			// Regex mode, just sort
			return idModeSort
		}
		if ctx.scoreParameters[0].Count != nil && mode == idModeParse {
			mode = idModeParseSort
			slog.Debug("Using `parseSort` ID mode")
		}
	}
	return mode
}
