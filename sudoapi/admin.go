package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

func (s *BaseAPI) ResetWaitingSubmissions(ctx context.Context) *StatusError {
	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{Status: kilonova.StatusWorking}, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}
	return nil
}

func (s *BaseAPI) ResetSubmission(ctx context.Context, id int) *StatusError {
	err := s.db.UpdateSubmission(ctx, id, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting})
	if err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submission")
	}

	var f = false
	var zero = 0
	err = s.db.UpdateSubmissionSubTests(ctx, id, kilonova.SubTestUpdate{Done: &f, Score: &zero})
	if err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update submission's subtests")
	}

	return nil
}

func (s *BaseAPI) SetAdmin(ctx context.Context, userID int, toSet bool) *StatusError {
	if userID <= 0 {
		return Statusf(400, "Invalid ID")
	}

	if !toSet {
		if userID == 1 {
			return Statusf(406, "First user must have admin rights.")
		}
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user #%d to admin status", userID)
	} else {
		s.LogUserAction(ctx, "Demoted user #%d from admin status", userID)
	}

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Admin: &toSet, Proposer: &toSet})
}

func (s *BaseAPI) SetProposer(ctx context.Context, userID int, toSet bool) *StatusError {
	user, err := s.UserBrief(ctx, userID)
	if err != nil {
		return err
	}

	if user.Admin {
		return Statusf(400, "Cannot update proposer status of an admin.")
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user #%d to proposer status", userID)
	} else {
		s.LogUserAction(ctx, "Demoted user #%d from proposer status", userID)
	}

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Proposer: &toSet})
}

type logEntry struct {
	Message  string
	AuthorID *int
	System   bool
}

func (s *BaseAPI) LogSystemAction(ctx context.Context, msg string) {
	s.logChan <- &logEntry{
		Message:  msg,
		AuthorID: nil,
		System:   true,
	}
}

func (s *BaseAPI) LogUserAction(ctx context.Context, msg string, args ...any) {
	var id *int
	if util.UserBriefContext(ctx) != nil {
		id = &util.UserBriefContext(ctx).ID
	}

	s.logChan <- &logEntry{
		Message:  fmt.Sprintf(msg, args...),
		AuthorID: id,
		System:   false,
	}
}

func (s *BaseAPI) GetAuditLogs(ctx context.Context, count int, offset int) ([]*kilonova.AuditLog, *StatusError) {
	logs, err := s.db.AuditLogs(ctx, count, offset)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch audit logs")
	}
	return logs, nil
}

func (s *BaseAPI) GetLogCount(ctx context.Context) (int, *StatusError) {
	cnt, err := s.db.AuditLogCount(ctx)
	if err != nil {
		return -1, WrapError(err, "Couldn't get audit log count")
	}
	return cnt, nil
}

func (s *BaseAPI) ingestAuditLogs(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case val := <-s.logChan:
			if _, err := s.db.CreateAuditLog(ctx, val.Message, val.AuthorID, val.System); err != nil {
				zap.S().Warn("Couldn't store audit log entry to database: ", err)
			}

			var s strings.Builder
			s.WriteString("Action")
			if val.AuthorID != nil {
				s.WriteString(fmt.Sprintf(" (by user %d)", *val.AuthorID))
			}
			if val.System {
				s.WriteString(" (system)")
			}
			s.WriteString(": " + val.Message)

			zap.S().Info(s.String())
			if config.Common.UpdatesWebhook != "" {
				vals := make(url.Values)
				vals.Add("content", s.String())
				vals.Add("username", "Kilonova Audit Log")
				_, err := http.PostForm(config.Common.UpdatesWebhook, vals)
				if err != nil {
					zap.S().Warn(err)
				}
			}
		}
	}
}
