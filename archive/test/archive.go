package test

import (
	"archive/zip"
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	ErrBadTestFile = kilonova.Statusf(400, "Bad test score file")
	ErrBadArchive  = kilonova.Statusf(400, "Bad archive")
)

type ArchiveCtx struct {
	tests       map[string]archiveTest
	attachments map[string]archiveAttachment
	props       *properties

	submissions []*submissionStub

	params *TestProcessParams

	scoreParameters []ScoreParamEntry

	testScores ScoreFileEntries
}

type properties struct {
	Subtasks map[int]Subtask
	// seconds
	TimeLimit *float64
	// kbytes
	MemoryLimit *int

	Tags         []*mockTag
	Source       *string
	ConsoleInput *bool
	TestName     *string
	ProblemName  *string

	DefaultPoints *decimal.Decimal

	SubtaskedTests []int

	Editors []string

	ScorePrecision  *int32
	ScoringStrategy kilonova.ScoringType
}

func NewArchiveCtx(params *TestProcessParams) *ArchiveCtx {
	return &ArchiveCtx{
		tests:       make(map[string]archiveTest),
		attachments: make(map[string]archiveAttachment),
		testScores:  make(ScoreFileEntries),

		params: params,
	}
}

var (
	testInputSuffixes  = []string{".in", ".input"}
	testOutputSuffixes = []string{".out", ".output", ".ok", ".sol", ".a", ".ans"}
)

func ProcessArchiveFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	if strings.Contains(file.Name, "__MACOSX") { // Support archives from MacOS by skipping MACOSX directory
		return nil
	}
	if slices.Contains(filepath.SplitList(path.Dir(file.Name)), "attachments") { // Is in "attachments" directory
		return ProcessAttachmentFile(ctx, file)
	}

	if slices.Contains(filepath.SplitList(path.Dir(file.Name)), "submissions") { // Is in "submissions" directory
		return ProcessSubmissionFile(ctx, file)
	}

	ext := strings.ToLower(path.Ext(file.Name))
	if ext == ".txt" { // test score file
		// if using score parameters, test score file is redundant
		if len(ctx.scoreParameters) > 0 {
			return kilonova.Statusf(400, "Archive cannot contain tests.txt if you specified score parameters")
		}

		vals, err := ParseScoreFile(file)
		if err != nil {
			return err
		}

		ctx.testScores = vals
		return nil
	}

	if ext == ".properties" { // test properties file
		return ProcessPropertiesFile(ctx, file)
	}

	if strings.ToLower(file.Name) == "problem.xml" { // Polygon archive format
		r, err := file.Open()
		if err != nil {
			return kilonova.WrapError(err, "Could not open problem.xml")
		}
		defer r.Close()
		return ProcessProblemXMLFile(ctx, r)
	}

	// Polygon-specific handling
	if ctx.params.Polygon {
		if strings.HasPrefix(file.Name, "solutions") {
			return ProcessSubmissionFile(ctx, file)
		}

		if strings.HasPrefix(file.Name, "tests") {
			if ext == ".a" {
				return ProcessTestOutputFile(ctx, file)
			}

			return ProcessTestInputFile(ctx, file)
		}

		if file.Name == "check.cpp" {
			return ProcessPolygonCheckFile(ctx, file)
		}

		return nil
	}

	// if nothing else is detected, it should be a test file
	if slices.Contains(testInputSuffixes, ext) || strings.HasPrefix(file.Name, "input") { // test input file (ex: 01.in)
		return ProcessTestInputFile(ctx, file)
	}

	if slices.Contains(testOutputSuffixes, ext) || strings.HasPrefix(file.Name, "output") { // test output file (ex: 01.out/01.ok)
		return ProcessTestOutputFile(ctx, file)
	}

	return nil
}

type TestProcessParams struct {
	Requestor *kilonova.UserFull

	ScoreParamsStr string

	Polygon          bool
	MergeAttachments bool

	// MergeTests bool
}

