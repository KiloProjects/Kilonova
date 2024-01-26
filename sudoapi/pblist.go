package sudoapi

import (
	"context"
	"errors"
	"slices"

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
	pbs, err := s.ScoredProblems(ctx, kilonova.ProblemFilter{IDs: ids, LookingUser: lookingUser, Look: true}, lookingUser, lookingUser)
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

func (s *BaseAPI) ProblemLists(ctx context.Context, filter kilonova.ProblemListFilter) ([]*kilonova.ProblemList, *StatusError) {
	pblists, err := s.db.ProblemLists(ctx, filter)
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
		if !errors.Is(err, kilonova.ErrNoUpdates) {
			zap.S().Warn(err)
		}
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

func (s *BaseAPI) NumSolvedFromPblists(ctx context.Context, listIDs []int, user *kilonova.UserBrief) (map[int]int, *StatusError) {
	if user == nil {
		vals := make(map[int]int)
		for _, id := range listIDs {
			vals[id] = -1
		}
		return vals, nil
	}
	vals, err := s.db.NumBulkedSolvedPblistProblems(ctx, user.ID, listIDs)
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

type FullProblemList struct {
	kilonova.ProblemList
	Problems []*kilonova.ScoredProblem `json:"problems"`
	SubLists []*FullProblemList        `json:"problem_lists"`

	SolvedCount int  `json:"solved_count"`
	DepthLevel  int  `json:"depth_level"`
	Root        bool `json:"root"`
}

// FullProblemList returns an entire problem list DAG. The operation will probably be slow.
// Note that recursion to a "higher" level is automatically stripped
func (s *BaseAPI) FullProblemList(ctx context.Context, listID int, user *kilonova.UserBrief, lookingUser *kilonova.UserBrief) (*FullProblemList, *StatusError) {
	// Get all sublists
	lists, err := s.ProblemLists(ctx, kilonova.ProblemListFilter{ParentID: &listID})
	if err != nil {
		return nil, err
	}
	// Get all problems
	pbs, err := s.ScoredProblems(ctx, kilonova.ProblemFilter{Look: true, LookingUser: lookingUser, DeepListID: &listID}, user, lookingUser)
	if err != nil {
		return nil, err
	}
	listIDs := make([]int, 0, len(lists))
	for _, list := range lists {
		listIDs = append(listIDs, list.ID)
	}
	// Get solved count
	solvedCnt, err := s.NumSolvedFromPblists(ctx, listIDs, user)
	if err != nil {
		return nil, err
	}

	// Try to get parent list
	var firstList *kilonova.ProblemList
	for _, list := range lists {
		if list.ID == listID {
			firstList = list
			break
		}
	}
	if firstList == nil {
		return nil, kilonova.Statusf(500, "Could not load problem list tree")
	}

	return s.hydrateFullList(firstList, []int{firstList.ID}, lists, pbs, solvedCnt), nil
}

func (s *BaseAPI) hydrateFullList(list *kilonova.ProblemList, path []int, lists []*kilonova.ProblemList, pbs []*kilonova.ScoredProblem, solvedCnt map[int]int) *FullProblemList {
	l := &FullProblemList{ProblemList: *list}
	for _, pbid := range list.List {
		if idx := slices.IndexFunc(pbs, func(pb *kilonova.ScoredProblem) bool { return pb.ID == pbid }); idx >= 0 {
			l.Problems = append(l.Problems, pbs[idx])
		}
	}
	var depthLvl int
	for _, sublist := range list.SubLists {
		if slices.Contains(path, sublist.ID) { // Try to break recursion
			continue
		}
		if idx := slices.IndexFunc(lists, func(pl *kilonova.ProblemList) bool { return pl.ID == sublist.ID }); idx >= 0 {
			list := s.hydrateFullList(lists[idx], append([]int{lists[idx].ID}, path...), lists, pbs, solvedCnt)
			l.SubLists = append(l.SubLists, list)
			depthLvl = max(depthLvl, list.DepthLevel)
		}
	}
	if val, ok := solvedCnt[list.ID]; ok {
		l.SolvedCount = val
	}
	l.DepthLevel = depthLvl + 1
	if len(path) == 1 {
		l.Root = true
	}

	return l
}
