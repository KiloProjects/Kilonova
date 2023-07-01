package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

type ConfigUpdate struct {
	DefaultLanguage *string `json:"default_lang"`

	Grader  *bool `json:"grader"`
	Signup  *bool `json:"signup"`
	Pastes  *bool `json:"pastes"`
	AllSubs *bool `json:"all_subs"`

	CCDisclaimer *bool `json:"ccDisclaimer"`
	FrontPagePbs *bool `json:"frontPagePbs"`

	TestMaxMem   *int   `json:"test_max_mem"`
	GlobalMaxMem *int64 `json:"global_max_mem"`
	NumWorkers   *int   `json:"num_workers"`
}

func (s *BaseAPI) UpdateConfig(ctx context.Context, upd ConfigUpdate) *StatusError {
	if upd.DefaultLanguage != nil {
		config.Common.DefaultLang = *upd.DefaultLanguage
	}
	if upd.Grader != nil {
		config.Features.Grader = *upd.Grader
	}
	if upd.Signup != nil {
		SignupEnabled.Update(*upd.Signup)
	}
	if upd.Pastes != nil {
		config.Features.Pastes = *upd.Pastes
	}
	if upd.AllSubs != nil {
		config.Features.AllSubs = *upd.AllSubs
	}
	if upd.CCDisclaimer != nil {
		flg, ok := config.GetFlag[bool]("feature.frontend.cc_disclaimer")
		if !ok {
			zap.S().Warn("CC disclaimer flag not present")
		}
		flg.Update(*upd.CCDisclaimer)
	}
	if upd.FrontPagePbs != nil {
		config.Features.FrontPagePbs = *upd.FrontPagePbs
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
