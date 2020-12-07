package grader

import (
	"bytes"
	"context"
	"log"
	"sync"
	"time"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
	pb "github.com/KiloProjects/Kilonova/internal/grpc"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
)

type Handler struct {
	ctx   context.Context
	tChan chan *db.Submission
	kn    *logic.Kilonova
	db    *db.DB
	dm    datamanager.Manager
	debug bool
}

func NewHandler(ctx context.Context, kn *logic.Kilonova) *Handler {
	ch := make(chan *db.Submission, 5)
	return &Handler{ctx, ch, kn, kn.DB, kn.DM, kn.Debug}
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder() {
	ticker := time.NewTicker(4 * time.Second)
	for {
		select {
		case <-ticker.C:
			subs, err := h.db.WaitingSubmissions(h.ctx)
			if err != nil {
				log.Println("Error fetching submissions:", err)
				continue
			}
			if subs != nil {
				log.Printf("Found %d submissions\n", len(subs))
				for _, sub := range subs {
					h.tChan <- sub
				}
			}
		case <-h.ctx.Done():
			ticker.Stop()
			return
		}
	}
}

func (h *Handler) Handle(ctx context.Context, client pb.EvalClient) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case t, more := <-h.tChan:
			if !more {
				return nil
			}

			err := t.SetStatus(db.StatusWorking, 0)
			if err != nil {
				log.Println(err)
				continue
			}

			var score_mu sync.Mutex
			var score int

			resp, err := client.Compile(ctx, &pb.CompileRequest{ID: int32(t.ID), Code: t.Code, Lang: t.Language})

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

			if err := t.SetCompilation(!resp.Success, resp.Output); err != nil {
				log.Println("Error during update of compile information:", err)
				continue
			}

			problem, err := h.db.Problem(ctx, t.ProblemID)
			if err != nil {
				log.Println("Error during submission problem getting:", err)
				continue
			}

			tests, err := h.db.SubTests(ctx, t.ID)
			if resp.Success == false || err != nil {
				if err := t.SetStatus(db.StatusFinished, score); err != nil {
					log.Println(err)
				}
				continue
			}

			var wg sync.WaitGroup

			for _, test := range tests {
				test := test
				wg.Add(1)
				pbTest, err := h.db.TestByID(ctx, test.TestID)
				if err != nil {
					log.Println("Error during test getting (0.5):", err)
					continue
				}

				input, _, err := h.dm.Test(t.ProblemID, pbTest.VisibleID)
				if err != nil {
					log.Println("Error during test getting (1):", err)
					continue
				}

				filename := problem.TestName
				if problem.ConsoleInput {
					filename = "stdin"
				}

				pbtest := &pb.Test{
					ID:          int32(t.ID),
					TID:         int32(test.ID),
					Filename:    filename,
					StackLimit:  int32(problem.StackLimit),
					MemoryLimit: int32(problem.MemoryLimit),
					TimeLimit:   problem.TimeLimit,
					Lang:        t.Language,
					Input:       input,
				}
				go func() {
					defer wg.Done()
					subTestID := test.ID
					pbTest := pbTest

					resp, err := client.Execute(ctx, pbtest)
					if err != nil {
						log.Printf("Error executing test: %v\n", err)
						return
					}

					if h.debug {
						old := resp.Output
						resp.Output = []byte("<output stripped>")
						spew.Dump(resp)
						resp.Output = old
					}

					var testScore int

					if resp.Comments == "" {
						_, out, err := h.dm.Test(t.ProblemID, pbTest.VisibleID)
						if err != nil {
							log.Println("Error during test getting (2):", err)
							return
						}
						equal := compareOutputs(out, []byte(resp.Output))

						if equal {
							testScore = int(pbTest.Score)
							resp.Comments = "Correct Answer"
						} else {
							resp.Comments = "Wrong Answer"
						}
					}

					// Make sure TLEs are fully handled
					if resp.Time > problem.TimeLimit {
						resp.Comments = "TLE"
						testScore = 0
					}

					subTest, err := h.db.SubTest(h.ctx, subTestID)
					if err != nil {
						log.Printf("Error getting subtest: %v\n", err)
						return
					}

					if err := subTest.SetData(resp.Memory, testScore, resp.Time, resp.Comments); err != nil {
						log.Println("Error during evaltest updating:", err)
						return
					}

					score_mu.Lock()
					score += testScore
					score_mu.Unlock()

				}()
			}

			wg.Wait()

			if _, err := client.Clean(ctx, &pb.CleanArgs{ID: int32(t.ID)}); err != nil {
				log.Printf("Couldn't clean task: %s\n", err)
			}

			t.SetStatus(db.StatusFinished, score)
		}
	}
}

func (h *Handler) Start() {
	// Dial here to pre-emptively exit in case it fails
	conn, err := grpc.Dial("localhost:8001", grpc.WithInsecure())
	if err != nil {
		log.Println("Dialing error:", err)
		return
	}

	client := pb.NewEvalClient(conn)

	if _, err = client.Ping(context.Background(), &pb.Empty{}); err != nil {
		log.Println(err)
		return
	}

	go h.chFeeder()

	go func() {
		defer conn.Close()
		log.Println("Connected to eval")

		if err := h.Handle(h.ctx, client); err != nil {
			log.Println("Handling error:", err)
		}
	}()
}

func compareOutputs(tOut, cOut []byte) bool {
	tOut = bytes.TrimSpace(tOut)
	tOut = bytes.ReplaceAll(tOut, []byte{'\r', '\n'}, []byte{'\n'})

	cOut = bytes.TrimSpace(cOut)
	cOut = bytes.ReplaceAll(cOut, []byte{'\r', '\n'}, []byte{'\n'})

	return bytes.Equal(tOut, cOut)
}
