package test

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/shopspring/decimal"
)

var (
	ErrBadTestFile = kilonova.Statusf(400, "Bad test score file")
	ErrBadArchive  = kilonova.Statusf(400, "Bad archive")
)

type ArchiveCtx struct {
	fs fs.FS

	tests       map[string]archiveTest
	attachments map[string]archiveAttachment
	props       *properties

	submissions []*submissionStub

	params *TestProcessParams

	scoreParameters []ScoreParamEntry

	testScores ScoreFileEntries

	ctx context.Context
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

	TaskType kilonova.TaskType

	CommunicationProcesses *int
}

func NewArchiveCtx(ctx context.Context, params *TestProcessParams, filesystem fs.FS) *ArchiveCtx {
	return &ArchiveCtx{
		fs:          filesystem,
		tests:       make(map[string]archiveTest),
		attachments: make(map[string]archiveAttachment),
		testScores:  make(ScoreFileEntries),

		params: params,

		ctx: ctx,
	}
}

var (
	testInputSuffixes  = []string{".in", ".input"}
	testOutputSuffixes = []string{".out", ".output", ".ok", ".sol", ".a", ".ans"}
)

func ProcessArchiveFile(ctx *ArchiveCtx, fpath string, base *sudoapi.BaseAPI) error {
	if strings.Contains(fpath, "__MACOSX") || strings.Contains(fpath, ".DS_Store") { // Support archives from MacOS
		return nil
	}
	if slices.Contains(strings.Split(path.Dir(fpath), "/"), "attachments") { // Is in "attachments" directory
		return ProcessAttachmentFile(ctx, fpath)
	}

	if slices.Contains(strings.Split(path.Dir(fpath), "/"), "submissions") { // Is in "submissions" directory
		r, err := ctx.fs.Open(fpath)
		if err != nil {
			return fmt.Errorf("could not open submission file: %w", err)
		}
		defer r.Close()
		return ProcessSubmissionFile(ctx, fpath, r, base)
	}

	ext := strings.ToLower(path.Ext(fpath))
	if ext == ".txt" { // test score file
		// if using score parameters, test score file is redundant
		if len(ctx.scoreParameters) > 0 {
			return kilonova.Statusf(400, "Archive cannot contain tests.txt if you specified score parameters")
		}

		r, err := ctx.fs.Open(fpath)
		if err != nil {
			return fmt.Errorf("could not open problem.xml: %w", err)
		}
		defer r.Close()
		vals, err := ParseScoreFile(ctx.ctx, r)
		if err != nil {
			return err
		}

		ctx.testScores = vals
		return nil
	}

	if ext == ".properties" { // test properties file
		r, err := ctx.fs.Open(fpath)
		if err != nil {
			return fmt.Errorf("could not open properties file: %w", err)
		}
		defer r.Close()
		return ProcessPropertiesFile(ctx, r)
	}

	if strings.ToLower(fpath) == "problem.xml" { // Polygon archive format
		r, err := ctx.fs.Open(fpath)
		if err != nil {
			return fmt.Errorf("could not open problem.xml: %w", err)
		}
		defer r.Close()
		return ProcessProblemXMLFile(ctx, r)
	}

	// Polygon-specific handling
	if ctx.params.Polygon {
		if strings.HasPrefix(fpath, "solutions") {
			r, err := ctx.fs.Open(fpath)
			if err != nil {
				return fmt.Errorf("could not open submission file: %w", err)
			}
			defer r.Close()
			return ProcessSubmissionFile(ctx, fpath, r, base)
		}

		if strings.HasPrefix(fpath, "tests") {
			if slices.Contains(testOutputSuffixes, ext) {
				return ProcessTestOutputFile(ctx, fpath)
			}

			return ProcessTestInputFile(ctx, fpath)
		}

		if fpath == "check.cpp" {
			return ProcessPolygonCheckFile(ctx, fpath)
		}

		match, err := path.Match("statements/.pdf/*/problem.pdf", fpath)
		if err != nil {
			slog.WarnContext(ctx.ctx, "Error matching filepath", slog.Any("err", err))
		}
		if err == nil && match {
			return ProcessPolygonPDFStatement(ctx, fpath)
		}

		return nil
	}

	// if nothing else is detected, it should be a test file
	if slices.Contains(testInputSuffixes, ext) || strings.HasPrefix(fpath, "input") { // test input file (ex: 01.in)
		return ProcessTestInputFile(ctx, fpath)
	}

	if slices.Contains(testOutputSuffixes, ext) || strings.HasPrefix(fpath, "output") { // test output file (ex: 01.out/01.ok)
		return ProcessTestOutputFile(ctx, fpath)
	}

	return nil
}

type TestProcessParams struct {
	Requestor *kilonova.UserFull

	ScoreParamsStr string

	Polygon          bool
	MergeAttachments bool

	// It's used when importing problems, since they use a stub name and, if not set, should be updated anyway, if available.
	// Also when uploading ICPC archive decide if importing as ICPC or not
	FirstImport bool

	// MergeTests bool
}

