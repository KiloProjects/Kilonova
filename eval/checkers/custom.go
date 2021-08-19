package checkers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
)

var _ eval.Checker = &CustomChecker{}
var _ eval.Task = &customCheckerTask{}

type CustomChecker struct {
	mgr eval.Runner
	pb  *kilonova.Problem
	sub *kilonova.Submission
}

// Prepare compiles the grader
func (c *CustomChecker) Prepare(ctx context.Context) (string, error) {
	job := &tasks.CompileTask{
		Req: &eval.CompileRequest{
			ID:   -c.sub.ID,
			Code: []byte(c.pb.HelperCode),
			Lang: c.pb.HelperCodeLang,
		},
	}

	err := c.mgr.RunTask(ctx, job)
	if err != nil {
		return "Couldn't compile checker", err
	}

	if !job.Resp.Success {
		return fmt.Sprintf("Output:\n%s\nOther:\n%s", job.Resp.Output, job.Resp.Other), &kilonova.Error{Code: kilonova.EINVALID, Message: "Invalid helper code"}
	}

	return "", nil
}

type customCheckerTask struct {
	c    *CustomChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader

	// filled by Execute
	score  int
	output string
}

var customTaskErr = kilonova.Error{Code: kilonova.EINTERNAL, Message: ErrOut}

func (job *customCheckerTask) Execute(ctx context.Context, box eval.Sandbox) error {
	lang, ok := eval.Langs[job.c.pb.HelperCodeLang]
	if !ok {
		job.output = ErrOut
		return nil
	}

	if err := box.WriteFile("/box/program.out", job.pOut, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := box.WriteFile("/box/correct.in", job.cIn, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := box.WriteFile("/box/correct.out", job.cOut, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := eval.CopyInBox(box, path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", -job.c.sub.ID)), lang.CompiledName); err != nil {
		job.output = ErrOut
		return nil
	}

	goodCmd, err := eval.MakeGoodCommand(lang.RunCommand)
	if err != nil {
		job.output = ErrOut
		return nil
	}
	// TODO: Make sure all supported languages can have this
	// Add the program output, correct output and max score parameters
	goodCmd = append(goodCmd, "/box/program.out", "/box/correct.out", "/box/correct.in")

	var out bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &out,

		MemoryLimit: 64 * 1024,
		StackLimit:  32 * 1024,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		job.output = ErrOut
		return nil
	}

	if _, err := fmt.Fscanf(&out, "%d ", &job.score); err != nil {
		job.output = "Wrong checker output"
		return nil
	}

	job.output = out.String()
	return nil
}

func (c *CustomChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, int) {
	task := &customCheckerTask{
		c:    c,
		pOut: pOut,
		cIn:  cIn,
		cOut: cOut,
	}

	if err := c.mgr.RunTask(ctx, task); err != nil {
		return ErrOut, 0
	}

	return task.output, task.score
}

func (c *CustomChecker) Cleanup(_ context.Context) error {
	return eval.CleanCompilation(-c.sub.ID)
}

func NewCustomChecker(mgr eval.Runner, pb *kilonova.Problem, sub *kilonova.Submission) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub}, nil
}
