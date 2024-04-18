package checkers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
	"time"

	_ "embed"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	checkerMemoryLimit = 512 * 1024
)

var checkerPrepareMu sync.RWMutex

var _ Checker = &customChecker{}

//go:embed checkerdata/testlib.h
var testlibFile []byte

type customCheckerInput struct {
	c    *customChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader
}

type checkerResult struct {
	Percentage decimal.Decimal
	Output     string
}

// note that customChecker should not be used between submissions
type customChecker struct {
	mgr      eval.BoxScheduler
	pb       *kilonova.Problem
	filename string
	code     []byte
	subCode  []byte

	// lastUpdatedAt is used to check if the checker needs to be recompiled, in the case it exists
	lastUpdatedAt time.Time

	Logger *zap.SugaredLogger

	legacy bool
}

// Prepare compiles the checker for the submission
func (c *customChecker) Prepare(ctx context.Context) (string, error) {
	var shouldCompile bool
	stat, err := datastore.GetBucket(datastore.BucketTypeCheckers).Stat(fmt.Sprintf("%d.bin", c.pb.ID))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warn("Checker stat error:", err)
		}
		shouldCompile = true
	} else if stat.ModTime().Before(c.lastUpdatedAt) {
		shouldCompile = true
	}

	if !shouldCompile {
		c.Logger.Infof("Using cached checker")
		return "", nil
	}

	zap.S().Debugf("Compiling checker for problem %d", c.pb.ID)
	c.Logger.Infof("Compiling checker for problem %d", c.pb.ID)
	checkerPrepareMu.Lock()
	defer checkerPrepareMu.Unlock()

	resp, err := tasks.GetCompileTask(c.Logger).Run(ctx, c.mgr, 0, &tasks.CompileRequest{
		ID: -c.pb.ID,
		CodeFiles: map[string][]byte{
			eval.Langs[eval.GetLangByFilename(c.filename)].SourceName: c.code,
		}, HeaderFiles: map[string][]byte{
			"/box/testlib.h": testlibFile,
		},
		Lang: eval.GetLangByFilename(c.filename),
	})
	if err != nil {
		return "Couldn't compile checker", err
	}

	if !resp.Success {
		return fmt.Sprintf("Output:\n%s\nOther:\n%s", resp.Output, resp.Other), kilonova.Statusf(400, "Invalid helper code")
	}

	c.Logger.Infof("Checker compilation time: %d ms", int(resp.Stats.Time*1000))

	return "", nil
}

func (c *customChecker) RunChecker(ctx context.Context, pOut, cIn, cOut io.Reader) (string, decimal.Decimal) {
	checkerPrepareMu.RLock()
	defer checkerPrepareMu.RUnlock()
	var out checkerResult

	var task eval.Task[customCheckerInput, checkerResult] = standardCheckerTask
	if c.legacy {
		task = legacyCheckerTask
	}

	resp, err := task.Run(ctx, c.mgr, checkerMemoryLimit, &customCheckerInput{
		c:    c,
		pOut: pOut,
		cIn:  cIn,
		cOut: cOut,
	})
	if err != nil || resp == nil {
		return ErrOut, decimal.Zero
	}

	out = *resp

	return out.Output, out.Percentage
}

func (c *customChecker) Cleanup(_ context.Context) error {
	// Don't clean checkers all the time anymore
	return nil // eval.CleanCompilation(-c.sub.ID)
}

func NewLegacyCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, filename string, code []byte, subCode []byte, lastUpdatedAt time.Time) Checker {
	return &customChecker{mgr, pb, filename, code, subCode, lastUpdatedAt, logger, true}
}

func NewStandardCustomChecker(mgr eval.BoxScheduler, logger *zap.SugaredLogger, pb *kilonova.Problem, filename string, code []byte, subCode []byte, lastUpdatedAt time.Time) Checker {
	return &customChecker{mgr, pb, filename, code, subCode, lastUpdatedAt, logger, false}
}
