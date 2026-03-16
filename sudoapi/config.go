package sudoapi

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
)

type ConfigUpdate struct {
	DefaultLanguage *string `json:"default_lang"`

	TestMaxMem   *int   `json:"test_max_mem"`
	GlobalMaxMem *int64 `json:"global_max_mem"`
	NumWorkers   *int   `json:"num_workers"`

	BannedHotProblems []int `json:"banned_hot_pbs"`
}

func (s *BaseAPI) UpdateConfig(ctx context.Context, upd ConfigUpdate) error {
	if upd.DefaultLanguage != nil {
		kilonova.SetDefaultLanguage(*upd.DefaultLanguage)
	}
	if upd.TestMaxMem != nil {
		config.Common.TestMaxMemKB = *upd.TestMaxMem
	}
	if upd.GlobalMaxMem != nil {
		config.Eval.GlobalMaxMem = *upd.GlobalMaxMem
	}
	if upd.NumWorkers != nil {
		config.Eval.NumConcurrent = *upd.NumWorkers
	}
	if upd.BannedHotProblems != nil {
		config.Frontend.BannedHotProblems = upd.BannedHotProblems
		defer func() {
			if err := s.db.RefreshHotProblems(ctx, upd.BannedHotProblems); err != nil {
				slog.WarnContext(ctx, "Couldn't refresh hot problems", slog.Any("err", err))
			}
		}()
	}
	if s.cmd == nil {
		slog.WarnContext(ctx, "Command not initialized, cannot persist config update")
		return nil
	}

	if cfg := s.cmd.String("config"); len(cfg) > 0 {
		if err := config.Save(cfg); err != nil {
			slog.WarnContext(ctx, "Couldn't update config", slog.Any("err", err))
			return fmt.Errorf("couldn't update config. This is *very* bad: %w", err)
		}
	} else {
		slog.WarnContext(ctx, "Couldn't update config, config file not found")
	}
	return nil
}
