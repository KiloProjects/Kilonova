package tasks

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

var _ eval.Task = &ExecuteTask{}

type ExecuteTask struct {
	Req   *eval.ExecRequest
	Resp  *eval.ExecResponse
	DM    kilonova.GraderStore
	Debug bool
}

func (job *ExecuteTask) Execute(ctx context.Context, box eval.Sandbox) error {
	if job.Debug {
		zap.S().Debugf("Executing test %d using box %d", job.Req.SubtestID, box.GetID())
	}

	in, err := job.DM.TestInput(job.Req.TestID)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := box.WriteFile("/box/"+job.Req.Filename+".in", in, 0644); err != nil {
		zap.S().Info("Can't write input file:", err)
		job.Resp.Comments = "Sandbox error: Couldn't write input file"
		return err
	}
	consoleInput := job.Req.Filename == "stdin"

	lang := eval.Langs[job.Req.Lang]
	if err := eval.CopyInBox(box, getIDExec(job.Req.SubID), lang.CompiledName); err != nil {
		job.Resp.Comments = "Couldn't copy executable in box"
		return err
	}

	lim := eval.Limits{
		MemoryLimit: job.Req.MemoryLimit,
		TimeLimit:   job.Req.TimeLimit,
	}
	meta, err := eval.RunSubmission(ctx, box, eval.Langs[job.Req.Lang], lim, consoleInput)
	if err != nil {
		job.Resp.Comments = fmt.Sprintf("Error running submission: %v", err)
		return nil
	}
	job.Resp.Time = meta.Time
	job.Resp.Memory = meta.Memory

	switch meta.Status {
	case "TO":
		job.Resp.Comments = "TLE: " + meta.Message
	case "RE":
		job.Resp.Comments = "Runtime Error: " + meta.Message
	case "SG":
		job.Resp.Comments = meta.Message
	case "XX":
		job.Resp.Comments = "Sandbox Error: " + meta.Message
	}

	boxOut := fmt.Sprintf("/box/%s.out", job.Req.Filename)
	if !box.FileExists(boxOut) {
		job.Resp.Comments = "No output file found"
		zap.S().Warn("No output file found", zap.Int("subtest_id", job.Req.SubtestID), zap.Int("box_id", box.GetID()), zap.Int("sub_id", job.Req.SubID))
		zap.S().Info("This may be a bug: ", spew.Sdump(box.ReadDir("/box/")))
		return nil
	}

	w, err := job.DM.SubtestWriter(job.Req.SubtestID)
	if err != nil {
		job.Resp.Comments = "Could not open problem output"
		return nil
	}

	if err := eval.CopyFromBox(box, boxOut, w); err != nil {
		job.Resp.Comments = "Could not write output file"
		return nil
	}

	if err := w.Close(); err != nil {
		job.Resp.Comments = "Could not close output file"
		return nil
	}

	return nil
}