func ProcessZipTestArchive(ctx context.Context, pb *kilonova.Problem, ar *zip.Reader, base *sudoapi.BaseAPI, params *TestProcessParams) *kilonova.StatusError {
	if params.Requestor == nil {
		return kilonova.Statusf(400, "There must be a requestor")
	}

	aCtx := NewArchiveCtx(params)

	// Try to autodetect polygon archive
	if _, err := fs.Stat(ar, "problem.xml"); err == nil {
		aCtx.params.Polygon = true
		aCtx.params.MergeAttachments = true
	}

	if len(params.ScoreParamsStr) > 0 {
		scoreParams, err := ParseScoreParameters([]byte(params.ScoreParamsStr))
		if err != nil {
			return err
		}
		aCtx.scoreParameters = scoreParams
	}

	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := ProcessArchiveFile(aCtx, file); err != nil {
			return err
		}
	}

	if aCtx.props != nil && aCtx.props.Subtasks != nil && len(aCtx.props.SubtaskedTests) != len(aCtx.tests) {
		zap.S().Info(len(aCtx.props.SubtaskedTests), len(aCtx.tests))
		return kilonova.Statusf(400, "Mismatched number of tests in archive and tests that correspond to at least one subtask")
	}

	for k, v := range aCtx.tests {
		if v.InFile == nil || v.OutFile == nil {
			return kilonova.Statusf(400, "Missing input or output file for test %q", k)
		}
	}

	// The archive may not have tests
	if len(aCtx.tests) > 0 {
		idMode := deduceTestIDMode(aCtx)

		// testsByID := make(map[int]*archiveTest)
		var tests []archiveTest
		if idMode == idModeParse || idMode == idModeParseSort {
			tests = make([]archiveTest, 0, len(aCtx.tests))
			for k, v := range aCtx.tests {
				v := v
				// Error ommited since deduceTestIDMode already checks for error on parse mode
				v.VisibleID, _ = getTestID(k)
				v.Key = fmt.Sprintf("%04d", v.VisibleID)
				tests = append(tests, v)
			}
		}

		if idMode == idModeParseSort {
			aCtx.tests = make(map[string]archiveTest)
			for _, test := range tests {
				aCtx.tests[test.Key] = test
			}
		}

		if idMode == idModeSort || idMode == idModeParseSort {
			tests = make([]archiveTest, 0, len(aCtx.tests))
			for _, test := range aCtx.tests {
				tests = append(tests, test)
			}
			slices.SortFunc(tests, func(a, b archiveTest) int {
				return cmp.Compare(a.Key, b.Key)
			})
			for i := range tests {
				tests[i].VisibleID = i
			}
		}

		// Sanity check: sort tests by visible IDs since idModeParse might not handle them correctly
		slices.SortFunc(tests, func(a, b archiveTest) int {
			return cmp.Compare(a.VisibleID, b.VisibleID)
		})

		if isMaskedScoring(aCtx.scoreParameters, tests) {
			buildParamTestScores(aCtx, tests)
			aCtx.scoreParameters = aCtx.scoreParameters[:0]
		}

		precision := pb.ScorePrecision
		if aCtx.props != nil && aCtx.props.ScorePrecision != nil {
			precision = *aCtx.props.ScorePrecision
		}

		var mustAutofillTests bool = false
		for i := range tests {
			val, ok := aCtx.testScores[tests[i].VisibleID]
			if !ok {
				// Mark as needing score
				mustAutofillTests = true
				tests[i].Score = decimal.NewFromInt(-1)
			} else {
				tests[i].Score = val.Round(int32(precision))
			}
		}

		if mustAutofillTests {
			// Try to deduce scoring for remaining tests
			// zap.S().Info("Automatically inserting scores...")
			var n decimal.Decimal
			totalScore := decimal.NewFromInt(100)
			for _, test := range tests {
				if test.Score.IsPositive() {
					totalScore = totalScore.Sub(test.Score)
				} else {
					n = n.Add(decimal.NewFromInt(1))
				}
			}

			perTest := totalScore.Div(n).RoundDown(int32(precision))
			toAdd := decimal.Zero
			dif := totalScore.Sub(perTest.Mul(n))
			if !dif.IsZero() {
				// If not zero, we need to compensate on some tests
				// But keep the delta <= 1.0 points
				// totalScore > perTest*n, since we rounded down

				// divide the difference by its ceiling to get the delta to insert to scores
				// we'll handle with rounding approximations later.
				toAdd = dif.DivRound(dif.Ceil(), int32(precision))
			}
			k := 0

			for i := range tests {
				if tests[i].Score.Equal(decimal.NewFromInt(-1)) {
					tests[i].Score = perTest
					if !dif.IsZero() {
						tests[i].Score = tests[i].Score.Add(toAdd)
						dif = dif.Sub(toAdd)
						if !dif.IsZero() && dif.Abs().LessThan(toAdd) {
							// Pour the remaining difference here
							// This should fix the roundings
							tests[i].Score = tests[i].Score.Add(dif)
							toAdd = decimal.Zero
						}
					}
					k++
				}
			}
		}

		// If we are loading an archive, the user might want to remove all tests first
		// So let's do it for them
		if err := base.DeleteTests(ctx, pb.ID); err != nil {
			zap.S().Warn(err)
			return err
		}

		createdTests := map[int]kilonova.Test{}

		for _, v := range tests {
			var test kilonova.Test
			test.ProblemID = pb.ID
			test.VisibleID = v.VisibleID
			test.Score = v.Score
			if err := base.CreateTest(ctx, &test); err != nil {
				zap.S().Warn(err)
				return err
			}

			createdTests[v.VisibleID] = test

			f, err := v.InFile.Open()
			if err != nil {
				return kilonova.WrapError(err, "Couldn't open() input file")
			}
			if err := base.SaveTestInput(test.ID, f); err != nil {
				zap.S().Warn("Couldn't create test input", err)
				f.Close()
				return kilonova.WrapError(err, "Couldn't create test input")
			}
			f.Close()
			f, err = v.OutFile.Open()
			if err != nil {
				return kilonova.WrapError(err, "Couldn't open() output file")
			}
			if err := base.SaveTestOutput(test.ID, f); err != nil {
				zap.S().Warn("Couldn't create test output", err)
				f.Close()
				return kilonova.WrapError(err, "Couldn't create test output")
			}
			f.Close()
		}

		if err := base.DeleteSubTasks(ctx, pb.ID); err != nil {
			zap.S().Warn(err)
			return kilonova.WrapError(err, "Couldn't delete existing subtasks")
		}
		if len(aCtx.scoreParameters) > 0 {
			// Decide subtasks based on score parameters, if they exist
			startIdx := 0
			for i, entry := range aCtx.scoreParameters {
				testIDs := []int{}
				if entry.Count != nil {
					if startIdx < len(tests) {
						for i := startIdx; i < startIdx+*entry.Count && i < len(tests); i++ {
							test, ok := createdTests[tests[i].VisibleID]
							if !ok {
								zap.S().Warn("Created test not found anymore", tests[i].VisibleID)
								continue
							}
							testIDs = append(testIDs, test.ID)
						}
						startIdx += *entry.Count
					}
				} else if entry.Match != nil {
					for _, test := range tests {
						if test.Matches(entry.Match) {
							test, ok := createdTests[test.VisibleID]
							if !ok {
								zap.S().Warn("Created test not found anymore", test.VisibleID)
								continue
							}
							testIDs = append(testIDs, test.ID)
						}
					}
				} else {
					zap.S().Warn("Somehow score param doesn't have neither count nor match non-nil")
				}
				if len(testIDs) > 0 {
					// Tests are found, create subtask
					if err := base.CreateSubTask(ctx, &kilonova.SubTask{
						ProblemID: pb.ID,
						VisibleID: i + 1,
						Score:     entry.Score,
						Tests:     testIDs,
					}); err != nil {
						zap.S().Warn(err)
						return kilonova.WrapError(err, "Couldn't create subtask")
					}
				}
			}
		} else if aCtx.props != nil && aCtx.props.Subtasks != nil {
			// Else, decide subtasks based on grader.properties
			for stkId, stk := range aCtx.props.Subtasks {
				tests := make([]int, 0, len(stk.Tests))
				for _, test := range stk.Tests {
					if tt, exists := createdTests[test]; !exists {
						return kilonova.Statusf(400, "Test %d not found in added tests. Aborting subtask creation", test)
					} else {
						tests = append(tests, tt.ID)
					}
				}

				if err := base.CreateSubTask(ctx, &kilonova.SubTask{
					ProblemID: pb.ID,
					VisibleID: stkId,
					Score:     stk.Score,
					Tests:     tests,
				}); err != nil {
					zap.S().Warn(err)
					return kilonova.WrapError(err, "Couldn't create subtask")
				}
			}
		}
	}

	if len(aCtx.attachments) > 0 {
		if err := createAttachments(ctx, aCtx, pb, base, params); err != nil {
			return err
		}
	}

	if aCtx.props != nil {
		shouldUpd := false
		upd := kilonova.ProblemUpdate{}
		if aCtx.props.MemoryLimit != nil {
			shouldUpd = true
			upd.MemoryLimit = aCtx.props.MemoryLimit
		}
		if aCtx.props.TimeLimit != nil {
			shouldUpd = true
			upd.TimeLimit = aCtx.props.TimeLimit
		}
		if aCtx.props.DefaultPoints != nil {
			shouldUpd = true
			upd.DefaultPoints = aCtx.props.DefaultPoints
		}
		if aCtx.props.Source != nil {
			shouldUpd = true
			upd.SourceCredits = aCtx.props.Source
		}
		if aCtx.props.ConsoleInput != nil {
			shouldUpd = true
			upd.ConsoleInput = aCtx.props.ConsoleInput
		}
		if aCtx.props.ScoringStrategy != kilonova.ScoringTypeNone {
			shouldUpd = true
			upd.ScoringStrategy = aCtx.props.ScoringStrategy
		}
		if aCtx.props.ScorePrecision != nil {
			shouldUpd = true
			upd.ScorePrecision = aCtx.props.ScorePrecision
		}
		if aCtx.props.TestName != nil {
			shouldUpd = true
			upd.TestName = aCtx.props.TestName
		}

		if aCtx.props.ProblemName != nil && *aCtx.props.ProblemName != "" {
			shouldUpd = true
			upd.Name = aCtx.props.ProblemName
		}

		if shouldUpd {
			if err := base.UpdateProblem(ctx, pb.ID, upd, nil); err != nil {
				zap.S().Warn(err)
				return kilonova.WrapError(err, "Couldn't update problem medatada")
			}
		}

		if len(aCtx.props.Tags) > 0 {
			realTagIDs := []int{}
			for _, mTag := range aCtx.props.Tags {
				tag, err := base.TagByLooseName(ctx, mTag.Name)
				if err != nil || tag == nil {
					id, err := base.CreateTag(ctx, mTag.Name, mTag.Type)
					if err != nil {
						zap.S().Warn("Couldn't create tag: ", err)
						continue
					}
					realTagIDs = append(realTagIDs, id)
					continue
				}
				realTagIDs = append(realTagIDs, tag.ID)
			}
			if err := base.UpdateProblemTags(ctx, pb.ID, realTagIDs); err != nil {
				zap.S().Warn(err)
				return kilonova.WrapError(err, "Couldn't update tags")
			}
		}

		if len(aCtx.props.Editors) > 0 {
			var newEditors []*kilonova.UserBrief
			// If user is not admin, add user to new editor list
			// Since they should always be a part of the problem editor team
			if !params.Requestor.Admin {
				newEditors = append(newEditors, params.Requestor.Brief())
			}
			// First, get the new editors to make sure they are valid
			for _, editor := range aCtx.props.Editors {
				user, err := base.UserBriefByName(ctx, editor)
				if err == nil && user != nil {
					newEditors = append(newEditors, user)
				}
			}

			if len(newEditors) > 0 {
				// Then, remove existing editors
				cEditors, err := base.ProblemEditors(ctx, pb.ID)
				if err != nil {
					zap.S().Warn(err)
					return err
				}
				for _, ed := range cEditors {
					if err := base.StripProblemAccess(ctx, pb.ID, ed.ID); err != nil {
						zap.S().Warn(err)
					}
				}

				// Lastly, add the new editors
				for _, editor := range newEditors {
					if err := base.AddProblemEditor(ctx, pb.ID, editor.ID); err != nil {
						zap.S().Warn(err)
					}
				}
			}

		}

	}

	// Do submissions at the end after all changes have been merged
	if len(aCtx.submissions) > 0 {
		for _, sub := range aCtx.submissions {
			lang, ok := eval.Langs[sub.lang]
			if !ok {
				zap.S().Warn("Skipping submission")
				continue
			}
			if _, err := base.CreateSubmission(ctx, params.Requestor, pb, sub.code, lang, nil, true); err != nil {
				zap.S().Warn(err)
			}
		}
	}

	return nil
}

