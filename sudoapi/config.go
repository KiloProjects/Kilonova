package sudoapi

import (
	"context"
	"log/slog"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

// ConfigUpdate carries the admin-editable instance settings. Everything else
// that used to live here is a KN_* environment variable now (see .env.example).
type ConfigUpdate struct {
	DefaultLanguage *string `json:"default_lang"`
	TestMaxMem      *int    `json:"test_max_mem"`

	BannedHotProblems []int `json:"banned_hot_pbs"`
}

func (s *BaseAPI) UpdateConfig(ctx context.Context, upd ConfigUpdate) error {
	if upd.DefaultLanguage != nil {
		flags.DefaultLanguage.Update(*upd.DefaultLanguage) // main.go's flag hook re-syncs kilonova.DefaultLanguage
	}
	if upd.TestMaxMem != nil {
		flags.TestMaxMemKB.Update(*upd.TestMaxMem)
	}
	if upd.BannedHotProblems != nil {
		flags.BannedHotProblems.Update(upd.BannedHotProblems)
		if err := s.db.RefreshHotProblems(ctx, upd.BannedHotProblems); err != nil {
			slog.WarnContext(ctx, "Couldn't refresh hot problems", slog.Any("err", err))
		}
	}
	return nil
}