func ProcessTestArchive(ctx context.Context, pb *kilonova.Problem, ar fs.FS, base *sudoapi.BaseAPI, params *TestProcessParams) error {
	if params.Requestor == nil {
		return kilonova.Statusf(400, "There must be a requestor")
	}

	aCtx := NewArchiveCtx(ctx, params, ar)

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

	if err := fs.WalkDir(ar, ".", func(path string, d fs.DirEntry, err error) error {
		// Skip directory entries
		if d.IsDir() {
			return nil
		}

		return ProcessArchiveFile(aCtx, path, base)
	}); err != nil {
		return err
	}

	if aCtx.props != nil && aCtx.props.Subtasks != nil && len(aCtx.props.SubtaskedTests) != len(aCtx.tests) {
		slog.InfoContext(ctx,
			"Mismatched tests and subtasked tests",
			slog.Int("tests", len(aCtx.tests)),
			slog.Int("subtaskedTests", len(aCtx.props.SubtaskedTests)),
		)
		return kilonova.Statusf(400, "Mismatched number of tests in archive and tests that correspond to at least one subtask")
	}

	for k, v := range aCtx.tests {
		if v.InFilePath == "" || v.OutFilePath == "" {
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
			// slog.InfoContext(ctx, "Automatically inserting scores...")
			var n decimal.Decimal
			totalScore := decimal.NewFromInt(100)
			for _, test := range tests {
				if test.Score.IsPositive() {
					totalScore = totalScore.Sub(test.Score)
				} else {
					n = n.Add(decimal.NewFromInt(1))
				}
			}

			// TODO: Round up and make a toSubtract instead of toAdd
			// We'd like test scores to be in ascending order instead of descending
			// Also, totalScore should also subtract default points when autofilling

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
			slog.WarnContext(ctx, "Couldn't delete tests", slog.Any("err", err))
			return err
		}

		createdTests := map[int]kilonova.Test{}

		for _, v := range tests {
			var test kilonova.Test
			test.ProblemID = pb.ID
			test.VisibleID = v.VisibleID
			test.Score = v.Score
			if err := base.CreateTest(ctx, &test); err != nil {
				slog.WarnContext(ctx, "Couldn't create test", slog.Any("err", err))
				return err
			}

			createdTests[v.VisibleID] = test

			f, err := aCtx.fs.Open(v.InFilePath)
			if err != nil {
				return fmt.Errorf("couldn't open() input file: %w", err)
			}
			if err := base.SaveTestInput(test.ID, f); err != nil {
				slog.WarnContext(ctx, "Couldn't save test input", slog.Any("err", err))
				f.Close()
				return fmt.Errorf("couldn't create test input: %w", err)
			}
			f.Close()
			f, err = aCtx.fs.Open(v.OutFilePath)
			if err != nil {
				return fmt.Errorf("couldn't open() output file: %w", err)
			}
			if err := base.SaveTestOutput(test.ID, f); err != nil {
				slog.WarnContext(ctx, "Couldn't save test output", slog.Any("err", err))
				f.Close()
				return fmt.Errorf("couldn't create test output: %w", err)
			}
			f.Close()
		}

		if err := base.DeleteSubTasks(ctx, pb.ID); err != nil {
			slog.WarnContext(ctx, "Couldn't delete subtasks", slog.Any("err", err))
			return fmt.Errorf("couldn't delete existing subtasks: %w", err)
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
								slog.WarnContext(ctx, "Created test not found anymore", slog.Int("id", tests[i].VisibleID))
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
								slog.WarnContext(ctx, "Created test not found anymore", slog.Int("id", test.VisibleID))
								continue
							}
							testIDs = append(testIDs, test.ID)
						}
					}
				} else {
					slog.WarnContext(ctx, "Somehow score param doesn't have neither count nor match non-nil")
				}
				if len(testIDs) > 0 {
					// Tests are found, create subtask
					if err := base.CreateSubTask(ctx, &kilonova.SubTask{
						ProblemID: pb.ID,
						VisibleID: i + 1,
						Score:     entry.Score,
						Tests:     testIDs,
					}); err != nil {
						slog.WarnContext(ctx, "Couldn't create subtask", slog.Any("err", err))
						return fmt.Errorf("couldn't create subtask: %w", err)
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
					slog.WarnContext(ctx, "Couldn't create subtask", slog.Any("err", err))
					return fmt.Errorf("couldn't create subtask: %w", err)
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
			upd.MemoryLimit, shouldUpd = aCtx.props.MemoryLimit, true
		}
		if aCtx.props.TimeLimit != nil {
			upd.TimeLimit, shouldUpd = aCtx.props.TimeLimit, true
		}
		if aCtx.props.DefaultPoints != nil {
			upd.DefaultPoints, shouldUpd = aCtx.props.DefaultPoints, true
		}
		if aCtx.props.Source != nil {
			upd.SourceCredits, shouldUpd = aCtx.props.Source, true
		}
		if aCtx.props.ConsoleInput != nil {
			upd.ConsoleInput, shouldUpd = aCtx.props.ConsoleInput, true
		}
		if aCtx.props.ScoringStrategy != kilonova.ScoringTypeNone {
			upd.ScoringStrategy, shouldUpd = aCtx.props.ScoringStrategy, true
		}
		if aCtx.props.TaskType != kilonova.TaskTypeNone {
			upd.TaskType, shouldUpd = aCtx.props.TaskType, true
		}
		if aCtx.props.CommunicationProcesses != nil {
			upd.CommunicationProcesses, shouldUpd = aCtx.props.CommunicationProcesses, true
		}
		if aCtx.props.ScorePrecision != nil {
			upd.ScorePrecision, shouldUpd = aCtx.props.ScorePrecision, true
		}
		if aCtx.props.TestName != nil {
			upd.TestName, shouldUpd = aCtx.props.TestName, true
		}

		if aCtx.props.ProblemName != nil && *aCtx.props.ProblemName != "" {
			upd.Name, shouldUpd = aCtx.props.ProblemName, true
		}

		if params.FirstImport && aCtx.props.TestName == nil {
			newTestName := ""
			if upd.Name != nil {
				newTestName = kilonova.MakeSlug(*upd.Name)
			}

			// TODO: More heuristics?

			if len(newTestName) > 0 {
				upd.TestName, shouldUpd = &newTestName, true
			}
		}

		if shouldUpd {
			if err := base.UpdateProblem(ctx, pb.ID, upd, nil); err != nil {
				slog.WarnContext(ctx, "Couldn't update problem", slog.Any("err", err))
				return fmt.Errorf("couldn't update problem medatada: %w", err)
			}
		}

		if len(aCtx.props.Tags) > 0 {
			realTagIDs := []int{}
			for _, mTag := range aCtx.props.Tags {
				tag, err := base.TagByLooseName(ctx, mTag.Name)
				if err != nil || tag == nil {
					id, err := base.CreateTag(ctx, mTag.Name, mTag.Type)
					if err != nil {
						slog.WarnContext(ctx, "Couldn't create tag", slog.Any("err", err))
						continue
					}
					realTagIDs = append(realTagIDs, id)
					continue
				}
				realTagIDs = append(realTagIDs, tag.ID)
			}
			if err := base.UpdateProblemTags(ctx, pb.ID, realTagIDs); err != nil {
				slog.WarnContext(ctx, "Couldn't update problem tags", slog.Any("err", err))
				return fmt.Errorf("couldn't update tags: %w", err)
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
					slog.WarnContext(ctx, "Couldn't update problem editors", slog.Any("err", err))
					return err
				}
				for _, ed := range cEditors {
					if err := base.StripProblemAccess(ctx, pb.ID, ed.ID); err != nil {
						slog.WarnContext(ctx, "Couldn't remove problem access", slog.Any("err", err))
					}
				}

				// Lastly, add the new editors
				for _, editor := range newEditors {
					if err := base.AddProblemEditor(ctx, pb.ID, editor.ID); err != nil {
						slog.WarnContext(ctx, "Couldn't add problem editor``", slog.Any("err", err))
					}
				}
			}

		}

	}

	// Do submissions at the end after all changes have been merged
	if len(aCtx.submissions) > 0 {
		for _, sub := range aCtx.submissions {
			lang := base.Language(ctx, sub.lang)
			if lang == nil {
				slog.InfoContext(ctx, "Skipping submission, unknown language")
				continue
			}
			if _, err := base.CreateSubmission(ctx, params.Requestor, pb, sub.code, sub.filename, lang, nil, true); err != nil {
				slog.WarnContext(ctx, "Couldn't create submission", slog.Any("err", err))
			}
		}
	}

	return nil
}

func createAttachments(ctx context.Context, aCtx *ArchiveCtx, pb *kilonova.Problem, base *sudoapi.BaseAPI, params *TestProcessParams) error {
	atts, err := base.ProblemAttachments(ctx, pb.ID)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get problem attachments", slog.Any("err", err))
		return fmt.Errorf("couldn't get attachments: %w", err)
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
			slog.WarnContext(ctx, "Couldn't remove attachments", slog.Any("err", err))
			return fmt.Errorf("couldn't delete attachments: %w", err)
		}
	}
	for _, att := range aCtx.attachments {
		if att.FilePath == "" {
			slog.InfoContext(ctx, "Skipping attachment since it only has props", slog.String("name", att.Name))
			continue
		}

		f, err := aCtx.fs.Open(att.FilePath)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't open attachment zip file", slog.Any("err", err))
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
			slog.WarnContext(ctx, "Couldn't create attachment", slog.Any("err", err))
			f.Close()
			continue
		}
		f.Close()
	}

	return nil
}
