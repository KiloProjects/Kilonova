package checkers

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
)

func legacyCheckerTask(ctx context.Context, box eval.Sandbox, job *customCheckerInput) (*checkerResult, error) {
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
	goodCmd = append(goodCmd, "/box/program.out", "/box/correct.out", "/box/correct.in")

	var out bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &out,

		MemoryLimit: checkerMemoryLimit,

		WallTimeLimit: 20,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	if _, err := fmt.Fscanf(&out, "%d ", &rez.Score); err != nil {
		rez.Output = "Wrong checker output"
		return rez, nil
	}

	rez.Output = strings.TrimSpace(out.String())
	return rez, nil
}
