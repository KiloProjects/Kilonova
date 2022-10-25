package grader

import (
	"context"
	"log"
	"math"
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
	waitingSubs   = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting}
	workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}
)

type Handler struct {
	ctx   context.Context
	sChan chan eval.GraderSubmission
	base  *sudoapi.BaseAPI

	debug bool
}

func NewHandler(ctx context.Context, base *sudoapi.BaseAPI) *Handler {
	ch := make(chan eval.GraderSubmission, 5)
	return &Handler{ctx, ch, base, config.Common.Debug}
}

// chFeeder "feeds" sChan with relevant data
func (h *Handler) chFeeder(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			sub, err := h.base.GraderSubmission(h.ctx)
			if err != nil {
				zap.S().Warnf("Error fetching submission: %s", err)
				continue
			}
			if sub == nil {
				continue
			}

			if config.Common.Debug {
				zap.S().Info("Found submission")
			}
			if err := sub.Update(workingUpdate); err != nil {
				sub.Close()
				zap.S().Warn(err)
				continue
			}
			h.sChan <- sub

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
			defer sub.Close()
			if err := h.ExecuteSubmission(ctx, runner, sub); err != nil {
				log.Println("Couldn't run submission:", err)
			}
		}
	}
}

func (h *Handler) genSubCompileRequest(ctx context.Context, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (*eval.CompileRequest, error) {
	req := &eval.CompileRequest{
		ID:          sub.ID,
		Lang:        sub.Language,
		CodeFiles:   make(map[string][]byte),
		HeaderFiles: make(map[string][]byte),
	}

	atts, err := h.base.ProblemAttachments(ctx, pb.ID) // h.base.Attachments(ctx, true, kilonova.AttachmentFilter{ProblemID:&pb.ID})
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

func (h *Handler) ExecuteSubmission(ctx context.Context, runner eval.Runner, gsub eval.GraderSubmission) error {
	sub := gsub.Submission()
	defer func() {
		if err := gsub.Close(); err != nil {
			log.Printf("Could not finish submission: %s\n", err)
		}
	}()

	defer func() {
		err := h.MarkSubtestsDone(ctx, sub)
		if err != nil {
			log.Println("Couldn't clean up subtests:", err)
		}
	}()

	problem, err1 := h.base.Problem(ctx, sub.ProblemID)
	if err1 != nil {
		return sudoapi.WrapError(err1, "Couldn't get submission problem")
	}

	problemSettings, err := h.base.GetProblemSettings(ctx, sub.ProblemID)
	if err != nil {
		return sudoapi.WrapError(err, "Couldn't get problem settings")
	}

	req, err := h.genSubCompileRequest(ctx, sub, problem, problemSettings)
	if err != nil {
		zap.S().Warn(err)
		return kilonova.WrapError(err, "Couldn't generate compilation request")
	}

	task := &tasks.CompileTask{
		Req:   req,
		Debug: h.debug,
	}
	err = runner.RunTask(ctx, task)
	if err != nil {
		return sudoapi.WrapError(err, "Error from eval")
	}

	resp := task.Resp
	if h.debug {
		old := resp.Output
		resp.Output = "<output stripped>"
		spew.Dump(resp)
		resp.Output = old
	}

	compileError := !resp.Success
	if err := gsub.Update(kilonova.SubmissionUpdate{CompileError: &compileError, CompileMessage: &resp.Output}); err != nil {
		return sudoapi.WrapError(err, "Error updating submission")
	}
	// Cleanup compilation at the end
	defer func() {
		if err := eval.CleanCompilation(sub.ID); err != nil {
			log.Printf("Couldn't clean task: %s\n", err)
		}
	}()

	checker, err := h.getAppropriateChecker(ctx, runner, sub, problem, problemSettings)
	if err != nil {
		return sudoapi.WrapError(err, "Couldn't get checker")
	}
	// Defer checker cleanup
	defer func() {
		if err := checker.Cleanup(ctx); err != nil {
			log.Printf("Couldn't clean checker: %s\n", err)
		}
	}()

	if info, err := checker.Prepare(ctx); err != nil {
		t := true
		if err := gsub.Update(kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints, CompileError: &t, CompileMessage: &info}); err != nil {
			return sudoapi.WrapError(err, "Error during update of compile information")
		}
		return sudoapi.WrapError(err, "Could not prepare checker")
	}

	subTests, err1 := h.base.SubTests(ctx, sub.ID)
	if !resp.Success || err1 != nil {
		if err1 := gsub.Update(kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err1 != nil {
			return err1
		}
		return err1
	}

	var wg sync.WaitGroup

	for _, subTest := range subTests {
		subTest := subTest
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := h.HandleSubTest(ctx, runner, checker, sub, problem, subTest)
			if err != nil {
				log.Println("Error handling subTest:", err)
			}
		}()
	}

	wg.Wait()

	if err := h.ScoreTests(ctx, gsub, problem); err != nil {
		log.Printf("Couldn't score test: %s\n", err)
	}

	return nil
}

func (h *Handler) HandleSubTest(ctx context.Context, runner eval.Runner, checker eval.Checker, sub *kilonova.Submission, problem *kilonova.Problem, subTest *kilonova.SubTest) error {
	pbTest, err := h.base.TestByID(ctx, subTest.TestID)
	if err != nil {
		return kilonova.WrapError(err, "Error during test getting (0.5)")
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

	task := &tasks.ExecuteTask{
		Req:   execRequest,
		Resp:  &eval.ExecResponse{},
		Debug: h.debug,
		DM:    h.base,
	}

	if err := runner.RunTask(ctx, task); err != nil {
		return kilonova.WrapError(err, "Error executing test")
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
			log.Printf("Couldn't mark subtest %d done: %s\n", st.ID, err)
		}
	}
	return nil
}

func (h *Handler) ScoreTests(ctx context.Context, gsub eval.GraderSubmission, problem *kilonova.Problem) error {
	sub := gsub.Submission()
	subtests, err := h.base.SubTests(ctx, sub.ID)
	if err != nil {
		return err
	}

	subTasks, err := h.base.SubTasks(ctx, problem.ID)
	if err != nil {
		return err
	}
	if len(subTasks) == 0 {
		subTasks = nil
	}

	var score = problem.DefaultPoints

	if len(subTasks) > 0 {
		if h.debug {
			log.Println("Evaluating by subtasks")
		}
		subMap := make(map[int]*kilonova.SubTest)
		for _, st := range subtests {
			subMap[st.TestID] = st
		}
		for _, stk := range subTasks {
			percentage := 100
			for _, id := range stk.Tests {
				st, ok := subMap[id]
				if !ok {
					log.Printf("Warning: couldn't find a subtest for subtask %d\n", stk.VisibleID)
					continue
				}
				if st.Score < percentage {
					percentage = st.Score
				}
			}
			score += int(math.Round(float64(stk.Score) * float64(percentage) / 100.0))
		}
	} else {
		if h.debug {
			log.Println("Evaluating by addition")
		}
		for _, subtest := range subtests {
			pbTest, err := h.base.TestByID(ctx, subtest.TestID)
			if err != nil {
				log.Println("Couldn't get test (0xasdf):", err)
				continue
			}
			score += int(math.Round(float64(pbTest.Score) * float64(subtest.Score) / 100.0))
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

	return gsub.Update(kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, MaxTime: &time, MaxMemory: &memory})
}

// Start begins running the grader in a blocking way and returns only if an error has occured
func (h *Handler) Start() error {
	runner, err := h.getAppropriateRunner()
	if err != nil {
		return err
	}

	go h.chFeeder(4 * time.Second)

	defer runner.Close(h.ctx)
	zap.S().Info("Connected to eval")

	err = h.handle(h.ctx, runner)
	if err != nil {
		zap.S().Error("Grader handling error:", zap.Error(err))
	}
	return err
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
}

func (h *Handler) getAppropriateChecker(ctx context.Context, runner eval.Runner, sub *kilonova.Submission, pb *kilonova.Problem, settings *kilonova.ProblemEvalSettings) (eval.Checker, error) {
	if settings.CheckerName == "" {
		return &checkers.DiffChecker{}, nil
	} else {
		data, err := h.base.AttachmentDataByName(ctx, pb.ID, settings.CheckerName)
		if err != nil {
			return nil, kilonova.WrapError(err, "Couldn't get problem checker")
		}
		return checkers.NewCustomChecker(runner, pb, sub, settings.CheckerName, data)
	}
}
