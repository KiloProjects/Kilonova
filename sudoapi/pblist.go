package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) ProblemList(ctx context.Context, id int) (*kilonova.ProblemList, *StatusError) {
	pblist, err := s.db.ProblemList(ctx, id)
	if err != nil || pblist == nil {
		return nil, WrapError(ErrNotFound, "Problem list not found")
	}
	return pblist, nil
}

func (s *BaseAPI) ProblemLists(ctx context.Context, filter kilonova.ProblemListFilter) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ProblemLists(ctx, filter)
	if err != nil {
		return nil, ErrUnknownError
	}
	return pblists, nil
}

func (s *BaseAPI) CreateProblemList(ctx context.Context, pblist *kilonova.ProblemList) *StatusError {
	if err := s.db.CreateProblemList(ctx, pblist); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create problem list")
	}
	return nil
}

func (s *BaseAPI) UpdateProblemList(ctx context.Context, id int, upd kilonova.ProblemListUpdate) *StatusError {
	if err := s.db.UpdateProblemList(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem list metadata")
	}
	return nil
}

func (s *BaseAPI) UpdateProblemListProblems(ctx context.Context, id int, list []int) *StatusError {
	if err := s.db.UpdateProblemListProblems(ctx, id, list); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem list problems")
	}
	return nil
}

func (s *BaseAPI) DeleteProblemList(ctx context.Context, id int) *StatusError {
	if err := s.db.DeleteProblemList(ctx, id); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete problem list")
	}
	return nil
}
