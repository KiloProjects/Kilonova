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

func (s *BaseAPI) ProblemListByName(ctx context.Context, name string) (*kilonova.ProblemList, *StatusError) {
	pblist, err := s.db.ProblemListByName(ctx, name)
	if err != nil || pblist == nil {
		return nil, WrapError(ErrNotFound, "Problem list not found")
	}
	return pblist, nil
}

// Returns a list of problems in the slice's order
func (s *BaseAPI) ProblemListProblems(ctx context.Context, ids []int, lookingUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	pbs, err := s.ScoredProblems(ctx, kilonova.ProblemFilter{IDs: ids, LookingUser: lookingUser, Look: true}, lookingUser)
	if err != nil {
		return nil, err
	}

	// Do this in order to maintain problemIDs order.
	// Necessary for problem list ordering
	available := make(map[int]*kilonova.ScoredProblem)
	for _, pb := range pbs {
		available[pb.ID] = pb
	}

	rez := []*kilonova.ScoredProblem{}
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

func (s *BaseAPI) ProblemParentLists(ctx context.Context, problemID int, showHidable bool) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ProblemListsByProblemID(ctx, problemID, showHidable)
	if err != nil {
		return nil, ErrUnknownError
	}
	return pblists, nil
}

func (s *BaseAPI) PblistParentLists(ctx context.Context, problemListID int) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ParentProblemListsByPblistID(ctx, problemListID)
	if err != nil {
		return nil, ErrUnknownError
	}
	return pblists, nil
}

func (s *BaseAPI) PblistChildrenLists(ctx context.Context, problemListID int) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ChildrenProblemListsByPblistID(ctx, problemListID)
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

func (s *BaseAPI) NumSolvedFromPblists(ctx context.Context, listIDs []int, userID int) (map[int]int, *StatusError) {
	vals, err := s.db.NumBulkedSolvedPblistProblems(ctx, userID, listIDs)
	if err != nil || vals == nil {
		return nil, WrapError(err, "Couldn't get number of solved problems")
	}

	for _, id := range listIDs {
		if _, ok := vals[id]; !ok {
			vals[id] = 0
		}
	}

	return vals, nil
}
