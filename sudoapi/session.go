package sudoapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	TrueIPHeader = config.GenFlag[string]("server.listen.true_ip_header", "", "True IP header. Leave empty if not behind reverse proxy, the proxy's remote ip header (X-Forwarded-For, for example) otherwise")
)

func (s *BaseAPI) CreateSession(ctx context.Context, uid int) (string, error) {
	sid, err := s.db.CreateSession(ctx, uid)
	if err != nil {
		slog.WarnContext(ctx, "Failed to create session", slog.Any("err", err))
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	if sessions, err := s.db.RemoveOldSessions(ctx, uid); err != nil {
		slog.WarnContext(ctx, "Failed to remove old sessions", slog.Any("err", err))
	} else if len(sessions) > 0 {
		for _, sess := range sessions {
			s.sessionUserCache.Delete(sess)
		}
		slog.DebugContext(ctx, "Removed old sessions", slog.Int("count", len(sessions)))
	}

	return sid, nil
}

// Please note that, when unauthed, GetSession will return a session with UserID set to -1
func (s *BaseAPI) GetSession(ctx context.Context, sid string) (int, error) {
	uid, err := s.db.GetSession(ctx, sid)
	if err != nil {
		if err.Error() == "Unauthed" {
			return -1, nil
		}
		slog.WarnContext(ctx, "Failed to get session", slog.Any("err", err))
		return -1, fmt.Errorf("failed to get session: %w", err)
	}

	return uid, nil
}

// Uncached function
func (s *BaseAPI) sessionUser(ctx context.Context, sid string) (*kilonova.UserFull, error) {
	user, err := s.db.User(ctx, kilonova.UserFilter{SessionID: &sid})
	if err != nil {
		return nil, fmt.Errorf("failed to get session user: %w", err)
	}
	return user, nil
}

// Cached function
// Should be called only in session initialization
func (s *BaseAPI) SessionUser(ctx context.Context, sid string, r *http.Request) (*kilonova.UserFull, error) {
	user, err := s.sessionUserCache.Get(ctx, sid)
	if err != nil {
		slog.WarnContext(ctx, "session user cache error", slog.Any("err", err))
		return s.sessionUser(ctx, sid)
	}
	if user != nil {
		go func(uid int, r *http.Request) {
			ip, ua := s.GetRequestInfo(r)
			if err := s.db.UpdateSessionDevice(context.Background(), sid, user.ID, ip, &ua); err != nil {
				slog.WarnContext(ctx, "Couldn't update session device", slog.Any("err", err))
			}
		}(user.ID, r)
	}
	return user, nil
}

func (s *BaseAPI) GetRequestInfo(r *http.Request) (ip *netip.Addr, ua string) {
	hostport, err := netip.ParseAddrPort(r.RemoteAddr)
	if err == nil {
		ip2 := hostport.Addr()
		ip = &ip2
	}
	if len(TrueIPHeader.Value()) > 0 && len(r.Header.Get(TrueIPHeader.Value())) > 0 {
		addr, err := netip.ParseAddr(r.Header.Get(TrueIPHeader.Value()))
		if err != nil {
			slog.WarnContext(r.Context(), "Invalid address in reverse proxy header", slog.Any("err", err))
		} else {
			ip = &addr
		}
	}

	ua = r.Header.Get("User-Agent")
	return
}

type Session = db.Session

type SessionDevice struct {
	SessID        string    `json:"session_id"`
	CreatedAt     time.Time `json:"created_at"`
	LastCheckedAt time.Time `json:"last_checked_at"`

	IPAddr    *netip.Addr `json:"ip_addr"`
	UserAgent *string     `json:"user_agent"`
	UserID    *int        `json:"user_id"`
}

type SessionFilter = db.SessionFilter

func (s *BaseAPI) Sessions(ctx context.Context, filter *SessionFilter) ([]*Session, error) {
	sessions, err := s.db.Sessions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("could not filter sessions: %w", err)
	}
	return sessions, nil
}
func (s *BaseAPI) CountSessions(ctx context.Context, filter *SessionFilter) (int, error) {
	sessions, err := s.db.CountSessions(ctx, filter)
	if err != nil {
		return -1, fmt.Errorf("could not query session count: %w", err)
	}
	return sessions, nil
}

func (s *BaseAPI) UserSessions(ctx context.Context, userID int) ([]*Session, error) {
	sessions, err := s.db.Sessions(ctx, &db.SessionFilter{UserID: &userID})
	if err != nil {
		return nil, fmt.Errorf("could not get user sessions: %w", err)
	}
	return sessions, nil
}

func (s *BaseAPI) SessionDevices(ctx context.Context, sid string) ([]*SessionDevice, error) {
	devices, err := s.db.SessionDevices(ctx, sid)
	if err != nil {
		return nil, fmt.Errorf("could not get session devices: %w", err)
	}
	retDevices := make([]*SessionDevice, 0, len(devices))
	for _, device := range devices {
		retDevices = append(retDevices, &SessionDevice{
			SessID:        device.SessID,
			CreatedAt:     device.CreatedAt,
			LastCheckedAt: device.LastCheckedAt,

			IPAddr:    device.IPAddr,
			UserAgent: device.UserAgent,
			UserID:    device.UserID,
		})
	}
	return retDevices, nil
}

func (s *BaseAPI) RemoveSession(ctx context.Context, sid string) error {
	if err := s.db.RemoveSession(ctx, sid); err != nil {
		slog.WarnContext(ctx, "Failed to remove session", slog.Any("err", err))
		return fmt.Errorf("failed to remove session: %w", err)
	}
	s.sessionUserCache.Delete(sid)
	return nil
}

func (s *BaseAPI) RemoveUserSessions(ctx context.Context, uid int) error {
	removedSessions, err := s.db.RemoveSessions(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to remove sessions: %w", err)
	}
	for _, sess := range removedSessions {
		s.sessionUserCache.Delete(sess)
	}
	return nil
}

func (s *BaseAPI) ExtendSession(ctx context.Context, sid string) (time.Time, error) {
	newExpiration, err := s.db.ExtendSession(ctx, sid)
	if err != nil {
		if err.Error() == "Unauthed" {
			return time.Now(), Statusf(400, "Session already expired")
		}
		return time.Now(), fmt.Errorf("couldn't extend session: %w", err)
	}
	return newExpiration, nil
}

func (s *BaseAPI) GetSessCookie(r *http.Request) string {
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return ""
	}
	if cookie.Value == "guest" {
		return ""
	}
	return cookie.Value
}
