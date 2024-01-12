package sudoapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"go.uber.org/zap"
)

func (s *BaseAPI) CreateSession(ctx context.Context, uid int) (string, *StatusError) {
	sid, err := s.db.CreateSession(ctx, uid)
	if err != nil {
		zap.S().Warn("Failed to create session: ", err)
		return "", WrapError(err, "Failed to create session")
	}
	if cnt, err := s.db.RemoveOldSessions(ctx, uid); err != nil {
		zap.S().Warn("Failed to remove old sessions: ", err)
	} else if cnt > 0 {
		zap.S().Debugf("Removed %d old sessions", cnt)
	}

	return sid, nil
}

// Please note that, when unauthed, GetSession will return a session with UserID set to -1
func (s *BaseAPI) GetSession(ctx context.Context, sid string) (int, *StatusError) {
	uid, err := s.db.GetSession(ctx, sid)
	if err != nil {
		if err.Error() == "Unauthed" {
			return -1, nil
		}
		zap.S().Warn("Failed to get session: ", err)
		return -1, WrapError(err, "Failed to get session")
	}

	return uid, nil
}

// Uncached function
func (s *BaseAPI) sessionUser(ctx context.Context, sid string) (*kilonova.UserFull, *StatusError) {
	user, err := s.db.User(ctx, kilonova.UserFilter{SessionID: &sid})
	if err != nil {
		return nil, WrapError(err, "Failed to get session user")
	}
	return user.ToFull(), nil
}

// Cached function
func (s *BaseAPI) SessionUser(ctx context.Context, sid string) (*kilonova.UserFull, *StatusError) {
	user, err := s.sessionUserCache.Get(ctx, sid)
	if err != nil {
		var err1 *StatusError
		if errors.As(err, &err1) {
			return nil, err1
		}
		zap.S().Warn("session user cache error: ", err)
		return s.sessionUser(ctx, sid)
	}
	return user, nil
}

type Session = db.Session

func (s *BaseAPI) UserSessions(ctx context.Context, userID int) ([]*Session, *StatusError) {
	sessions, err := s.db.UserSessions(ctx, userID)
	if err != nil {
		return nil, WrapError(err, "Could not get user sessions")
	}
	return sessions, nil
}

func (s *BaseAPI) RemoveSession(ctx context.Context, sid string) *StatusError {
	if err := s.db.RemoveSession(ctx, sid); err != nil {
		zap.S().Warn("Failed to remove session: ", err)
		return WrapError(err, "Failed to remove session")
	}
	s.sessionUserCache.Delete(sid)
	return nil
}

func (s *BaseAPI) ExtendSession(ctx context.Context, sid string) (time.Time, *StatusError) {
	newExpiration, err := s.db.ExtendSession(ctx, sid)
	if err != nil {
		if err.Error() == "Unauthed" {
			return time.Now(), Statusf(400, "Session already expired")
		}
		return time.Now(), kilonova.WrapError(err, "Couldn't extend session")
	}
	return newExpiration, nil
}

func (s *BaseAPI) GetSessCookie(r *http.Request) string {
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return ""
	}
	return cookie.Value
}
