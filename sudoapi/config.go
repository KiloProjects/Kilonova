package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

type ConfigUpdate struct {
	DefaultLanguage *string `json:"default_lang"`

	TestMaxMem   *int   `json:"test_max_mem"`
	GlobalMaxMem *int64 `json:"global_max_mem"`
	NumWorkers   *int   `json:"num_workers"`
}

func (s *BaseAPI) UpdateConfig(ctx context.Context, upd ConfigUpdate) *StatusError {
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
	if err := config.Save(); err != nil {
		zap.S().Error(err)
		return WrapError(err, "Couldn't update config. This is *very* bad")
	}
	return nil
}
