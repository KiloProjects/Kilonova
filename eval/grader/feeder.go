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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
func NewHandler(base *sudoapi.BaseAPI) (*Handler, error) {
	initLogger()
	return &Handler{sChan: make(chan *kilonova.Submission, 1), base: base, wakeChan: make(chan struct{}, 1)}, nil
}

func (h *Handler) Wake() {
	select {
	case h.wakeChan <- struct{}{}:
	default:
	}
}

func (h *Handler) runSubmission(ctx context.Context, runner eval.BoxScheduler, langMgr eval.LanguageManager, sub *kilonova.Submission) error {
	ctx, span := otel.Tracer("grader").Start(ctx, "RunSubmission", trace.WithNewRoot(),
		trace.WithAttributes(attribute.Int("submission.id", sub.ID)))
	defer span.End()

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

	if err := h.base.UpdateSubmission(ctx, sub.ID, workingUpdate); err != nil {
		return err
	}
	// go func(sub *kilonova.Submission, runner eval.BoxScheduler) {
	// defer runner.Close(ctx)
	if err := executeSubmission(ctx, h.base, runner, langMgr, sub); err != nil {
		slog.WarnContext(ctx, "Couldn't run submission", slog.Any("err", err))
	}
	// }(sub, runner)
	return nil
}

func (h *Handler) ScheduleSubmission(ctx context.Context, runner eval.BoxScheduler, langMgr eval.LanguageManager, sub *kilonova.Submission) error {
	return h.runSubmission(ctx, runner, langMgr, sub)
}

func (h *Handler) handle(ctx context.Context, runner eval.BoxScheduler, langMgr eval.LanguageManager) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, more := <-h.wakeChan:
			if !more {
				return nil
			}
			var rewake bool

			subs, err := h.base.RawSubmissions(ctx, waitingSubs)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get waiting submissions", slog.Any("err", err))
			} else if len(subs) > 0 {
				graderLogger.InfoContext(ctx, "Found waiting submissions", slog.Int("count", len(subs)))
				if len(subs) > 40 {
					subs = subs[:40]
					rewake = true
				}
				for _, sub := range subs {
					if err := h.ScheduleSubmission(ctx, runner, langMgr, sub); err != nil {
						slog.WarnContext(ctx, "Couldn't schedule waiting submission", slog.Any("err", err))
					}
				}
			}

			reevalQueue, err := h.base.RawSubmissions(ctx, reevalingSubs)
			if err != nil {
				slog.WarnContext(ctx, "Could not get raw submissions", slog.Any("err", err))
			} else if len(reevalQueue) > 0 {
				graderLogger.InfoContext(ctx, "Found submissions to reevaluate", slog.Int("count", len(reevalQueue)))
				if len(reevalQueue) > 5 {
					reevalQueue = reevalQueue[:5]
					rewake = true
				}
				for _, sub := range reevalQueue {
					if err := h.base.ResetSubmission(ctx, sub.ID); err != nil {
						slog.WarnContext(ctx, "Couldn't reset submission", slog.Any("err", err))
						continue
					}
					sub2, err := h.base.RawSubmission(ctx, sub.ID)
					if err != nil {
						slog.WarnContext(ctx, "Error refetching submission for reeval", slog.Any("err", err))
						sub2 = sub
					}
					if err := h.ScheduleSubmission(ctx, runner, langMgr, sub2); err != nil {
						slog.WarnContext(ctx, "Couldn't schedule reevaluation submission", slog.Any("err", err))
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
func (h *Handler) Start(ctx context.Context, backend *Backend) error {
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
			case <-ctx.Done():
				return
			}
		}
	}()

	slog.InfoContext(ctx, "Connected to eval")

	if err := h.handle(ctx, runner, langMgr); err != nil {
		slog.ErrorContext(ctx, "Handling error", slog.Any("err", err))
		return err
	}
	return nil
}

func (h *Handler) Close(ctx context.Context) {
	h.closeOnce.Do(func() { close(h.wakeChan) })
	closeAction.Do(func() {
		// Nothing to close when logs go to stdout (KN_LOG_FILE=false).
		if c, ok := logFile.(io.Closer); ok {
			if err := c.Close(); err != nil {
				slog.WarnContext(ctx, "Error closing grader.log", slog.Any("err", err))
			}
		}
	})
}
