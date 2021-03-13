package checkers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/boxmanager"
	"github.com/KiloProjects/kilonova/internal/config"
)

var _ eval.Checker = &CustomChecker{}

type CustomChecker struct {
	mgr eval.Runner
	pb  *kilonova.Problem
	sub *kilonova.Submission
}

// Prepare compiles the grader
func (c *CustomChecker) Prepare(ctx context.Context) error {
	box, err := c.mgr.GetSandbox(ctx)
	if err != nil {
		return err
	}
	defer c.mgr.ReleaseSandbox(box)

	resp, err := c.mgr.Compile(ctx, &eval.CompileRequest{
		ID:   -c.sub.ID,
		Code: []byte(c.pb.HelperCode),
		Lang: c.pb.HelperCodeLang,
	})
	if err != nil {
		return err
	}

	if !resp.Success {
		return errors.New("Invalid helper code")
	}

	return nil
}

/*
var _ eval.Job = &CustomCheckerJob{}

type CustomCheckerJob struct {
	pOut     io.Reader
	cOut     io.Reader
	maxScore int
	c        *CustomChecker
}

func (c *CustomCheckerJob) Execute(ctx context.Context, box eval.Sandbox) error {
	return nil
}
*/

func (c *CustomChecker) RunChecker(ctx context.Context, pOut, cOut io.Reader, maxScore int) (string, int) {
	box, err := c.mgr.GetSandbox(ctx)
	if err != nil {
		return ErrOut, 0
	}
	defer c.mgr.ReleaseSandbox(box)

	lang, ok := config.Languages[c.pb.HelperCodeLang]
	if !ok {
		return ErrOut, 0
	}

	if err := box.WriteFile("/box/program.out", pOut, 0644); err != nil {
		return ErrOut, 0
	}
	if err := box.WriteFile("/box/correct.out", cOut, 0644); err != nil {
		return ErrOut, 0
	}
	if err := eval.CopyInBox(box, path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", -c.sub.ID)), lang.CompiledName); err != nil {
		return ErrOut, 0
	}

	goodCmd, err := boxmanager.MakeGoodCommand(lang.RunCommand)
	if err != nil {
		return ErrOut, 0
	}
	// TODO: Make sure all supported languages can have this
	// Add the program output, correct output and max score parameters
	goodCmd = append(goodCmd, "/box/program.out", "/box/correct.out", strconv.Itoa(maxScore))

	var out bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &out,

		MemoryLimit: 64 * 1024,
		StackLimit:  32 * 1024,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		return ErrOut, 0
	}

	var score int
	if _, err := fmt.Fscanf(&out, "%d ", &score); err != nil {
		return "Wrong checker output", 0
	}

	return out.String(), score
}

func (c *CustomChecker) Cleanup(ctx context.Context) error {
	return c.mgr.Clean(ctx, -c.sub.ID)
}

func NewCustomChecker(mgr eval.Runner, pb *kilonova.Problem, sub *kilonova.Submission) (*CustomChecker, error) {
	return &CustomChecker{mgr, pb, sub}, nil
}
