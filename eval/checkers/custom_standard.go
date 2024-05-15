package checkers

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func standardCheckerTask(ctx context.Context, box eval.Sandbox, job *customCheckerInput) (*checkerResult, error) {
	rez := &checkerResult{}
	lang, ok := eval.Langs[eval.GetLangByFilename(job.c.filename)]
	if !ok {
		rez.Output = ErrOut
		return rez, nil
	}

	if ok := copyFiles(box, lang, job); !ok {
		rez.Output = ErrOut
		return rez, nil
	}

	goodCmd, err := eval.MakeGoodCommand(lang.RunCommand)
	if err != nil {
		rez.Output = ErrOut
		return rez, nil
	}
	goodCmd = append(goodCmd, "/box/correct.in", "/box/correct.out", "/box/program.out")

	conf := &eval.RunConfig{
		OutputPath: "/box/checker_verdict.out",
		StderrPath: "/box/checker_verdict.err",

		MemoryLimit: checkerMemoryLimit,

		WallTimeLimit: 20,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	var stdout, stderr bytes.Buffer
	if err := box.ReadFile("/box/checker_verdict.out", &stdout); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warn("Couldn't read checker stdout: ", err)
		}
		stdout.Reset()
	}
	if err := box.ReadFile("/box/checker_verdict.err", &stderr); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warn("Couldn't read checker stderr: ", err)
		}
		stderr.Reset()
	}

	floatScore, err := strconv.ParseFloat(strings.TrimSpace(stdout.String()), 64)
	if err != nil {
		rez.Output = "Invalid checker score"
		return rez, nil
	}
	rez.Percentage = decimal.NewFromFloat(floatScore).Shift(2)

	rez.Output = strings.TrimSpace(stderr.String())
	if rez.Output == "" {
		rez.Output = "No checker message"
	}
	return rez, nil
}
