package checkers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
)

type standardCustomCheckerTask struct {
	c    *CustomChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader

	// filled by Execute
	score  int
	output string
}

func (job *standardCustomCheckerTask) Execute(ctx context.Context, box eval.Sandbox) error {
	lang, ok := eval.Langs[eval.GetLangByFilename(job.c.filename)]
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
	goodCmd = append(goodCmd, "/box/correct.in", "/box/correct.out", "/box/program.out")

	var stdout, stderr bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &stdout,
		Stderr: &stderr,

		MemoryLimit: 512 * 1024,

		WallTimeLimit: 20,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		job.output = ErrOut
		return nil
	}

	floatScore, err := strconv.ParseFloat(strings.TrimSpace(stdout.String()), 64)
	if err != nil {
		job.output = "Invalid checker score"
		return nil
	}
	job.score = int(floatScore * 100)

	job.output = strings.TrimSpace(stderr.String())
	if job.output == "" {
		job.output = "No checker message"
	}
	return nil
}
