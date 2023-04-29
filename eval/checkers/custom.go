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

const (
	checkerMemoryLimit = 512 * 1024
)

var _ eval.Checker = &customChecker{}

type customCheckerInput struct {
	c    *customChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader
}

type checkerResult struct {
	Score  int
	Output string
}

type customChecker struct {
	mgr      eval.BoxScheduler
	pb       *kilonova.Problem
	sub      *kilonova.Submission
	filename string
	code     []byte
	Logger   *zap.SugaredLogger

	legacy bool
}

// Prepare compiles the grader
func (c *customChecker) Prepare(ctx context.Context) (string, error) {
	resp, err := eval.RunTask(ctx, c.mgr, 0, &eval.CompileRequest{
		ID: -c.sub.ID,
		CodeFiles: map[string][]byte{
			eval.Langs[eval.GetLangByFilename(c.filename)].SourceName: c.code,
		},
		Lang: eval.GetLangByFilename(c.filename),
	}, tasks.GetCompileTask(c.Logger))
	if err != nil {
		return "Couldn't compile checker", err
	}

	if !resp.Success {
		return fmt.Sprintf("Output:\n%s\nOther:\n%s", resp.Output, resp.Other), kilonova.Statusf(400, "Invalid helper code")
	}

	return "", nil
}

func (c *customChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, int) {
	var out checkerResult

	task := standardCheckerTask
	if c.legacy {
		task = legacyCheckerTask
	}

	resp, err := eval.RunTask(ctx, c.mgr, checkerMemoryLimit, &customCheckerInput{
		c:    c,
		pOut: pOut,
		cIn:  cIn,
		cOut: cOut,
	}, task)
	if err != nil || resp == nil {
		return ErrOut, 0
	}

	out = *resp

	return out.Output, out.Score
}

func (c *customChecker) Cleanup(_ context.Context) error {
	return eval.CleanCompilation(-c.sub.ID)
}

func NewLegacyCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) eval.Checker {
	return &customChecker{mgr, pb, sub, filename, code, logger, true}
}

func NewStandardCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) eval.Checker {
	return &customChecker{mgr, pb, sub, filename, code, logger, false}
}
