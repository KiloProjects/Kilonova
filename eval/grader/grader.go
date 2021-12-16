package grader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/boxmanager"
	"github.com/KiloProjects/kilonova/eval/checkers"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
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
	dm    kilonova.GraderStore

	debug bool
	db    *db.DB
}

func NewHandler(ctx context.Context, db *db.DB, dm kilonova.DataStore, debug bool) *Handler {
	ch := make(chan eval.GraderSubmission, 5)
	return &Handler{ctx, ch, dm, debug, db}
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			sub, err := h.db.FetchGraderSubmission(h.ctx)
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

			/*
				subs, err := h.db.Submissions(h.ctx, waitingSubs)
				if err != nil {
					log.Println("Error fetching submissions:", err)
					continue
				}
				if subs != nil {
					if config.Common.Debug {
						log.Printf("Found %d submissions\n", len(subs))
					}

					for _, sub := range subs {
						if err := h.db.UpdateSubmission(h.ctx, sub.ID, workingUpdate); err != nil {
							log.Println(err)
							continue
						}
						h.sChan <- sub
					}
				}
			*/
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
	task := &tasks.CompileTask{
		Req:   &eval.CompileRequest{ID: sub.ID, Code: []byte(sub.Code), Lang: sub.Language},
		Debug: h.debug,
	}
	err := runner.RunTask(ctx, task)
	if err != nil {
		return kilonova.WrapError(kilonova.EINTERNAL, "Error from eval", err)
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
		return kilonova.WrapInternal(err)
	}
	// Cleanup compilation at the end
	defer func() {
		if err := eval.CleanCompilation(sub.ID); err != nil {
			log.Printf("Couldn't clean task: %s\n", err)
		}
	}()

	problem, err := h.db.Problem(ctx, sub.ProblemID)
	if err != nil {
		return kilonova.WrapError(kilonova.EINTERNAL, "Couldn't get submission problem", err)
	}

	checker, err := h.getAppropriateChecker(ctx, runner, sub, problem)
	if err != nil {
		return kilonova.WrapError(kilonova.EINTERNAL, "Couldn't get checker:", err)
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
			return kilonova.WrapError(kilonova.EINTERNAL, "Error during update of compile information:", err)
		}
		return kilonova.WrapError(kilonova.EINTERNAL, "Could not prepare checker", err)
	}

	subTests, err := h.db.SubTestsBySubID(ctx, sub.ID)
	if resp.Success == false || err != nil {
		if err := gsub.Update(kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &problem.DefaultPoints}); err != nil {
			return err
		}
		return err
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
	pbTest, err := h.db.TestByID(ctx, subTest.TestID)
	if err != nil {
		return fmt.Errorf("Error during test getting (0.5): %w", err)
	}

	execRequest := &eval.ExecRequest{
		SubID:       sub.ID,
		SubtestID:   subTest.ID,
		TestID:      pbTest.ID,
		Filename:    problem.TestName,
		StackLimit:  problem.StackLimit,
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
		DM:    h.dm,
	}

	err = runner.RunTask(ctx, task)
	if err != nil {
		return fmt.Errorf("Error executing test: %w", err)
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
		tin, err := h.dm.TestInput(pbTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer tin.Close()
		tout, err := h.dm.TestOutput(pbTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer tout.Close()
		sout, err := h.dm.SubtestReader(subTest.ID)
		if err != nil {
			resp.Comments = "Internal grader error"
			skipped = true
		}
		defer sout.Close()

		if !skipped {
			resp.Comments, testScore = checker.RunChecker(ctx, sout, tin, tout)
		}
	}

	if err := h.db.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{Memory: &resp.Memory, Score: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
		return fmt.Errorf("Error during evaltest updating: %w", err)
	}
	return nil
}

func (h *Handler) MarkSubtestsDone(ctx context.Context, sub *kilonova.Submission) error {
	sts, err := h.db.SubTestsBySubID(ctx, sub.ID)
	if err != nil {
		return fmt.Errorf("Error during subtest getting: %w", err)
	}
	for _, st := range sts {
		if st.Done {
			continue
		}
		if err := h.db.UpdateSubTest(ctx, st.ID, kilonova.SubTestUpdate{Done: &True}); err != nil {
			log.Printf("Couldn't mark subtest %d done: %s\n", st.ID, err)
		}
	}
	return nil
}

func (h *Handler) ScoreTests(ctx context.Context, gsub eval.GraderSubmission, problem *kilonova.Problem) error {
	sub := gsub.Submission()
	subtests, err := h.db.SubTestsBySubID(ctx, sub.ID)
	if err != nil {
		return err
	}

	subTasks, err := h.db.SubTasks(ctx, problem.ID)
	if err != nil {
		return err
	}
	if len(subTasks) == 0 {
		subTasks = nil
	}

	var score = problem.DefaultPoints

	if subTasks != nil && len(subTasks) > 0 {
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
			pbTest, err := h.db.TestByID(ctx, subtest.TestID)
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

func (h *Handler) Start() error {
	runner, err := h.getAppropriateRunner()
	if err != nil {
		return err
	}

	go h.chFeeder(4 * time.Second)

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
	bm, err := boxmanager.New(config.Eval.NumConcurrent, h.dm)
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
	zap.S().Fatal("Remote grader has been disabled because it can't run problems with custom checker")
	return nil, nil
	/*
		Disabled until it fully works
		return nil, nil
		log.Println("Could not spin up local grader, trying to contact remote")
		return newGrpcRunner(config.Eval.Address)
	*/
}

func (h *Handler) getProblemChecker(ctx context.Context, pb *kilonova.Problem) (*kilonova.Attachment, error) {
	// TODO: Do not get all attachments data
	atts, err := h.db.Attachments(ctx, true, kilonova.AttachmentFilter{ProblemID: &pb.ID})
	if err != nil || len(atts) == 0 {
		return nil, errors.New("No attachments found")
	}
	for _, att := range atts {
		filename := path.Base(att.Name)
		filename = strings.TrimSuffix(filename, path.Ext(filename))
		if filename == "checker" && eval.GetLangByFilename(att.Name) != "" {
			return att, nil
		}
	}

	return nil, errors.New("No checker found")
}

func (h *Handler) getAppropriateChecker(ctx context.Context, runner eval.Runner, sub *kilonova.Submission, pb *kilonova.Problem) (eval.Checker, error) {
	switch pb.Type {
	case kilonova.ProblemTypeClassic:
		return &checkers.DiffChecker{}, nil
	case kilonova.ProblemTypeCustomChecker:
		att, err := h.getProblemChecker(ctx, pb)
		if err != nil {
			return nil, kilonova.WrapError(kilonova.EINVALID, "Couldn't get problem checker", err)
		}
		return checkers.NewCustomChecker(runner, pb, sub, att)
	default:
		zap.S().Warn("Unknown problem type", pb.Type)
		return nil, &kilonova.Error{Code: kilonova.EINTERNAL, Message: "Unknown problem type"}
	}
}
