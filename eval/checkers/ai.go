package checkers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/shopspring/decimal"
)

var _ Checker = (*aiChecker)(nil)

// aiChecker is the checker for AI-style problems
type aiChecker struct {
	mgr  eval.BoxScheduler
	pb   *kilonova.Problem
	code []byte

	uvToml []byte
	uvLock []byte

	store *datastore.Manager

	Logger *slog.Logger

	filename string
}

func NewAIChecker(mgr eval.BoxScheduler, store *datastore.Manager, logger *slog.Logger, pb *kilonova.Problem, filename string, code []byte, uvToml []byte, uvLock []byte) Checker {
	return &aiChecker{
		mgr:    mgr,
		pb:     pb,
		code:   code,
		uvToml: uvToml,
		uvLock: uvLock,
		store:  store,
		Logger: logger,

		filename: filename,
	}
}

func (c *aiChecker) Prepare(ctx context.Context) (string, error) {
	checkerPrepareMu.Lock()
	defer checkerPrepareMu.Unlock()
	// Run an `uv sync` on the code, so the compile task just copies the file (uv has set compiled=false)

	lang := c.Language()
	resp, err := tasks.CompileTask(ctx, c.mgr, &tasks.CompileRequest{
		File: &eval.BucketFile{
			Bucket:   datastore.BucketTypeCheckers,
			Filename: fmt.Sprintf("%d.bin", c.pb.ID),
			Mode:     0777,
		},
		CodeFiles: map[string][]byte{
			lang.SourceName(c.filename): c.code,
		}, HeaderFiles: map[string][]byte{
			"/box/pyproject.toml": c.uvToml,
			"/box/uv.lock":        c.uvLock,
		},
		Lang: lang,

		OriginalFilename: c.filename,

		Store: c.store,
	}, c.Logger)
	if err != nil {
		return "Couldn't sync uv", err
	}

	if !resp.Success {
		return fmt.Sprintf("Output:\n%s\nOther:\n%s", resp.Output, resp.Other), kilonova.Statusf(400, "Invalid helper code")
	}

	//c.Logger.InfoContext(ctx, "Synced AI checker", slog.Duration("duration", time.Duration(resp.Stats.Time*float64(time.Second))))

	return "", nil
}

func (c *aiChecker) Cleanup(ctx context.Context) error {
	return nil
}

func (c *aiChecker) CodeFilename() string {
	return c.filename
}

func (c *aiChecker) RunChecker(ctx context.Context, subtestID int, testID int) (string, decimal.Decimal) {
	checkerPrepareMu.RLock()
	defer checkerPrepareMu.RUnlock()

	req := &eval.Box2Request{
		InputBucketFiles: map[string]*eval.BucketFile{
			"/box/submission.csv": {
				Bucket:   datastore.BucketTypeSubtests,
				Filename: strconv.Itoa(subtestID),
				Mode:     0666,
			},
			"/box/metric.txt": {
				Bucket:   datastore.BucketTypeTests,
				Filename: strconv.Itoa(testID) + ".in",
				Mode:     0666,
			},
			"/box/ground_truth.csv": {
				Bucket:   datastore.BucketTypeTests,
				Filename: strconv.Itoa(testID) + ".out",
				Mode:     0666,
			},
			c.Language().CompiledName(c.filename): {
				Bucket:   datastore.BucketTypeCheckers,
				Filename: fmt.Sprintf("%d.bin", c.pb.ID),
				Mode:     0666,
			},
		},
		InputByteFiles: map[string]*eval.ByteFile{
			//"/box/contestant.txt": {
			//	Data: job.c.subCode,
			//	Mode: 0666,
			//},
			"/box/pyproject.toml": {
				Data: c.uvToml,
				Mode: 0666,
			},
			"/box/uv.lock": {
				Data: c.uvLock,
				Mode: 0666,
			},
		},
		OutputByteFiles: []string{"/box/verdict.out", "/box/verdict.err"},

		Command: c.Language().RunCommand([]string{c.Language().ExecuteName(c.filename)}, checkerMemoryLimit),

		RunConfig: &eval.RunConfig{
			MemoryLimit: checkerMemoryLimit,
			OutputPath:  "/box/verdict.out",
			StderrPath:  "/box/verdict.err",

			WallTimeLimit: 20,

			EnableInternet: true, // TODO: Remove this hack

			Directories: c.Language().Mounts(),
		},
	}

	resp, err := c.mgr.RunBox2(ctx, req, checkerMemoryLimit)
	if resp == nil || err != nil {
		return "Couldn't run AI checker", decimal.Zero
	}

	percentage, output := tasks.ParseStandardManagerOutput(resp)
	return output, percentage
}

func (c *aiChecker) Language() language.GraderLang {
	return language.Uv()
}
