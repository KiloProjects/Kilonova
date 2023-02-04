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
}

func (s *BaseAPI) UpdateConfig(ctx context.Context, upd ConfigUpdate) *StatusError {
	if upd.DefaultLanguage != nil {
		config.Common.DefaultLang = *upd.DefaultLanguage
	}
	if upd.Grader != nil {
		config.Features.Grader = *upd.Grader
	}
	if upd.Signup != nil {
		config.Features.Signup = *upd.Signup
	}
	if upd.Pastes != nil {
		config.Features.Pastes = *upd.Pastes
	}
	if upd.AllSubs != nil {
		config.Features.AllSubs = *upd.AllSubs
	}
	if upd.CCDisclaimer != nil {
		config.Features.CCDisclaimer = *upd.CCDisclaimer
	}
	if err := config.Save(); err != nil {
		zap.S().Error(err)
		return WrapError(err, "Couldn't update config")
	}
	return nil
}
