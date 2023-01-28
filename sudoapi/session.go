package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) CreateSession(ctx context.Context, uid int) (string, *StatusError) {
	sid, err := s.db.CreateSession(ctx, uid)
	if err != nil {
		zap.S().Warn("Failed to create session: ", err)
		return "", WrapError(err, "Failed to create session")
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

func (s *BaseAPI) RemoveSession(ctx context.Context, sid string) *StatusError {
	if err := s.db.RemoveSession(ctx, sid); err != nil {
		zap.S().Warn("Failed to remove session: ", err)
		return WrapError(err, "Failed to remove session")
	}
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
