package grader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	pb "github.com/KiloProjects/kilonova/grpc"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
)

var True = true
var waitingSubs = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting}
var workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}

type Handler struct {
	ctx    context.Context
	sChan  chan *kilonova.Submission
	kn     *logic.Kilonova
	dm     kilonova.DataStore
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

func (h *Handler) handle(ctx context.Context, client pb.EvalClient) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case sub, more := <-h.sChan:
			if !more {
				return nil
			}

			var score_mu sync.Mutex
			var score int

			resp, err := client.Compile(ctx, &pb.CompileRequest{ID: int32(sub.ID), Code: sub.Code, Lang: sub.Language})

			if err != nil {
				log.Println("Error from eval:", err)
				continue
			}

			if h.debug {
				old := resp.Output
				resp.Output = "<output stripped>"
				spew.Dump(resp)
				resp.Output = old
			}

			// if err := sub.SetCompilation(!resp.Success, resp.Output); err != nil {
			if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{}); err != nil {
				log.Println("Error during update of compile information:", err)
				continue
			}

			problem, err := h.pserv.ProblemByID(ctx, sub.ProblemID)
			if err != nil {
				log.Println("Error during submission problem getting:", err)
				continue
			}

			score = problem.DefaultPoints

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

				protobufTest := &pb.Test{
					ID:          int32(sub.ID),
					TID:         int32(test.ID),
					Filename:    filename,
					StackLimit:  int32(problem.StackLimit),
					MemoryLimit: int32(problem.MemoryLimit),
					TimeLimit:   problem.TimeLimit,
					Lang:        sub.Language,
					TestID:      int64(pbTest.ID),
				}
				go func() {
					defer wg.Done()
					subTestID := test.ID
					pbTest := pbTest

					resp, err := client.Execute(ctx, protobufTest)
					if err != nil {
						log.Printf("Error executing test: %v\n", err)
						return
					}

					if h.debug {
						spew.Dump(resp)
					}

					var testScore int

					// Make sure TLEs are fully handled
					if resp.Time > problem.TimeLimit {
						resp.Comments = "TLE"
						testScore = 0
					}

					if resp.Comments == "" {
						tPath := h.dm.TestOutputPath(pbTest.ID)
						sPath := h.dm.SubtestPath(subTestID)

						equal := compareOutputs(tPath, sPath)

						// After comparing outputs, I think we can safely delete the submission.
						// In actuality we can't because it's a file made by root
						// TODO
						/*if err := h.dm.RemoveSubtestData(subTestID); err != nil {
							log.Printf("WARNING: Couldn't remove subtest data for %d: %q", subTestID, err)
						}*/

						if equal {
							testScore = int(pbTest.Score)
							resp.Comments = "Correct Answer"
						} else {
							resp.Comments = "Wrong Answer"
						}
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

			if _, err := client.Clean(ctx, &pb.CleanArgs{ID: int32(sub.ID)}); err != nil {
				log.Printf("Couldn't clean task: %s\n", err)
			}

			if err := h.sserv.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score}); err != nil {
				log.Println("Error 1214:", err)
			}
		}
	}
}

func (h *Handler) Start() error {
	// Dial here to pre-emptively exit in case it fails
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, config.Eval.Address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		if err == context.DeadlineExceeded {
			return errors.New("WARNING: No grader found, will not grade submissions")
		}
		return fmt.Errorf("Dialing error: %w", err)
	}

	client := pb.NewEvalClient(conn)

	go h.chFeeder(4 * time.Second)

	eCh := make(chan error, 1)
	go func() {
		defer conn.Close()
		log.Println("Connected to eval")

		err := h.handle(h.ctx, client)
		if err != nil {
			log.Println("Handling error:", err)
		}
		eCh <- err
	}()

	return <-eCh
}

func compareOutputs(tPath, cPath string) bool {
	// temporary solution until i find something better
	cmd := exec.Command("diff", "-qBbEa", tPath, cPath)
	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			return err.ExitCode() == 0
		}
		return false
	}
	return true
}