func createAttachments(ctx context.Context, aCtx *ArchiveCtx, pb *kilonova.Problem, base *sudoapi.BaseAPI, params *TestProcessParams) *kilonova.StatusError {
	atts, err := base.ProblemAttachments(ctx, pb.ID)
	if err != nil {
		zap.S().Warn("Couldn't get problem attachments")
		return kilonova.WrapError(err, "Couldn't get attachments")
	}

	// attachment IDs to mark for deletion
	var attIDs []int
	for _, att := range atts {
		if aCtx.params.MergeAttachments {
			// TODO: Use map for proper lookup, instead of O(N*M) nested for
			for _, newAtt := range aCtx.attachments {
				if newAtt.Name == att.Name {
					attIDs = append(attIDs, att.ID)
					break
				}
			}
		} else {
			attIDs = append(attIDs, att.ID)
		}
	}
	if len(attIDs) > 0 {
		if _, err := base.DeleteProblemAtts(ctx, pb.ID, attIDs); err != nil {
			zap.S().Warn("Couldn't remove attachments")
			return kilonova.WrapError(err, "Couldn't delete attachments")
		}
	}
	for _, att := range aCtx.attachments {
		if att.File == nil {
			zap.S().Infof("Skipping attachment %s since it only has props", att.Name)
			continue
		}

		f, err := att.File.Open()
		if err != nil {
			zap.S().Warn("Couldn't open attachment zip file", err)
			continue
		}

		var userID *int
		if params.Requestor != nil {
			userID = &params.Requestor.ID
		}

		if err := base.CreateProblemAttachment(ctx, &kilonova.Attachment{
			Name:    att.Name,
			Private: att.Private,
			Visible: att.Visible,
			Exec:    att.Exec,
		}, pb.ID, f, userID); err != nil {
			zap.S().Warn("Couldn't create attachment", err)
			f.Close()
			continue
		}
		f.Close()
	}

	return nil
}
