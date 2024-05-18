package grader

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/box"
	"github.com/KiloProjects/kilonova/eval/checkers"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	True            = true
	skippedVerdict  = "translate:skipped"
	acceptedVerdict = "test_verdict.accepted"
)

func genSubCompileRequest(ctx context.Context, base *sudoapi.BaseAPI, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (*tasks.CompileRequest, *kilonova.StatusError) {
	req := &tasks.CompileRequest{
		ID:          sub.ID,
		Lang:        sub.Language,
		CodeFiles:   make(map[string][]byte),
		HeaderFiles: make(map[string][]byte),
	}
	atts, err := base.ProblemAttachments(ctx, pb.ID)
	if err != nil {
		return nil, err
	}
	for _, codeFile := range settings.GraderFiles {
		lang := eval.GetLangByFilename(codeFile)
		if lang != sub.Language && !slices.Contains(eval.Langs[sub.Language].SimilarLangs, lang) {
			continue
		}
		for _, att := range atts {
			if att.Name == codeFile {
				data, err := base.AttachmentData(ctx, att.ID)
				if err != nil {
					zap.S().Warn("Couldn't get attachment data:", err)
					return nil, kilonova.Statusf(500, "Couldn't get grader data")
				}
				name := strings.Replace(path.Base(att.Name), path.Ext(att.Name), eval.Langs[lang].Extensions[0], 1)
				req.CodeFiles[path.Join("/box", name)] = data
			}
		}
	}
	subCode, err := base.RawSubmissionCode(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	if len(settings.GraderFiles) > 0 && sub.Language == "pascal" {
		// In interactive problems, include the source code as header
		// Apparently the fpc compiler allows only one file as parameter, this should solve it
		req.HeaderFiles[eval.Langs[sub.Language].SourceName] = subCode
	} else {
		// But by default it should be a code file
		req.CodeFiles[eval.Langs[sub.Language].SourceName] = subCode
	}
	for _, headerFile := range settings.HeaderFiles {
		for _, att := range atts {
			if att.Name == headerFile {
				data, err := base.AttachmentData(ctx, att.ID)
				if err != nil {
					zap.S().Warn("Couldn't get attachment data:", err)
					return nil, kilonova.Statusf(500, "Couldn't get grader data")
				}
				req.HeaderFiles[path.Join("/box", path.Base(att.Name))] = data
			}
		}
	}
	return req, nil
}

func executeSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission) error {
	graderLogger.Info("Executing submission", slog.Int("id", sub.ID), slog.Any("status", sub.Status))
	defer func() {
		// In case anything ever happens, make sure it is at least marked as finished
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished}); err != nil {
			zap.S().Warn("Couldn't finish submission:", err)
		}
	}()

	defer func() {
		err := markSubtestsDone(ctx, base, sub)
		if err != nil {
			zap.S().Warn("Couldn't clean up subtests:", err)
		}
	}()

	problem, err1 := base.Problem(ctx, sub.ProblemID)
	if err1 != nil {
		return kilonova.WrapError(err1, "Couldn't get submission problem")
	}

	problemSettings, err1 := base.ProblemSettings(ctx, sub.ProblemID)
	if err1 != nil {
		return kilonova.WrapError(err1, "Couldn't get problem settings")
	}

	if err := compileSubmission(ctx, base, runner, sub, problem, problemSettings); err != nil {
		if err.Code != 204 { // Skip
			zap.S().Warn(err)
			return err
		}
		return nil
	}

	checker, err := getAppropriateChecker(ctx, base, runner, sub, problem, problemSettings)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't get checker")
	}

	if info, err := checker.Prepare(ctx); err != nil {
		t := true
		info = "Checker compile error:\n" + info
		internalErr := "test_verdict.internal_error"
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &problem.DefaultPoints,
			CompileError: &t, CompileMessage: &info,
			ChangeVerdict: true, ICPCVerdict: &internalErr,
		}); err != nil {
			return kilonova.WrapError(err, "Error during update of compile information")
		}
		return kilonova.WrapError(err, "Could not prepare checker")
	}

	subTests, err1 := base.SubTests(ctx, sub.ID)
	if err1 != nil {
		internalErr := "test_verdict.internal_error"
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &problem.DefaultPoints,
			ChangeVerdict: true, ICPCVerdict: &internalErr,
		}); err != nil {
			return kilonova.WrapError(err, "Could not update submission after subtest fetch fail")
		}
		return kilonova.WrapError(err1, "Could not fetch subtests")
	}

	// TODO: This is shit.
	// It is basically 2 implementations for ~ the same thing. It could be merged neater
	switch sub.SubmissionType {
	case kilonova.EvalTypeClassic:
		if err := handleClassicSubmission(ctx, base, runner, sub, problem, checker, subTests); err != nil {
			zap.S().Warn(err)
			return err
		}
	case kilonova.EvalTypeICPC:
		if err := handleICPCSubmission(ctx, base, runner, sub, problem, checker, subTests); err != nil {
			zap.S().Warn(err)
			return err
		}
	default:
		return kilonova.Statusf(500, "Invalid eval type")
	}

	if err := datastore.GetBucket(datastore.BucketTypeCompiles).RemoveFile(fmt.Sprintf("%d.bin", sub.ID)); err != nil {
		zap.S().Warn("Couldn't remove compilation artifact: ", err)
	}

	if err := checker.Cleanup(ctx); err != nil {
		zap.S().Warn("Couldn't remove checker artifact: ", err)
	}
	return nil
}

func handleClassicSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, problem *kilonova.Problem, checker checkers.Checker, subTests []*kilonova.SubTest) *kilonova.StatusError {
	var wg sync.WaitGroup

	for _, subTest := range subTests {
		subTest := subTest
		wg.Add(1)

		go func() {
			defer wg.Done()
			_, _, err := handleSubTest(ctx, base, runner, checker, sub, problem, subTest)
			if err != nil {
				zap.S().Warn("Error handling subtest:", err)
			}
		}()
	}

	wg.Wait()

	if err := scoreTests(ctx, base, sub, problem); err != nil {
		zap.S().Warn("Couldn't score test: ", err)
	}

	return nil
}

func handleICPCSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, problem *kilonova.Problem, checker checkers.Checker, subTests []*kilonova.SubTest) *kilonova.StatusError {
	var failed bool
	var upd kilonova.SubmissionUpdate
	upd.Status = kilonova.StatusFinished

	for _, subTest := range subTests {
		if failed {
			if err := base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{
				Done: &True, Skipped: &True,
				Verdict: &skippedVerdict,
			}); err != nil {
				zap.S().Warn("Couldn't update skipped subtest:", err)
			}
			continue
		}
		score, verdict, err := handleSubTest(ctx, base, runner, checker, sub, problem, subTest)
		if err != nil {
			zap.S().Warn("Error handling subtest:", err)
			continue
		}
		if !score.Equal(decimal.NewFromInt(100)) {
			upd.Score = &problem.DefaultPoints
			upd.ChangeVerdict = true

			verdict = fmt.Sprintf("%s (test_verdict.test_x #%d)", strings.ReplaceAll(verdict, "translate:", "test_verdict."), subTest.VisibleID)
			upd.ICPCVerdict = &verdict

			failed = true
		}
	}

	if !failed {
		hundred := decimal.NewFromInt(100)
		upd.Score = &hundred
		upd.ChangeVerdict = true
		upd.ICPCVerdict = &acceptedVerdict
	}

	subTests, err := base.SubTests(ctx, sub.ID)
	if err != nil {
		zap.S().Warn("Could not get subtests for max score/mem updating:", err)
		return err
	}

	var memory int
	var time float64
	for _, subtest := range subTests {
		memory = max(memory, subtest.Memory)
		time = max(time, subtest.Time)
	}
	upd.MaxTime = &time
	upd.MaxMemory = &memory

	return base.UpdateSubmission(ctx, sub.ID, upd)
}

func compileSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, problem *kilonova.Problem, problemSettings *kilonova.ProblemEvalSettings) *kilonova.StatusError {
	req, err := genSubCompileRequest(ctx, base, sub, problem, problemSettings)
	if err != nil {
		zap.S().Warn(err)
		return kilonova.WrapError(err, "Couldn't generate compilation request")
	}

	resp, err1 := tasks.CompileTask(ctx, runner, req, graderLogger)
	if err1 != nil {
		return kilonova.WrapError(err1, "Error from eval")
	}
	// if !resp.Success && resp.Other != "" {
	// 	// zap.S().Warnf("Internal grader error during compilation (#%d): %s", sub.ID, resp.Other)
	// 	// resp.Output += "\nGrader notes: " + resp.Other
	// }

	var compileTime *float64
	if resp.Stats != nil {
		compileTime = &resp.Stats.Time
	}
	compileError := !resp.Success
	if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
		CompileError: &compileError, CompileMessage: &resp.Output, CompileTime: compileTime,
	}); err != nil {
		spew.Dump(err)
		return kilonova.WrapError(err, "Couldn't update submission")
	}

	if !resp.Success {
		compileErrVerdict := "test_verdict.compile_error"
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &problem.DefaultPoints,
			ChangeVerdict: true, ICPCVerdict: &compileErrVerdict,
		}); err != nil {
			return kilonova.WrapError(err, "Couldn't finalize submission with compiler error")
		}
		stks, err := base.SubmissionSubTasks(ctx, sub.ID)
		if err != nil {
			return kilonova.WrapError(err, "Couldn't get submission subtasks")
		}
		for _, stk := range stks {
			if err := base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, decimal.Zero); err != nil {
				return kilonova.WrapError(err, "Couldn't finish subtasks")
			}
		}
		return kilonova.Statusf(204, "Compile failed")
	}
	return nil
}

func handleSubTest(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, checker checkers.Checker, sub *kilonova.Submission, problem *kilonova.Problem, subTest *kilonova.SubTest) (decimal.Decimal, string, error) {
	if subTest.TestID == nil {
		zap.S().Error("A subtest whose test was purged was detected.", spew.Sdump(subTest))
		return decimal.Zero, "", kilonova.Statusf(400, "Trying to handle subtest whose test was purged. This should never happen")
	}

	execRequest := &tasks.ExecRequest{
		SubID:       sub.ID,
		SubtestID:   subTest.ID,
		Filename:    problem.TestName,
		MemoryLimit: problem.MemoryLimit,
		TimeLimit:   problem.TimeLimit,
		Lang:        sub.Language,
		TestID:      *subTest.TestID,
	}
	if problem.ConsoleInput {
		execRequest.Filename = "stdin"
	}

	resp, err := tasks.ExecuteTask(ctx, runner, int64(problem.MemoryLimit), execRequest, graderLogger)
	if err != nil {
		return decimal.Zero, "", kilonova.WrapError(err, "Couldn't execute subtest")
	}
	var testScore decimal.Decimal

	// Make sure TLEs are fully handled
	if resp.Time > problem.TimeLimit {
		resp.Time = problem.TimeLimit
		resp.Comments = "translate:timeout"
	}

	if resp.Comments == "" {
		resp.Comments, testScore = checker.RunChecker(ctx, subTest.ID, *subTest.TestID)
	}

	// Hide fatal signals for ICPC submissions
	if sub.SubmissionType == kilonova.EvalTypeICPC {
		if strings.Contains(resp.Comments, "signal 9") {
			resp.Comments = "translate:memory_limit"
		}
		if strings.Contains(resp.Comments, "Caught fatal signal") || strings.Contains(resp.Comments, "Exited with error status") {
			resp.Comments = "translate:runtime_error"
		}
	}

	if err := base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{Memory: &resp.Memory, Percentage: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
		return decimal.Zero, "", kilonova.WrapError(err, "Error during evaltest updating")
	}
	return testScore, resp.Comments, nil
}

func markSubtestsDone(ctx context.Context, base *sudoapi.BaseAPI, sub *kilonova.Submission) error {
	sts, err := base.SubTests(ctx, sub.ID)
	if err != nil {
		return kilonova.WrapError(err, "Error during getting subtests")
	}
	for _, st := range sts {
		if st.Done {
			continue
		}
		if err := base.UpdateSubTest(ctx, st.ID, kilonova.SubTestUpdate{Done: &True}); err != nil {
			zap.S().Warnf("Couldn't mark subtest %d done: %s", st.ID, err)
		}
	}
	return nil
}

