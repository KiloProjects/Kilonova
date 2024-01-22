package grader

import (
	"context"
	"errors"
	"path"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	waitingSubs   = kilonova.SubmissionFilter{Status: kilonova.StatusWaiting, Ascending: true, Limit: 41}
	reevalingSubs = kilonova.SubmissionFilter{Status: kilonova.StatusReevaling, Ascending: true, Limit: 6}
	workingUpdate = kilonova.SubmissionUpdate{Status: kilonova.StatusWorking}

	// If future me is running multiple grader handlers
	// I have only one question: "Why are you doing it?"
	openAction   sync.Once
	closeAction  sync.Once
	logFile      *lumberjack.Logger
	graderLogger *zap.SugaredLogger
)

type Handler struct {
	ctx   context.Context
	sChan chan *kilonova.Submission
	base  *sudoapi.BaseAPI

	wakeChan chan struct{}
}

func NewHandler(ctx context.Context, base *sudoapi.BaseAPI) (*Handler, *kilonova.StatusError) {
	ch := make(chan *kilonova.Submission, 1)
	wCh := make(chan struct{}, 1)

	openAction.Do(func() {
		logFile = &lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "grader.log"),
			MaxSize:  20, //MB
			Compress: true,
		}
		graderLogger = zap.New(kilonova.GetZapCore(config.Common.Debug, false, logFile), zap.AddCaller()).Sugar()
	})

	return &Handler{ctx, ch, base, wCh}, nil
}

func (h *Handler) Wake() {
	select {
	case h.wakeChan <- struct{}{}:
	default:
	}
}

func (h *Handler) ScheduleSubmission(runner eval.BoxScheduler, sub *kilonova.Submission) error {
	var subRunner eval.BoxScheduler
	if sub.SubmissionType == kilonova.EvalTypeClassic {
		r, err := runner.SubRunner(h.ctx, runner.NumConcurrent())
		if err != nil {
			return err
		} else {
			subRunner = r
		}
	} else {
		r, err := runner.SubRunner(h.ctx, 1)
		if err != nil {
			return err
		} else {
			subRunner = r
		}
	}
	if err := h.base.UpdateSubmission(h.ctx, sub.ID, workingUpdate); err != nil {
		return err
	}
	go func(sub *kilonova.Submission, r eval.BoxScheduler) {
		defer r.Close(h.ctx)
		if err := executeSubmission(h.ctx, h.base, r, sub); err != nil {
			zap.S().Warn("Couldn't run submission: ", err)
		}
	}(sub, subRunner)
	return nil
}

func (h *Handler) handle(runner eval.BoxScheduler) error {
	for {
		select {
		case <-h.ctx.Done():
			if !errors.Is(h.ctx.Err(), context.Canceled) {
				return h.ctx.Err()
			}
			return nil
		case _, more := <-h.wakeChan:
			if !more {
				return nil
			}
			var rewake bool

			subs, err := h.base.RawSubmissions(h.ctx, waitingSubs)
			if err != nil {
				zap.S().Warn(err)
			} else if len(subs) > 0 {
				graderLogger.Infof("Found %d submissions", len(subs))
				if len(subs) > 40 {
					subs = subs[:40]
					rewake = true
				}
				for _, sub := range subs {
					if err := h.ScheduleSubmission(runner, sub); err != nil {
						zap.S().Warn(err)
					}
				}
			}

			reevalQueue, err := h.base.RawSubmissions(h.ctx, reevalingSubs)
			if err != nil {
				zap.S().Warn(err)
			} else if len(reevalQueue) > 0 {
				graderLogger.Infof("Found %d submissions for reevaluation", len(reevalQueue))
				if len(reevalQueue) > 5 {
					reevalQueue = reevalQueue[:5]
					rewake = true
				}
				for _, sub := range reevalQueue {
					if err := h.base.ResetSubmission(h.ctx, sub.ID); err != nil {
						zap.S().Warn("Couldn't reset submission: ", err)
						continue
					}
					sub2, err := h.base.RawSubmission(h.ctx, sub.ID)
					if err != nil {
						zap.S().Warn("Error refetching submission for reeval: ", err)
						sub2 = sub
					}
					if err := h.ScheduleSubmission(runner, sub2); err != nil {
						zap.S().Warn(err)
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

func (h *Handler) Start() error {
	runner, err := getAppropriateRunner(h.base)
	if err != nil {
		return err
	}

	h.base.RegisterGrader(h) // To allow waking from outside grader

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

	defer runner.Close(h.ctx)
	zap.S().Info("Connected to eval")

	if err = h.handle(runner); err != nil {
		zap.S().Error("Handling error:", zap.Error(err))
		return err
	}
	return nil
}

func (h *Handler) Close() {
	closeAction.Do(func() {
		if err := logFile.Close(); err != nil {
			zap.S().Warn("Error closing grader.log:", err)
		}
		close(h.wakeChan)
	})
}
