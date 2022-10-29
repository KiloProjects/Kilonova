package sudoapi

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
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
		// TODO: Disallow removing own admin once callee is added to context
	}

	// TODO: Once admin/proposer toggle time are added,
	// Make sure user keeps proposer rights on admin remove (if valid)

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

	// TODO: Disallow removing own proposer rank once callee is added to context
	if toSet {
		s.LogUserAction(ctx, "Promoted user #%d to proposer status", userID)
	} else {
		s.LogUserAction(ctx, "Demoted user #%d from proposer status", userID)
	}

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Proposer: &toSet})
}

func (s *BaseAPI) LogSytemAction(ctx context.Context, msg string) {
	if _, err := s.db.CreateAuditLog(ctx, msg, nil, true); err != nil {
		zap.S().Warn(WrapError(err, "Couldn't register systm action to audit log"))
		zap.S().Info("Action: %q", msg)
	}
}

func (s *BaseAPI) LogUserAction(ctx context.Context, msg string, args ...any) {
	if util.UserBriefContext(ctx) == nil {
		zap.S().Warn("Empty user provided")
		zap.S().Infof("Action: %q", msg)
		return
	}

	msg = fmt.Sprintf(msg, args...)
	if _, err := s.db.CreateAuditLog(ctx, msg, &util.UserBriefContext(ctx).ID, false); err != nil {
		zap.S().Warn(WrapError(err, "Couldn't register user action to audit log"))
		zap.S().Infof("Action (by user #%d): %q", util.UserBriefContext(ctx).ID, msg)
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
