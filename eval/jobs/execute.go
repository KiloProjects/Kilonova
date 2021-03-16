package jobs

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
)

var _ eval.Job = &ExecuteJob{}

type ExecuteJob struct {
	Req   *eval.ExecRequest
	Resp  *eval.ExecResponse
	DM    kilonova.DataStore
	Debug bool
}

func (job *ExecuteJob) Execute(ctx context.Context, box eval.Sandbox) error {
	if job.Debug {
		log.Printf("Executing test %d using box %d\n", job.Req.SubtestID, box.GetID())
	}

	in, err := job.DM.TestInput(int(job.Req.TestID))
	if err != nil {
		return err
	}
	defer in.Close()

	if err := box.WriteFile("/box/"+job.Req.Filename+".in", in, 0644); err != nil {
		fmt.Println("Can't write input file:", err)
		job.Resp.Comments = "Sandbox error: Couldn't write input file"
		return err
	}
	consoleInput := job.Req.Filename == "stdin"

	lang := config.Languages[job.Req.Lang]
	if err := eval.CopyInBox(box, path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", job.Req.SubID)), lang.CompiledName); err != nil {
		job.Resp.Comments = "Couldn't copy executable in box"
		return err
	}

	lim := eval.Limits{
		MemoryLimit: job.Req.MemoryLimit,
		StackLimit:  job.Req.StackLimit,
		TimeLimit:   job.Req.TimeLimit,
	}
	meta, err := eval.RunSubmission(ctx, box, config.Languages[job.Req.Lang], lim, consoleInput)
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
