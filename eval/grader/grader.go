package grader

import (
	"context"
	"math"
	"path"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/boxmanager"
	"github.com/KiloProjects/kilonova/eval/checkers"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

var True = true

func genSubCompileRequest(ctx context.Context, base *sudoapi.BaseAPI, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (*eval.CompileRequest, *kilonova.StatusError) {
	req := &eval.CompileRequest{
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
		for _, att := range atts {
			if att.Name == codeFile {
				data, err := base.AttachmentData(ctx, att.ID)
				if err != nil {
					zap.S().Warn("Couldn't get attachment data:", err)
					return nil, kilonova.Statusf(500, "Couldn't get grader data")
				}
				req.CodeFiles[path.Join("/box", path.Base(att.Name))] = data
			}
		}
	}
	req.CodeFiles[eval.Langs[sub.Language].SourceName] = []byte(sub.Code)
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
	graderLogger.Infof("Executing submission %d with status %q", sub.ID, sub.Status)
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
		} else {
			return nil
		}
	}

	checker, err := getAppropriateChecker(ctx, base, runner, sub, problem, problemSettings)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't get checker")
	}

	if info, err := checker.Prepare(ctx); err != nil {
		t := true
		info = "Checker compile error:\n" + info
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints, CompileError: &t, CompileMessage: &info}); err != nil {
			return kilonova.WrapError(err, "Error during update of compile information")
		}
		return kilonova.WrapError(err, "Could not prepare checker")
	}

	subTests, err1 := base.SubTests(ctx, sub.ID)
	if err1 != nil {
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err != nil {
			return kilonova.WrapError(err, "Could not update submission after subtest fetch fail")
		}
		return kilonova.WrapError(err1, "Could not fetch subtests")
	}

	var wg sync.WaitGroup

	for _, subTest := range subTests {
		subTest := subTest
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := handleSubTest(ctx, base, runner, checker, sub, problem, subTest)
			if err != nil {
				zap.S().Warn("Error handling subtest:", err)
			}
		}()
	}

	wg.Wait()

	if err := scoreTests(ctx, base, sub, problem); err != nil {
		zap.S().Warn("Couldn't score test: ", err)
	}

	if err := eval.CleanCompilation(sub.ID); err != nil {
		zap.S().Warn("Couldn't remove compilation artifact: ", err)
	}

	if err := checker.Cleanup(ctx); err != nil {
		zap.S().Warn("Couldn't remove checker artifact: ", err)
	}
	return nil
}

func compileSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, problem *kilonova.Problem, problemSettings *kilonova.ProblemEvalSettings) *kilonova.StatusError {
	req, err := genSubCompileRequest(ctx, base, sub, problem, problemSettings)
	if err != nil {
		zap.S().Warn(err)
		return kilonova.WrapError(err, "Couldn't generate compilation request")
	}

	resp, err1 := eval.RunTask(ctx, runner, 0, req, tasks.GetCompileTask(graderLogger))
	if err1 != nil {
		return kilonova.WrapError(err1, "Error from eval")
	}
	// if !resp.Success && resp.Other != "" {
	// 	// zap.S().Warnf("Internal grader error during compilation (#%d): %s", sub.ID, resp.Other)
	// 	// resp.Output += "\nGrader notes: " + resp.Other
	// }

	compileError := !resp.Success
	if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{CompileError: &compileError, CompileMessage: &resp.Output}); err != nil {
		spew.Dump(err)
		return kilonova.WrapError(err, "Couldn't update submission")
	}

	if !resp.Success {
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err != nil {
			return kilonova.WrapError(err, "Couldn't finalize submission with compiler error")
		}
		stks, err := base.SubmissionSubTasks(ctx, sub.ID)
		if err != nil {
			return kilonova.WrapError(err, "Couldn't get submission subtasks")
		}
		for _, stk := range stks {
			if err := base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, 0); err != nil {
				return kilonova.WrapError(err, "Couldn't finish subtasks")
			}
		}
		return kilonova.Statusf(204, "Compile failed")
	}
	return nil
}

