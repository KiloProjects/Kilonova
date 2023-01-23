package checkers

import (
	"context"
	"fmt"
	"io"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"go.uber.org/zap"
)

var _ eval.Checker = &CustomChecker{}

type CustomChecker struct {
	mgr      eval.Runner
	pb       *kilonova.Problem
	sub      *kilonova.Submission
	filename string
	code     []byte
	Logger   *zap.SugaredLogger

	legacy bool
}

// Prepare compiles the grader
func (c *CustomChecker) Prepare(ctx context.Context) (string, error) {
	job := tasks.NewCompileTask(&eval.CompileRequest{
		ID: -c.sub.ID,
		CodeFiles: map[string][]byte{
			eval.Langs[eval.GetLangByFilename(c.filename)].SourceName: c.code,
		},
		Lang: eval.GetLangByFilename(c.filename),
	},
		c.Logger,
	)

	err := c.mgr.RunTask(ctx, job)
	if err != nil {
		return "Couldn't compile checker", err
	}

	if !job.Resp.Success {
		return fmt.Sprintf("Output:\n%s\nOther:\n%s", job.Resp.Output, job.Resp.Other), kilonova.Statusf(400, "Invalid helper code")
	}

	return "", nil
}

func (c *CustomChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, int) {
	if c.legacy {
		task := &legacyCustomCheckerTask{
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

	task := &standardCustomCheckerTask{
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

func NewLegacyCustomChecker(mgr eval.Runner, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub, filename, code, logger, true}, nil
}

func NewStandardCustomChecker(mgr eval.Runner, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub, filename, code, logger, false}, nil
}
