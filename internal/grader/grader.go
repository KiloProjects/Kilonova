package grader

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"net"
	"time"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/proto"
	"github.com/davecgh/go-spew/spew"
)

type Handler struct {
	tChan  chan db.Submission
	db     *db.Queries
	dm     datamanager.Manager
	ctx    context.Context
	logger *log.Logger
}

func NewHandler(ctx context.Context, DB *db.Queries, dm datamanager.Manager, logger *log.Logger) *Handler {
	ch := make(chan db.Submission, 5)
	return &Handler{ch, DB, dm, ctx, logger}
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder() {
	ticker := time.NewTicker(4 * time.Second)
	for {
		select {
		case <-ticker.C:
			subs, err := h.db.WaitingSubmissions(h.ctx)
			if err != nil {
				h.logger.Println("Error fetching submissions:", err)
				continue
			}
			if subs != nil {
				h.logger.Printf("Found %d submissions\n", len(subs))
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

func ldump(logger *log.Logger, args ...interface{}) {
	spew.Fdump(logger.Writer(), args...)
}

func (h *Handler) Handle(ctx context.Context, send chan<- proto.Message, recv <-chan proto.Message) error {
	// TODO: Change to "stdin" for input, also maybe allow separate filename for stdout
	for {
		select {
		case <-ctx.Done():
			return nil
		case t, more := <-h.tChan:
			if !more {
				return nil
			}

			err := h.db.SetSubmissionStatus(ctx, db.SetSubmissionStatusParams{ID: t.ID, Status: db.StatusWorking})
			if err != nil {
				h.logger.Println(err)
			}
			var score int32

			send <- proto.ArgToMessage(proto.Compile{ID: t.ID, Code: t.Code, Language: t.Language})

			var resp proto.CResponse

			msg := <-recv
			if msg.Type == "Error" {
				var perr proto.Error
				proto.DecodeArgs(msg, &perr)
				h.logger.Println("Error from eval:", perr.Value)
			}

			proto.DecodeArgs(msg, &resp)

			{
				old := resp.Output
				resp.Output = "<output stripped>"
				ldump(h.logger, resp)
				resp.Output = old
			}

			if err := h.db.SetCompilation(ctx, db.SetCompilationParams{
				ID:             resp.ID,
				CompileError:   sql.NullBool{Bool: !resp.Success, Valid: true},
				CompileMessage: sql.NullString{String: resp.Output, Valid: true}}); err != nil {

				h.logger.Println("Error during update of compile information:", err)
				continue
			}

			problem, err := h.db.Problem(ctx, t.ProblemID)
			if err != nil {
				h.logger.Println("Error during submission problem getting:", err)
				continue
			}

			tests, err := h.db.SubTests(ctx, t.ID)
			if resp.Success == false || err != nil {
				if err := h.db.SetSubmissionStatus(ctx, db.SetSubmissionStatusParams{ID: t.ID, Status: db.StatusFinished, Score: score}); err != nil {
					h.logger.Println(err)
				}
				continue
			}

			for _, test := range tests {
				pbTest, err := h.db.Test(ctx, test.TestID)
				if err != nil {
					h.logger.Println("Error during test getting (0.5):", err)
					continue
				}

				input, _, err := h.dm.GetTest(t.ProblemID, int64(pbTest.VisibleID))
				if err != nil {
					h.logger.Println("Error during test getting (1):", err)
					continue
				}

				filename := problem.TestName
				if problem.ConsoleInput {
					filename = "input"
				}

				send <- proto.ArgToMessage(proto.Test{
					ID:          t.ID,
					TID:         test.ID,
					Filename:    filename,
					StackLimit:  int(problem.StackLimit),
					MemoryLimit: int(problem.MemoryLimit),
					TimeLimit:   problem.TimeLimit,
					Language:    t.Language,
					Input:       string(input),
				})
			}

			// TODO: this depends on the fact that we do stuff on a single loop
			for _, test := range tests {
				pbTest, err := h.db.Test(ctx, test.TestID)
				if err != nil {
					h.logger.Println("Error during test getting (0.5):", err)
					continue
				}

				var resp proto.TResponse

				msg := <-recv
				if msg.Type == "Error" {
					var perr proto.Error
					proto.DecodeArgs(msg, &perr)
					h.logger.Println("Error from eval:", perr.Value)
				}

				proto.DecodeArgs(msg, &resp)

				{
					old := resp.Output
					resp.Output = "<output stripped>"
					ldump(h.logger, resp)
					resp.Output = old
				}

				var testScore int32

				if resp.Comments == "" {
					_, out, err := h.dm.GetTest(t.ProblemID, int64(pbTest.VisibleID))
					if err != nil {
						h.logger.Println("Error during test getting (2):", err)
						continue
					}
					equal := compareOutputs(out, []byte(resp.Output))

					if equal {
						testScore = pbTest.Score
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

				if err := h.db.SetSubmissionTest(ctx, db.SetSubmissionTestParams{ID: resp.TID, Memory: int32(resp.Memory), Score: testScore, Time: resp.Time, Verdict: resp.Comments}); err != nil {
					h.logger.Println("Error during evaltest updating:", err)
				}

				score += testScore
			}

			send <- proto.ArgToMessage(proto.TRemove{ID: t.ID})

			h.db.SetSubmissionStatus(ctx, db.SetSubmissionStatusParams{ID: t.ID, Status: db.StatusFinished, Score: score})
		}
	}
}

func (h *Handler) Start(path string) {
	// Dial here to pre-emptively exit in case it fails
	conn, err := net.Dial("unix", path)
	if err != nil {
		log.Println("Dialing error:", err)
		return
	}

	go h.chFeeder()

	go func() {
		defer conn.Close()
		log.Println("Connected to eval")

		if err := proto.Handle(h.ctx, conn, h.Handle); err != nil {
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
