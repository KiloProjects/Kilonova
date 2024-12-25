package sudoapi

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
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
		config.Common.DefaultLang = *upd.DefaultLanguage
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
				zap.S().Warn(err)
			}
		}()
	}
	if err := config.Save(); err != nil {
		zap.S().Error(err)
		return fmt.Errorf("couldn't update config. This is *very* bad: %w", err)
	}
	return nil
}
