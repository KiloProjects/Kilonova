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

var _ eval.Checker = &CustomChecker{}

type customCheckerInput struct {
	c    *CustomChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader
}

type checkerResult struct {
	Score  int
	Output string
}

type CustomChecker struct {
	mgr      eval.BoxScheduler
	pb       *kilonova.Problem
	sub      *kilonova.Submission
	filename string
	code     []byte
	Logger   *zap.SugaredLogger

	legacy bool
}

// Prepare compiles the grader
func (c *CustomChecker) Prepare(ctx context.Context) (string, error) {
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

func (c *CustomChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, int) {
	var out checkerResult

	if c.legacy {
		resp, err := eval.RunTask(ctx, c.mgr, checkerMemoryLimit, &customCheckerInput{
			c:    c,
			pOut: pOut,
			cIn:  cIn,
			cOut: cOut,
		}, legacyCheckerTask)
		if err != nil || resp == nil {
			return ErrOut, 0
		}

		out = *resp
	} else {
		resp, err := eval.RunTask(ctx, c.mgr, checkerMemoryLimit, &customCheckerInput{
			c:    c,
			pOut: pOut,
			cIn:  cIn,
			cOut: cOut,
		}, standardCheckerTask)
		if err != nil || resp == nil {
			return ErrOut, 0
		}

		out = *resp
	}

	return out.Output, out.Score
}

func (c *CustomChecker) Cleanup(_ context.Context) error {
	return eval.CleanCompilation(-c.sub.ID)
}

func NewLegacyCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub, filename, code, logger, true}, nil
}

func NewStandardCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, sub *kilonova.Submission, filename string, code []byte) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub, filename, code, logger, false}, nil
}
