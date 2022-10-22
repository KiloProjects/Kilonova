package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
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

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Proposer: &toSet})
}
