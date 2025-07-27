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

func legacyCheckerTask(ctx context.Context, mgr eval.BoxScheduler, job *customCheckerInput, _ *slog.Logger) (string, decimal.Decimal) {
	lang := mgr.LanguageFromFilename(job.c.filename)
	if lang == nil {
		return ErrOut, decimal.Zero
	}

	req := initRequest(lang, job)

	req.Command = append(slices.Clone(lang.RunCommand), "/box/program.out", "/box/correct.out", "/box/correct.in")
	req.RunConfig.OutputPath = "/box/verdict.out"
	req.OutputByteFiles = []string{"/box/verdict.out"}

	resp, err := mgr.RunBox2(ctx, req, checkerMemoryLimit)
	if resp == nil || err != nil {
		return ErrOut, decimal.Zero
	}

	val, ok := resp.ByteFiles["/box/verdict.out"]
	if !ok || val == nil {
		return "Invalid checker output", decimal.Zero
	}
	percVal, message, found := strings.Cut(string(val), " ")

	percentage, err := strconv.ParseFloat(percVal, 64)
	if err != nil {
		return "Wrong checker output", decimal.Zero
	}

	var output string
	if val := strings.TrimSpace(message); val == "" || !found {
		output = "No checker message"
	} else {
		output = val
	}
	return output, decimal.NewFromFloat(percentage)
}
