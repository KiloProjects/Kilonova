package checkers

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
)

func standardCheckerTask(ctx context.Context, box eval.Sandbox, job *customCheckerInput) (*checkerResult, error) {
	rez := &checkerResult{}
	lang, ok := eval.Langs[eval.GetLangByFilename(job.c.filename)]
	if !ok {
		rez.Output = ErrOut
		return rez, nil
	}

	if err := box.WriteFile("/box/program.out", job.pOut, 0644); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	if err := box.WriteFile("/box/correct.in", job.cIn, 0644); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	if err := box.WriteFile("/box/correct.out", job.cOut, 0644); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	if err := box.WriteFile("/box/contestant.txt", strings.NewReader(job.c.sub.Code), 0644); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	if err := eval.CopyInBox(box, path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", -job.c.sub.ID)), lang.CompiledName); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	goodCmd, err := eval.MakeGoodCommand(lang.RunCommand)
	if err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	goodCmd = append(goodCmd, "/box/correct.in", "/box/correct.out", "/box/program.out")

	var stdout, stderr bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &stdout,
		Stderr: &stderr,

		MemoryLimit: checkerMemoryLimit,

		WallTimeLimit: 20,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	floatScore, err := strconv.ParseFloat(strings.TrimSpace(stdout.String()), 64)
	if err != nil {
		rez.Output = "Invalid checker score"
		return rez, nil
	}
	rez.Score = int(floatScore * 100)

	rez.Output = strings.TrimSpace(stderr.String())
	if rez.Output == "" {
		rez.Output = "No checker message"
	}
	return rez, nil
}
