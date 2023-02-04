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

// Returns a list of problems in the slice's order
func (s *BaseAPI) ProblemListProblems(ctx context.Context, ids []int, lookingUser *kilonova.UserBrief) ([]*kilonova.Problem, *StatusError) {
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{IDs: ids, LookingUser: lookingUser, Look: true})
	if err != nil {
		return nil, err
	}

	// Do this in order to maintain problemIDs order.
	// Necessary for problem list ordering
	available := make(map[int]*kilonova.Problem)
	for _, pb := range pbs {
		available[pb.ID] = pb
	}

	rez := []*kilonova.Problem{}
	for _, pb := range ids {
		if val, ok := available[pb]; ok {
			rez = append(rez, val)
		}
	}
	return rez, nil
}

func (s *BaseAPI) ProblemLists(ctx context.Context, root bool) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ProblemLists(ctx, root)
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

func (s *BaseAPI) UpdateProblemListSublists(ctx context.Context, id int, listIDs []int) *StatusError {
	if err := s.db.UpdateProblemListSublists(ctx, id, listIDs); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem list nested lists")
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

func (s *BaseAPI) NumSolvedFromPblist(ctx context.Context, listID int, userID int) (int, *StatusError) {
	num, err := s.db.NumSolvedPblistProblems(ctx, listID, userID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get number of solved problems")
	}
	return num, nil
}
