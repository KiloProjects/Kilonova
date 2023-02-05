package grader

import (
	"context"
	"math"
	"os"
	"path"
	"sync"
	"time"

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

var (
	True          = true
	waitingSubs   = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting, Ascending: true, Limit: 20}
	workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}
)

type Handler struct {
	ctx   context.Context
	sChan chan *kilonova.Submission
	base  *sudoapi.BaseAPI

	logFile     *os.File
	localLogger *zap.SugaredLogger
}

func NewHandler(ctx context.Context, base *sudoapi.BaseAPI) (*Handler, *kilonova.StatusError) {
	ch := make(chan *kilonova.Submission, 1)

	logFile, err := os.OpenFile(path.Join(config.Common.LogDir, "grader.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, kilonova.WrapError(err, "Could not open log file")
	}
	localLogger := zap.New(kilonova.GetZapCore(config.Common.Debug, false, logFile), zap.AddCaller())

	return &Handler{ctx, ch, base, logFile, localLogger.Sugar()}, nil
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			subs, err := h.base.RawSubmissions(h.ctx, waitingSubs)
			if err != nil {
				zap.S().Warn(err)
				continue
			}
			if len(subs) > 0 {
				h.localLogger.Infof("Found %d submissions", len(subs))

				for _, sub := range subs {
					sub := sub
					h.sChan <- sub
				}

			}
		case <-h.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (h *Handler) handle(ctx context.Context, runner eval.Runner) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case sub, more := <-h.sChan:
			if !more {
				return nil
			}
			if err := h.base.UpdateSubmission(h.ctx, sub.ID, workingUpdate); err != nil {
				zap.S().Warn(err)
				continue
			}
			if err := h.ExecuteSubmission(ctx, runner, sub); err != nil {
				zap.S().Warn("Couldn't run submission: ", err)
			}
		}
	}
}

