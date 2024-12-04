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

func legacyCheckerTask(ctx context.Context, mgr eval.BoxScheduler, job *customCheckerInput, _ *slog.Logger) (*checkerResult, error) {
	rez := &checkerResult{}
	lang := mgr.LanguageFromFilename(job.c.filename)
	if lang == nil {
		rez.Output = ErrOut
		return rez, nil
	}

	req := initRequest(lang, job)

	req.Command = append(slices.Clone(lang.RunCommand), "/box/program.out", "/box/correct.out", "/box/correct.in")
	req.RunConfig.OutputPath = "/box/checker_verdict.out"
	req.OutputByteFiles = []string{"/box/checker_verdict.out"}

	resp, err := mgr.RunBox2(ctx, req, checkerMemoryLimit)
	if resp == nil || err != nil {
		rez.Output = ErrOut
		return rez, nil
	}

	val, ok := resp.ByteFiles["/box/checker_verdict.out"]
	if !ok || val == nil {
		rez.Output = "Invalid checker output"
		return rez, nil
	}
	percVal, message, found := strings.Cut(string(val), " ")

	percentage, err := strconv.ParseFloat(percVal, 64)
	if err != nil {
		rez.Output = "Wrong checker output"
		return rez, nil
	}
	rez.Percentage = decimal.NewFromFloat(percentage)

	if val := strings.TrimSpace(message); val == "" || !found {
		rez.Output = "No checker message"
	} else {
		rez.Output = val
	}
	return rez, nil
}
