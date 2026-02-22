package layout

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

type platformInfoValue struct {
	Debug            bool                `json:"debug"`
	User             *kilonova.UserBrief `json:"user"`
	Admin            bool                `json:"admin"`
	UserID           int                 `json:"user_id"`
	Language         string              `json:"language"`
	EnabledLanguages map[string]string   `json:"langs"`
	SentryDSN        *string             `json:"sentryDSN"`
	InternalVersion  string              `json:"internalVersion"`
	APIPrefix        string              `json:"apiPrefix"`
}

var buildInfo = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			val := setting.Value
			if len(val) > 7 {
				val = val[:7]
			}
			return "-" + val
		}
	}
	return ""
})

func platformInfo(ctx context.Context, enabledLanguages map[string]string) platformInfoValue {
	userBrief := user.UserBriefContext(ctx)
	var userID int = 0
	if userBrief != nil {
		userID = userBrief.ID
	}

	var sentryDSN *string
	if val := flags.SentryDSN.Value(); flags.Sentry.Value() && len(val) > 0 {
		sentryDSN = &val
	}

	return platformInfoValue{
		Debug:            config.Common.Debug,
		User:             userBrief,
		Admin:            userBrief.IsAdmin(),
		UserID:           userID,
		Language:         language(ctx),
		EnabledLanguages: enabledLanguages,
		SentryDSN:        sentryDSN,
		InternalVersion:  kilonova.Version + buildInfo(),
		APIPrefix:        "/api",
	}
}
