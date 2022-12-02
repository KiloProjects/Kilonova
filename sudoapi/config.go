package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

type ConfigUpdate struct {
	DefaultLanguage *string `json:"default_lang"`

	Grader *bool `json:"grader"`
	Signup *bool `json:"signup"`
	Pastes *bool `json:"pastes"`
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
	if err := config.Save(); err != nil {
		zap.S().Error(err)
		return WrapError(err, "Couldn't update config")
	}
	return nil
}