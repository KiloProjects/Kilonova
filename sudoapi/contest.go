package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) CreateContest(ctx context.Context, name string, author *UserBrief) (int, *StatusError) {
	if author == nil {
		return -1, ErrMissingRequired
	}
	id, err := s.db.CreateContest(ctx, name)
	if err != nil {
		return -1, WrapError(err, "Couldn't create contest")
	}
	if err := s.db.AddContestEditor(ctx, id, author.ID); err != nil {
		zap.S().Warn(err)
		return id, WrapError(err, "Couldn't add author to contest editors")
	}
	return id, nil
}

func (s *BaseAPI) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) *kilonova.StatusError {
	if err := s.db.UpdateContest(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest")
	}
	return nil
}

func (s *BaseAPI) UpdateContestProblems(ctx context.Context, id int, list []int) *StatusError {
	if err := s.db.UpdateContestProblems(ctx, id, list); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update contest problems")
	}
	return nil
}

func (s *BaseAPI) DeleteContest(ctx context.Context, id int) *StatusError {
	if err := s.db.DeleteContest(ctx, id); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete contest")
	}
	s.LogUserAction(ctx, "Removed contest %d", id)
	return nil
}

func (s *BaseAPI) Contest(ctx context.Context, id int) (*kilonova.Contest, *StatusError) {
	contest, err := s.db.Contest(ctx, id)
	if err != nil || contest == nil {
		return nil, Statusf(400, "Contest not found")
	}
	return contest, nil
}

func (s *BaseAPI) VisibleContests(ctx context.Context, userID int) ([]*kilonova.Contest, *StatusError) {
	contests, err := s.db.VisibleContests(ctx, userID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) ProblemContests(ctx context.Context, problemID int) ([]*kilonova.Contest, *StatusError) {
	contests, err := s.db.ContestsByProblem(ctx, problemID)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch contests")
	}
	return contests, nil
}

func (s *BaseAPI) CanJoinContest(c *kilonova.Contest) bool {
	return c.PublicJoin && time.Now().Before(c.StartTime)
}