func scoreTests(ctx context.Context, base *sudoapi.BaseAPI, sub *kilonova.Submission, problem *kilonova.Problem) *kilonova.StatusError {
	subtests, err1 := base.SubTests(ctx, sub.ID)
	if err1 != nil {
		return err1
	}

	subTasks, err1 := base.SubmissionSubTasks(ctx, sub.ID)
	if err1 != nil {
		return err1
	}
	if len(subTasks) == 0 {
		subTasks = nil
	}

	var score = problem.DefaultPoints

	if len(subTasks) > 0 {
		subMap := make(map[int]*kilonova.SubTest)
		for _, st := range subtests {
			subMap[st.ID] = st
		}
		for _, stk := range subTasks {
			percentage := decimal.NewFromInt(100)
			if len(stk.Subtests) == 0 { // Empty subtasks should be invalidated
				percentage = decimal.Zero
			}
			for _, id := range stk.Subtests {
				st, ok := subMap[id]
				if !ok {
					zap.S().Warn("Couldn't find subtest. This should not really happen.")
					continue
				}
				percentage = decimal.Min(percentage, st.Percentage)
			}
			// subTaskScore = stk.Score * (percentage / 100) rounded to the precision
			subTaskScore := stk.Score.Mul(percentage.Shift(-2)).Round(problem.ScorePrecision)
			score = score.Add(subTaskScore)
			if err := base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, percentage); err != nil {
				zap.S().Warn(err)
			}
		}
	} else {
		for _, subtest := range subtests {
			// testScore = subtest.Score * (subtest.Percentage / 100) rounded to the precision
			testScore := subtest.Score.Mul(subtest.Percentage.Shift(-2)).Round(problem.ScorePrecision)
			score = score.Add(testScore)
		}
	}

	var memory int
	var time float64
	for _, subtest := range subtests {
		memory = max(memory, subtest.Memory)
		time = max(time, subtest.Time)
	}

	return base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, MaxTime: &time, MaxMemory: &memory})
}

var ForceSecureSandbox = config.GenFlag[bool]("feature.grader.force_secure_sandbox", true, "Force use of secure sandbox only. Should be always enabled in production environments")

func getAppropriateRunner() (eval.BoxScheduler, error) {
	var boxFunc scheduler.BoxFunc
	var boxVersion string = "NONE"
	if scheduler.CheckCanRun(box.New) {
		boxFunc = box.New
		boxVersion = box.IsolateVersion()
	} else if scheduler.CheckCanRun(box.NewStupid) && !ForceSecureSandbox.Value() {
		zap.S().Warn("Secure sandbox not found. Using stupid sandbox")
		boxFunc = box.NewStupid
		boxVersion = "stupid"
	}
	if boxFunc == nil {
		zap.S().Fatal("Remote grader has not been implemented. No grader available!")
	}

	zap.S().Info("Trying to spin up local grader")
	bm, err := scheduler.New(config.Eval.StartingBox, config.Eval.NumConcurrent, config.Eval.GlobalMaxMem, graderLogger, boxFunc)
	if err != nil {
		return nil, err
	}
	zap.S().Infof("Running local grader (version: %s)", boxVersion)

	return bm, nil
}

func getAppropriateChecker(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (checkers.Checker, error) {
	if settings.CheckerName == "" {
		return &checkers.DiffChecker{}, nil
	}
	att, err := base.ProblemAttByName(ctx, pb.ID, settings.CheckerName)
	if err != nil {
		return nil, kilonova.WrapError(err, "Couldn't get problem checker metadata")
	}
	data, err := base.ProblemAttDataByName(ctx, pb.ID, settings.CheckerName)
	if err != nil {
		return nil, kilonova.WrapError(err, "Couldn't get problem checker code")
	}
	subCode, err := base.RawSubmissionCode(ctx, sub.ID)
	if err != nil {
		return nil, kilonova.WrapError(err, "Couldn't get submission source code")
	}
	if settings.LegacyChecker {
		return checkers.NewLegacyCustomChecker(runner, graderLogger, pb, settings.CheckerName, data, subCode, att.LastUpdatedAt), nil
	}
	return checkers.NewStandardCustomChecker(runner, graderLogger, pb, settings.CheckerName, data, subCode, att.LastUpdatedAt), nil
}