func handleSubTest(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, checker eval.Checker, sub *kilonova.Submission, problem *kilonova.Problem, subTest *kilonova.SubTest) error {
	pbTest, err1 := base.TestByID(ctx, subTest.TestID)
	if err1 != nil {
		return kilonova.WrapError(err1, "Couldn't get test")
	}

	execRequest := &eval.ExecRequest{
		SubID:       sub.ID,
		SubtestID:   subTest.ID,
		TestID:      pbTest.ID,
		Filename:    problem.TestName,
		MemoryLimit: problem.MemoryLimit,
		TimeLimit:   problem.TimeLimit,
		Lang:        sub.Language,
	}
	if problem.ConsoleInput {
		execRequest.Filename = "stdin"
	}

	resp, err := eval.RunTask(ctx, runner, int64(problem.MemoryLimit), execRequest, tasks.GetExecuteTask(graderLogger, base))
	if err != nil {
		return kilonova.WrapError(err, "Couldn't execute test")
	}
	var testScore int

	// Make sure TLEs are fully handled
	if resp.Time > problem.TimeLimit {
		resp.Time = problem.TimeLimit
		resp.Comments = "translate:timeout"
	}

	if resp.Comments == "" {
		var skipped bool
		tin, err := base.TestInput(pbTest.ID)
		if err != nil {
			resp.Comments = "translate:internal_error"
			skipped = true
		}
		defer tin.Close()
		tout, err := base.TestOutput(pbTest.ID)
		if err != nil {
			resp.Comments = "translate:internal_error"
			skipped = true
		}
		defer tout.Close()
		sout, err := base.SubtestReader(subTest.ID)
		if err != nil {
			resp.Comments = "translate:internal_error"
			skipped = true
		}
		defer sout.Close()

		if !skipped {
			resp.Comments, testScore = checker.RunChecker(ctx, sout, tin, tout)
		}
	}

	if err := base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{Memory: &resp.Memory, Score: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
		return kilonova.WrapError(err, "Error during evaltest updating")
	}
	return nil
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
			percentage := 100
			if len(stk.Subtests) == 0 { // Empty subtasks should be invalidated
				percentage = 0
			}
			for _, id := range stk.Subtests {
				st, ok := subMap[id]
				if !ok {
					zap.S().Warn("Couldn't find subtest. This should not really happen.")
					continue
				}
				if st.Score < percentage {
					percentage = st.Score
				}
			}
			score += int(math.Round(float64(stk.Score) * float64(percentage) / 100.0))
			if err := base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, percentage); err != nil {
				zap.S().Warn(err)
			}
		}
	} else {
		for _, subtest := range subtests {
			score += int(math.Round(float64(subtest.MaxScore) * float64(subtest.Score) / 100.0))
		}
	}

	var memory int
	var time float64
	for _, subtest := range subtests {
		if subtest.Memory > memory {
			memory = subtest.Memory
		}
		if subtest.Time > time {
			time = subtest.Time
		}
	}

	return base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, MaxTime: &time, MaxMemory: &memory})
}

func getLocalRunner(base *sudoapi.BaseAPI) (eval.BoxScheduler, error) {
	zap.S().Info("Trying to spin up local grader")
	bm, err := boxmanager.New(config.Eval.StartingBox, config.Eval.NumConcurrent, config.Eval.GlobalMaxMem, base)
	if err != nil {
		return nil, err
	}
	zap.S().Info("Running local grader")
	return bm, nil
}

func getAppropriateRunner(base *sudoapi.BaseAPI) (eval.BoxScheduler, error) {
	if boxmanager.CheckCanRun() {
		runner, err := getLocalRunner(base)
		if err == nil {
			return runner, nil
		}
	}

	zap.S().Fatal("Remote grader has not been implemented. No grader available!")
	return nil, nil
	// zap.S().Fatal("Remote grader has been disabled because it can't run problems with custom checker")
	// return nil, nil
}

func getAppropriateChecker(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (eval.Checker, error) {
	if settings.CheckerName == "" {
		return &checkers.DiffChecker{}, nil
	} else {
		data, err := base.AttachmentDataByName(ctx, pb.ID, settings.CheckerName)
		if err != nil {
			return nil, kilonova.WrapError(err, "Couldn't get problem checker")
		}
		if settings.LegacyChecker {
			return checkers.NewLegacyCustomChecker(runner, graderLogger, pb, sub, settings.CheckerName, data), nil
		}
		return checkers.NewStandardCustomChecker(runner, graderLogger, pb, sub, settings.CheckerName, data), nil
	}
}
