package checkers

import (
	"context"
	"log/slog"
	"slices"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/shopspring/decimal"
)

func standardCheckerTask(ctx context.Context, mgr eval.BoxScheduler, job *customCheckerInput, _ *slog.Logger) (string, decimal.Decimal) {
	lang := mgr.LanguageFromFilename(job.c.filename)
	if lang == nil {
		return ErrOut, decimal.Zero
	}

	req := initRequest(lang, job)

	req.Command = append(slices.Clone(lang.RunCommand), "/box/correct.in", "/box/correct.out", "/box/program.out")
	req.RunConfig.OutputPath = "/box/verdict.out"
	req.RunConfig.StderrPath = "/box/verdict.err"
	req.OutputByteFiles = []string{"/box/verdict.out", "/box/verdict.err"}

	resp, err := mgr.RunBox2(ctx, req, checkerMemoryLimit)
	if resp == nil || err != nil {
		return ErrOut, decimal.Zero
	}

	percentage, output := tasks.ParseStandardManagerOutput(resp)
	return output, percentage
}
