package grader

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/boxmanager"
	"github.com/KiloProjects/kilonova/eval/checkers"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/davecgh/go-spew/spew"
)

var (
	True          = true
	waitingSubs   = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting}
	workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}
)

type Handler struct {
	ctx   context.Context
	sChan chan *kilonova.Submission
	kn    *logic.Kilonova
	dm    kilonova.GraderStore

	debug  bool
	sserv  kilonova.SubmissionService
	pserv  kilonova.ProblemService
	stserv kilonova.SubTestService
	tserv  kilonova.TestService
}

func NewHandler(ctx context.Context, kn *logic.Kilonova, db kilonova.TypeServicer) *Handler {
	ch := make(chan *kilonova.Submission, 5)
	return &Handler{ctx, ch, kn, kn.DM, kn.Debug,
		db.SubmissionService(), db.ProblemService(), db.SubTestService(), db.TestService()}
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			subs, err := h.sserv.Submissions(h.ctx, waitingSubs)
			if err != nil {
				log.Println("Error fetching submissions:", err)
				continue
			}
			if subs != nil {
				log.Printf("Found %d submissions\n", len(subs))
				for _, sub := range subs {
					//if err := sub.SetStatus(kilonova.StatusWorking, 0); err != nil {
					if err := h.sserv.UpdateSubmission(h.ctx, sub.ID, workingUpdate); err != nil {
						log.Println(err)
						continue
					}
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

			problem, err := h.pserv.ProblemByID(ctx, sub.ProblemID)
			if err != nil {
				log.Println("Error during submission problem getting:", err)
				continue
			}

			var score_mu sync.Mutex
			var score = problem.DefaultPoints

			task := &tasks.CompileTask{
				Req:   &eval.CompileRequest{ID: sub.ID, Code: []byte(sub.Code), Lang: sub.Language},
				Debug: h.debug,
			}
			err = runner.RunTask(ctx, task)
			if err != nil {
				log.Println("Error from eval:", err)
				continue
			}

			resp := task.Resp
			if h.debug {
				old := resp.Output
				resp.Output = "<output stripped>"
				spew.Dump(resp)
				resp.Output = old
			}

			compileError := !resp.Success
			if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{CompileError: &compileError, CompileMessage: &resp.Output}); err != nil {
				log.Println("Error during update of compile information:", err)
				continue
			}

			checker, err := getAppropriateChecker(runner, sub, problem)
			if err != nil {
				log.Println("Could not get checker:", err)
				continue
			}

			if info, err := checker.Prepare(ctx); err != nil {
				log.Println("Checker prepare error:", err)
				t := true
				if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, CompileError: &t, CompileMessage: &info}); err != nil {
					log.Println("Error during update of compile information:", err)
				}
				continue
			}

			tests, err := h.stserv.SubTestsBySubID(ctx, sub.ID)
			if resp.Success == false || err != nil {
				if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score}); err != nil {
					log.Println(err)
				}
				continue
			}

			var wg sync.WaitGroup

			for _, test := range tests {
				test := test
				wg.Add(1)

				pbTest, err := h.tserv.TestByID(ctx, test.TestID)
				if err != nil {
					log.Println("Error during test getting (0.5):", err)
					continue
				}

				filename := problem.TestName
				if problem.ConsoleInput {
					filename = "stdin"
				}

				execRequest := &eval.ExecRequest{
					SubID:       sub.ID,
					SubtestID:   test.ID,
					TestID:      pbTest.ID,
					Filename:    filename,
					StackLimit:  problem.StackLimit,
					MemoryLimit: problem.MemoryLimit,
					TimeLimit:   problem.TimeLimit,
					Lang:        sub.Language,
				}
				go func() {
					defer wg.Done()
					subTestID := test.ID
					pbTest := pbTest

					task := &tasks.ExecuteTask{
						Req:   execRequest,
						Resp:  &eval.ExecResponse{},
						Debug: h.debug,
						DM:    h.dm,
					}

					err := runner.RunTask(ctx, task)
					if err != nil {
						log.Printf("Error executing test: %v\n", err)
						return
					}

					resp := task.Resp

					/*
						if h.debug {
							spew.Dump(resp)
						}
					*/

					var testScore int

					// Make sure TLEs are fully handled
					if resp.Time > problem.TimeLimit {
						resp.Comments = "TLE"
						testScore = 0
					}

					if resp.Comments == "" {
						tout, err := h.dm.TestOutput(pbTest.ID)
						if err != nil {
							resp.Comments = "Internal grader error"
							score = 0
						}
						sout, err := h.dm.SubtestReader(subTestID)
						if err != nil {
							resp.Comments = "Internal grader error"
							score = 0
						}

						resp.Comments, testScore = checker.RunChecker(ctx, sout, tout, int(pbTest.Score))
					}

					mem := int(resp.Memory)
					if err := h.stserv.UpdateSubTest(ctx, subTestID, kilonova.SubTestUpdate{Memory: &mem, Score: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
						log.Println("Error during evaltest updating:", err)
						return
					}

					score_mu.Lock()
					score += testScore
					score_mu.Unlock()

				}()
			}

			wg.Wait()

			if err := eval.CleanCompilation(sub.ID); err != nil {
				log.Printf("Couldn't clean task: %s\n", err)
			}

			if err := checker.Cleanup(ctx); err != nil {
				log.Printf("Couldn't clean checker: %s\n", err)
			}

			if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score}); err != nil {
				log.Println("Error 1214:", err)
			}
		}
	}
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
		log.Println("Connected to eval")

		err := h.handle(h.ctx, runner)
		if err != nil {
			log.Println("Handling error:", err)
		}
		eCh <- err
	}()

	return <-eCh
}

func (h *Handler) getLocalRunner() (eval.Runner, error) {
	log.Println("Trying to spin up local grader")
	bm, err := boxmanager.New(config.Eval.NumConcurrent, h.dm)
	if err != nil {
		return nil, err
	}
	log.Println("Running local grader")
	return bm, nil
}

func (h *Handler) getAppropriateRunner() (eval.Runner, error) {
	if boxmanager.CheckCanRun() {
		runner, err := h.getLocalRunner()
		if err == nil {
			return runner, nil
		}
	}
	log.Fatalln("Remote grader has been disabled because it can't run problems with custom checker")
	return nil, nil
	/*
		Disabled until it fully works
		return nil, nil
		log.Println("Could not spin up local grader, trying to contact remote")
		return newGrpcRunner(config.Eval.Address)
	*/
}

func getAppropriateChecker(runner eval.Runner, sub *kilonova.Submission, pb *kilonova.Problem) (eval.Checker, error) {
	switch pb.Type {
	case kilonova.ProblemTypeClassic:
		return &checkers.DiffChecker{}, nil
	case kilonova.ProblemTypeCustomChecker:
		return checkers.NewCustomChecker(runner, pb, sub)
	default:
		log.Println("Unknown problem type", pb.Type)
		return nil, errors.New("Unknown problem type")
	}
}
