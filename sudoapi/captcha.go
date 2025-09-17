package sudoapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/dchest/captcha"
)

func (s *BaseAPI) CaptchaImageHandler() http.Handler {
	return captcha.Server(240, 80) // Defaults
}

func (s *BaseAPI) CheckCaptcha(id string, digits string) bool {
	return captcha.VerifyString(id, digits)
}

func (s *BaseAPI) NewCaptchaID() string {
	return captcha.New()
}

func (s *BaseAPI) MustSolveCaptcha(ctx context.Context, ip *netip.Addr) bool {
	if !flags.CaptchaEnabled.Value() {
		return false
	}
	if ip == nil {
		slog.WarnContext(ctx, "nil ip given to MustSolveCaptcha")
		return true // Err on the side of caution
	}
	cnt, err := s.db.CountSignups(ctx, *ip, time.Now().Add(-10*time.Minute))
	if err != nil {
		slog.WarnContext(ctx, "Could not count signups", slog.Any("err", err))
		return true // Err on the side of caution
	}
	return cnt > flags.CaptchaTriggerCount.Value()
}

func (s *BaseAPI) CaptchaEnabled() bool {
	return flags.CaptchaEnabled.Value()
}
