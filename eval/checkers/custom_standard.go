package checkers

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/shopspring/decimal"
)

func standardCheckerTask(ctx context.Context, mgr eval.BoxScheduler, job *customCheckerInput, log *slog.Logger) (*checkerResult, error) {
	rez := &checkerResult{}
	lang, ok := eval.Langs[eval.GetLangByFilename(job.c.filename)]
	if !ok {
		rez.Output = ErrOut
		return rez, nil
	}

	req := initRequest(lang, job)

	req.Command = append(slices.Clone(lang.RunCommand), "/box/correct.in", "/box/correct.out", "/box/program.out")
	req.RunConfig.OutputPath = "/box/checker_verdict.out"
	req.RunConfig.StderrPath = "/box/checker_verdict.err"
	req.OutputByteFiles = []string{"/box/checker_verdict.out", "/box/checker_verdict.err"}

	resp, err := mgr.RunBox2(ctx, req, checkerMemoryLimit)
	if resp == nil || err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	stdout, ok := resp.ByteFiles["/box/checker_verdict.out"]
	if !ok {
		stdout = []byte{}
	}
	stderr, ok := resp.ByteFiles["/box/checker_verdict.err"]
	if !ok {
		stderr = []byte{}
	}

	floatScore, err := strconv.ParseFloat(strings.TrimSpace(string(stdout)), 64)
	if err != nil {
		rez.Output = "Invalid checker score"
		return rez, nil
	}
	rez.Percentage = decimal.NewFromFloat(floatScore).Shift(2)

	rez.Output = strings.TrimSpace(string(stderr))
	if rez.Output == "" {
		rez.Output = "No checker message"
	}
	return rez, nil
}
