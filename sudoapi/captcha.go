package sudoapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/dchest/captcha"
)

var (
	captchaEnabled = config.GenFlag("feature.captcha.enabled", false, "Enable prompting for CAPTCHAs")
	triggerCount   = config.GenFlag("feature.captcha.min_trigger", 10, "Maximum number of sign ups from an ip in 10 minutes before all requests trigger a CAPTCHA")
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
	if !captchaEnabled.Value() {
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
	return cnt > triggerCount.Value()
}

func (s *BaseAPI) CaptchaEnabled() bool {
	return captchaEnabled.Value()
}
