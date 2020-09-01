package grader

import (
	"bytes"
	"context"
	"log"
	"net"
	"time"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader/proto"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/davecgh/go-spew/spew"
)

type Handler struct {
	tChan  chan common.Task
	db     *kndb.DB
	dm     datamanager.Manager
	ctx    context.Context
	logger *log.Logger
}

func NewHandler(ctx context.Context, db *kndb.DB, dm datamanager.Manager, logger *log.Logger) *Handler {
	ch := make(chan common.Task, 5)
	return &Handler{ch, db, dm, ctx, logger}
}

// chFeeder "feeds" tChan with relevant data
func (h *Handler) chFeeder() {
	ticker := time.NewTicker(4 * time.Second)
	for {
		select {
		case <-ticker.C:
			tasks, err := h.db.GetWaitingTasks()
			if err != nil {
				h.logger.Println("Error fetching tasks:", err)
				continue
			}
			if tasks != nil {
				h.logger.Printf("Found %d tasks\n", len(tasks))
				for _, task := range tasks {
					h.tChan <- task
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

func (h *Handler) Handle(send chan<- proto.Message, recv <-chan proto.Message) error {
	// TODO: Change to "stdin" for input, also maybe allow separate filename for stdout
	for t := range h.tChan {
		h.db.UpdateStatus(t.ID, common.StatusWorking, 0)
		var score int

		send <- proto.ArgToMessage(proto.Compile{ID: int(t.ID), Code: t.SourceCode, Language: t.Language})

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

		if err := h.db.UpdateCompilation(resp); err != nil {
			h.logger.Println("Error during update of compile information:", err)
			continue
		}

		if resp.Success == false {
			h.db.UpdateStatus(t.ID, common.StatusDone, score)
			continue
		}

		for _, test := range t.Tests {
			input, _, err := h.dm.GetTest(t.ProblemID, test.Test.VisibleID)
			if err != nil {
				h.logger.Println("Error during test getting (1):", err)
				continue
			}

			filename := t.Problem.TestName
			if t.Problem.ConsoleInput {
				filename = "input"
			}

			send <- proto.ArgToMessage(proto.STask{
				ID:          int(t.ID),
				TID:         int(test.ID),
				Filename:    filename,
				StackLimit:  int(t.Problem.StackLimit),
				MemoryLimit: int(t.Problem.MemoryLimit),
				TimeLimit:   t.Problem.TimeLimit,
				Language:    t.Language,
				Input:       string(input),
			})
		}

		// this depends on the fact that we do stuff on a single loop
		for _, test := range t.Tests {
			var resp proto.STResponse

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

			var testScore int

			if resp.Comments == "" {
				_, out, err := h.dm.GetTest(t.ProblemID, test.Test.VisibleID)
				if err != nil {
					h.logger.Println("Error during test getting (2):", err)
					continue
				}
				equal := compareOutputs(out, []byte(resp.Output))

				if equal {
					testScore = test.Test.Score
					resp.Comments = "Correct Answer"
				} else {
					resp.Comments = "Wrong Answer"
				}
			}

			// Make sure TLEs are fully handled
			if resp.Time > t.Problem.TimeLimit {
				resp.Comments = "TLE"
				testScore = 0
			}

			if err := h.db.UpdateEvalTest(resp, testScore); err != nil {
				h.logger.Println("Error during evaltest updating:", err)
			}

			score += testScore
		}

		send <- proto.ArgToMessage(proto.TRemove{ID: int(t.ID)})

		h.db.UpdateStatus(t.ID, common.StatusDone, score)
	}
	return nil
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

		if err := proto.Handle(conn, h.Handle); err != nil {
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
