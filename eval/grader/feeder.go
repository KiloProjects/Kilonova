package grader

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var (
	waitingSubs   = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting, Ascending: true, Limit: 41}
	reevalingSubs = kilonova.SubmissionFilter{Status: kilonova.StatusReevaling, Ascending: true, Limit: 6}
	workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}

	// grader.log is process-wide: one Backend and possibly one Handler per tenant share it.
	openAction   sync.Once
	closeAction  sync.Once
	logFile      io.Writer
	graderLogger *slog.Logger
)

type Handler struct {
	ctx   context.Context
	sChan chan *kilonova.Submission
	base  *sudoapi.BaseAPI

	wakeChan chan struct{}

	runner eval.BoxScheduler

	closeOnce sync.Once
}

// initLogger opens grader.log once per process; Backend and every Handler share it.
func initLogger() {
	openAction.Do(func() {
		// 80 MB: most rows are near-identical so they compress very small.
		logFile = config.LogWriter("grader.log", 80)
		lvl := slog.LevelInfo
		if kilonova.DebugMode() {
			lvl = slog.LevelDebug
		}
		graderLogger = slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
			AddSource: true,
			Level:     lvl,
		}))
	})
}

// NewHandler is one tenant's grader: it feeds base's waiting submissions to the
// shared Backend passed to Start.
func NewHandler(ctx context.Context, base *sudoapi.BaseAPI) (*Handler, error) {
	initLogger()
	return &Handler{ctx: ctx, sChan: make(chan *kilonova.Submission, 1), base: base, wakeChan: make(chan struct{}, 1)}, nil
}

func (h *Handler) Wake() {
	select {
	case h.wakeChan <- struct{}{}:
	default:
	}
}

func (h *Handler) runSubmission(runner eval.BoxScheduler, langMgr eval.LanguageManager, sub *kilonova.Submission) error {
	// TODO: Do this in a smarter way
	// var numConc int64
	// if sub.SubmissionType == kilonova.EvalTypeClassic {
	// 	numConc = runner.NumConcurrent()
	// } else {
	// 	numConc = 1
	// }
	// runner, err := runner.SubRunner(h.ctx, numConc)
	// if err != nil {
	// 	return err
	// }

	if err := h.base.UpdateSubmission(h.ctx, sub.ID, workingUpdate); err != nil {
		return err
	}
	// go func(sub *kilonova.Submission, runner eval.BoxScheduler) {
	// defer runner.Close(h.ctx)
	if err := executeSubmission(h.ctx, h.base, runner, langMgr, sub); err != nil {
		slog.WarnContext(h.ctx, "Couldn't run submission", slog.Any("err", err))
	}
	// }(sub, runner)
	return nil
}

func (h *Handler) ScheduleSubmission(runner eval.BoxScheduler, langMgr eval.LanguageManager, sub *kilonova.Submission) error {
	return h.runSubmission(runner, langMgr, sub)
}

func (h *Handler) handle(runner eval.BoxScheduler, langMgr eval.LanguageManager) error {
	for {
		select {
		case <-h.ctx.Done():
			return h.ctx.Err()
		case _, more := <-h.wakeChan:
			if !more {
				return nil
			}
			var rewake bool

			subs, err := h.base.RawSubmissions(h.ctx, waitingSubs)
			if err != nil {
				slog.WarnContext(h.ctx, "Couldn't get waiting submissions", slog.Any("err", err))
			} else if len(subs) > 0 {
				graderLogger.InfoContext(h.ctx, "Found waiting submissions", slog.Int("count", len(subs)))
				if len(subs) > 40 {
					subs = subs[:40]
					rewake = true
				}
				for _, sub := range subs {
					if err := h.ScheduleSubmission(runner, langMgr, sub); err != nil {
						slog.WarnContext(h.ctx, "Couldn't schedule waiting submission", slog.Any("err", err))
					}
				}
			}

			reevalQueue, err := h.base.RawSubmissions(h.ctx, reevalingSubs)
			if err != nil {
				slog.WarnContext(h.ctx, "Could not get raw submissions", slog.Any("err", err))
			} else if len(reevalQueue) > 0 {
				graderLogger.InfoContext(h.ctx, "Found submissions to reevaluate", slog.Int("count", len(reevalQueue)))
				if len(reevalQueue) > 5 {
					reevalQueue = reevalQueue[:5]
					rewake = true
				}
				for _, sub := range reevalQueue {
					if err := h.base.ResetSubmission(h.ctx, sub.ID); err != nil {
						slog.WarnContext(h.ctx, "Couldn't reset submission", slog.Any("err", err))
						continue
					}
					sub2, err := h.base.RawSubmission(h.ctx, sub.ID)
					if err != nil {
						slog.WarnContext(h.ctx, "Error refetching submission for reeval", slog.Any("err", err))
						sub2 = sub
					}
					if err := h.ScheduleSubmission(runner, langMgr, sub2); err != nil {
						slog.WarnContext(h.ctx, "Couldn't schedule reevaluation submission", slog.Any("err", err))
					}
				}
			}

			if rewake {
				// Try to instantly continue working on the queue
				h.Wake()
			}
		}
	}
}

// Start runs this tenant's feed loop on backend until ctx ends. It does not close
// backend, which other tenants may share.
func (h *Handler) Start(backend *Backend) error {
	runner, langMgr := backend.runner(h.base.DataStore()), backend.langs
	h.runner = runner

	h.base.RegisterGrader(h) // To allow waking from outside grader
	h.base.RegisterLanguageManager(langMgr)

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				h.Wake()
			case <-h.ctx.Done():
				return
			}
		}
	}()

	slog.InfoContext(h.ctx, "Connected to eval")

	if err := h.handle(runner, langMgr); err != nil {
		slog.ErrorContext(h.ctx, "Handling error", slog.Any("err", err))
		return err
	}
	return nil
}

func (h *Handler) Close() {
	h.closeOnce.Do(func() { close(h.wakeChan) })
	closeAction.Do(func() {
		// Nothing to close when logs go to stdout (KN_LOG_FILE=false).
		if c, ok := logFile.(io.Closer); ok {
			if err := c.Close(); err != nil {
				slog.WarnContext(h.ctx, "Error closing grader.log", slog.Any("err", err))
			}
		}
	})
}