func (h *Handler) genSubCompileRequest(ctx context.Context, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (*eval.CompileRequest, *kilonova.StatusError) {
	req := &eval.CompileRequest{
		ID:          sub.ID,
		Lang:        sub.Language,
		CodeFiles:   make(map[string][]byte),
		HeaderFiles: make(map[string][]byte),
	}
	atts, err := h.base.ProblemAttachments(ctx, pb.ID)
	if err != nil {
		return nil, err
	}
	for _, codeFile := range settings.GraderFiles {
		for _, att := range atts {
			if att.Name == codeFile {
				data, err := h.base.AttachmentData(ctx, att.ID)
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
				data, err := h.base.AttachmentData(ctx, att.ID)
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

func (h *Handler) ExecuteSubmission(ctx context.Context, runner eval.Runner, sub *kilonova.Submission) error {
	h.localLogger.Infof("Executing submission %d with status %q", sub.ID, sub.Status)
	defer func() {
		// In case anything ever happens, make sure it is at least marked as finished
		if err := h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished}); err != nil {
			zap.S().Warn("Couldn't finish submission:", err)
		}
	}()

	defer func() {
		err := h.MarkSubtestsDone(ctx, sub)
		if err != nil {
			zap.S().Warn("Couldn't clean up subtests:", err)
		}
	}()

	problem, err1 := h.base.Problem(ctx, sub.ProblemID)
	if err1 != nil {
		return kilonova.WrapError(err1, "Couldn't get submission problem")
	}

	problemSettings, err1 := h.base.ProblemSettings(ctx, sub.ProblemID)
	if err1 != nil {
		return kilonova.WrapError(err1, "Couldn't get problem settings")
	}

	if err := h.CompileSubmission(ctx, runner, sub, problem, problemSettings); err != nil {
		if err.Code != 204 { // Skip
			return err
		} else {
			return nil
		}
	}

	checker, err := h.getAppropriateChecker(ctx, runner, sub, problem, problemSettings)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't get checker")
	}

	if info, err := checker.Prepare(ctx); err != nil {
		t := true
		info = "Checker compile error:\n" + info
		if err := h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints, CompileError: &t, CompileMessage: &info}); err != nil {
			return kilonova.WrapError(err, "Error during update of compile information")
		}
		return kilonova.WrapError(err, "Could not prepare checker")
	}

	subTests, err1 := h.base.SubTests(ctx, sub.ID)
	if err1 != nil {
		if err := h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err != nil {
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
			err := h.HandleSubTest(ctx, runner, checker, sub, problem, subTest)
			if err != nil {
				zap.S().Warn("Error handling subtest:", err)
			}
		}()
	}

	wg.Wait()

	if err := h.ScoreTests(ctx, sub, problem); err != nil {
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

func (h *Handler) CompileSubmission(ctx context.Context, runner eval.Runner, sub *kilonova.Submission, problem *kilonova.Problem, problemSettings *kilonova.ProblemEvalSettings) *kilonova.StatusError {
	req, err := h.genSubCompileRequest(ctx, sub, problem, problemSettings)
	if err != nil {
		zap.S().Warn(err)
		return kilonova.WrapError(err, "Couldn't generate compilation request")
	}

	task := tasks.NewCompileTask(req, h.localLogger)
	if err := runner.RunTask(ctx, task); err != nil {
		return kilonova.WrapError(err, "Error from eval")
	}

	resp := task.Resp
	// if !resp.Success && resp.Other != "" {
	// 	// zap.S().Warnf("Internal grader error during compilation (#%d): %s", sub.ID, resp.Other)
	// 	// resp.Output += "\nGrader notes: " + resp.Other
	// }

	compileError := !resp.Success
	if err := h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{CompileError: &compileError, CompileMessage: &resp.Output}); err != nil {
		spew.Dump(err)
		return kilonova.WrapError(err, "Couldn't update submission")
	}

	if resp.Success == false {
		if err := h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err != nil {
			return kilonova.WrapError(err, "Couldn't finalize submission with compiler error")
		}
		return kilonova.Statusf(204, "Compile failed")
	}
	return nil
}

func (h *Handler) HandleSubTest(ctx context.Context, runner eval.Runner, checker eval.Checker, sub *kilonova.Submission, problem *kilonova.Problem, subTest *kilonova.SubTest) error {
	pbTest, err1 := h.base.TestByID(ctx, subTest.TestID)
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

	task := tasks.NewExecuteTask(execRequest, h.base, h.localLogger)

	if err := runner.RunTask(ctx, task); err != nil {
		return kilonova.WrapError(err, "Couldn't execute test")
	}

	resp := task.Resp
	var testScore int

	// Make sure TLEs are fully handled
	if resp.Time > problem.TimeLimit {
		resp.Time = problem.TimeLimit
		resp.Comments = "TLE"
	}

	if resp.Comments == "" {
		var skipped bool
		tin, err := h.base.TestInput(pbTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer tin.Close()
		tout, err := h.base.TestOutput(pbTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer tout.Close()
		sout, err := h.base.SubtestReader(subTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer sout.Close()

		if !skipped {
			resp.Comments, testScore = checker.RunChecker(ctx, sout, tin, tout)
		}
	}

	if err := h.base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{Memory: &resp.Memory, Score: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
		return kilonova.WrapError(err, "Error during evaltest updating")
	}
	return nil
}

func (h *Handler) MarkSubtestsDone(ctx context.Context, sub *kilonova.Submission) error {
	sts, err := h.base.SubTests(ctx, sub.ID)
	if err != nil {
		return kilonova.WrapError(err, "Error during getting subtests")
	}
	for _, st := range sts {
		if st.Done {
			continue
		}
		if err := h.base.UpdateSubTest(ctx, st.ID, kilonova.SubTestUpdate{Done: &True}); err != nil {
			zap.S().Warnf("Couldn't mark subtest %d done: %s", st.ID, err)
		}
	}
	return nil
}

func (h *Handler) ScoreTests(ctx context.Context, sub *kilonova.Submission, problem *kilonova.Problem) *kilonova.StatusError {
	subtests, err1 := h.base.SubTests(ctx, sub.ID)
	if err1 != nil {
		return err1
	}

	subTasks, err1 := h.base.SubmissionSubTasks(ctx, sub.ID)
	if err1 != nil {
		return err1
	}
	if len(subTasks) == 0 {
		subTasks = nil
	}

	var score = problem.DefaultPoints

	if subTasks != nil && len(subTasks) > 0 {
		h.localLogger.Info("Evaluating by subtasks")
		subMap := make(map[int]*kilonova.SubTest)
		for _, st := range subtests {
			subMap[st.ID] = st
		}
		for _, stk := range subTasks {
			percentage := 100
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
		}
	} else {
		h.localLogger.Info("Evaluating by addition")
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

	return h.base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, MaxTime: &time, MaxMemory: &memory})
}

func (h *Handler) Close() error {
	return h.logFile.Close()
}

func (h *Handler) Start() error {
	runner, err := h.getAppropriateRunner()
	if err != nil {
		return err
	}

	go h.chFeeder(2 * time.Second)

	eCh := make(chan error, 1)
	go func() {
		defer runner.Close(h.ctx)
		zap.S().Info("Connected to eval")

		err := h.handle(h.ctx, runner)
		if err != nil {
			zap.S().Error("Handling error:", zap.Error(err))
		}
		eCh <- err
	}()

	return <-eCh
}

func (h *Handler) getLocalRunner() (eval.Runner, error) {
	zap.S().Info("Trying to spin up local grader")
	bm, err := boxmanager.New(config.Eval.NumConcurrent, h.base)
	if err != nil {
		return nil, err
	}
	zap.S().Info("Running local grader")
	return bm, nil
}

func (h *Handler) getAppropriateRunner() (eval.Runner, error) {
	if boxmanager.CheckCanRun() {
		runner, err := h.getLocalRunner()
		if err == nil {
			return runner, nil
		}
	}

	zap.S().Fatal("Remote grader has not been implemented. No grader available!")
	return nil, nil
	// zap.S().Fatal("Remote grader has been disabled because it can't run problems with custom checker")
	// return nil, nil
	/*
		Disabled until it fully works
		return nil, nil
		zap.S().Info("Could not spin up local grader, trying to contact remote")
		return newGrpcRunner(config.Eval.Address)
	*/
}

func (h *Handler) getAppropriateChecker(ctx context.Context, runner eval.Runner, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (eval.Checker, error) {
	if settings.CheckerName == "" {
		return &checkers.DiffChecker{}, nil
	} else {
		data, err := h.base.AttachmentDataByName(ctx, pb.ID, settings.CheckerName)
		if err != nil {
			return nil, kilonova.WrapError(err, "Couldn't get problem checker")
		}
		if settings.LegacyChecker {
			return checkers.NewLegacyCustomChecker(runner, h.localLogger, pb, sub, settings.CheckerName, data)
		}
		return checkers.NewStandardCustomChecker(runner, h.localLogger, pb, sub, settings.CheckerName, data)
	}
}
