package sudoapi

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

// Problem stuff

// When editing Problem, please edit ScoredProblem as well
func (s *BaseAPI) Problem(ctx context.Context, id int) (*kilonova.Problem, *StatusError) {
	problem, err := s.db.Problem(ctx, id)
	if err != nil || problem == nil {
		return nil, WrapError(err, "Problem not found")
	}
	return problem, nil
}

func (s *BaseAPI) ScoredProblem(ctx context.Context, problemID int, userID int) (*kilonova.ScoredProblem, *StatusError) {
	problem, err := s.db.ScoredProblem(ctx, problemID, userID)
	if err != nil || problem == nil {
		return nil, WrapError(err, "Problem not found")
	}
	return problem, nil
}

// `updater` is an optional parameter, specifying the author of the change. It handles the visibility change option.
// Visibility is the only parameter that can not be updated by a mere problem editor, it requires admin permissions!
// Please note that, if updater is not specified, the function won't attempt to ensure correct permissions for visibility.
func (s *BaseAPI) UpdateProblem(ctx context.Context, id int, args kilonova.ProblemUpdate, updater *kilonova.UserBrief) *StatusError {
	if args.Name != nil && *args.Name == "" {
		return Statusf(400, "Title can't be empty!")
	}
	if updater != nil && args.Visible != nil && !s.IsAdmin(updater) {
		return Statusf(403, "User can't update visibility!")
	}
	if args.ScoringStrategy != kilonova.ScoringTypeNone && args.ScoringStrategy != kilonova.ScoringTypeMaxSub && args.ScoringStrategy != kilonova.ScoringTypeSumSubtasks {
		return Statusf(400, "Invalid scoring strategy!")
	}

	if err := s.db.UpdateProblem(ctx, id, args); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem")
	}

	// TODO: How do we do it now?
	// // Log in background, do not lock
	// go func(id int, args kilonova.ProblemUpdate) {
	// 	pb, err := s.Problem(context.Background(), id)
	// 	if err != nil {
	// 		zap.S().Warn(err)
	// 	}
	// 	if pb.Visible && args.Description != nil && !updater.Admin {
	// 		s.LogUserAction(context.WithValue(context.Background(), util.UserKey, updater), "Updated problem #%d (%s) description while visible", pb.ID, pb.Name)
	// 	}
	// }(id, args)

	return nil
}

func (s *BaseAPI) DeleteProblem(ctx context.Context, problem *kilonova.Problem) *StatusError {
	if err := s.db.DeleteProblem(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete problem")
	}
	s.LogUserAction(ctx, "Removed problem #%d: %s", problem.ID, problem.Name)
	return nil
}

// When editing Problems, please edit ScoredProblems as well
func (s *BaseAPI) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, *StatusError) {
	problems, err := s.db.Problems(ctx, filter)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

func (s *BaseAPI) ScoredProblems(ctx context.Context, filter kilonova.ProblemFilter, user *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	uid := -1
	if user != nil {
		uid = user.ID
	}
	problems, err := s.db.ScoredProblems(ctx, filter, uid)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

// When editing ContestProblems, please edit ScoredContestProblems as well
func (s *BaseAPI) ContestProblems(ctx context.Context, contest *kilonova.Contest, lookingUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if !s.CanViewContestProblems(ctx, lookingUser, contest) {
		return nil, Statusf(403, "User can't view contest problems")
	}
	userID := -1
	if lookingUser != nil {
		userID = lookingUser.ID
	}
	problems, err := s.db.ScoredContestProblems(ctx, contest.ID, userID)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

func (s *BaseAPI) ContestProblem(ctx context.Context, contest *kilonova.Contest, lookingUser *kilonova.UserBrief, problemID int) (*kilonova.Problem, *StatusError) {
	if !s.CanViewContestProblems(ctx, lookingUser, contest) {
		return nil, Statusf(403, "User can't view contest problem")
	}
	problems, err := s.db.ContestProblems(ctx, contest.ID)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problems")
	}
	for _, problem := range problems {
		problem := problem
		if problem.ID == problemID {
			return problem, nil
		}
	}
	return nil, WrapError(ErrNotFound, "Problem isn't in contest")
}

func (s *BaseAPI) SolvedProblems(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if user == nil {
		return []*kilonova.ScoredProblem{}, nil
	}
	ids, err := s.db.SolvedProblemIDs(ctx, user.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get solved problem IDs")
	}
	return s.hydrateProblemIDs(ctx, ids, user), nil
}

func (s *BaseAPI) AttemptedProblems(ctx context.Context, user *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if user == nil {
		return []*kilonova.ScoredProblem{}, nil
	}
	ids, err := s.db.AttemptedProblemsIDs(ctx, user.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get attempted problem IDs")
	}
	return s.hydrateProblemIDs(ctx, ids, user), nil
}

func (s *BaseAPI) hydrateProblemIDs(ctx context.Context, ids []int, user *kilonova.UserBrief) []*kilonova.ScoredProblem {
	var pbs = make([]*kilonova.ScoredProblem, 0, len(ids))
	for _, id := range ids {
		pb, err := s.ScoredProblem(ctx, id, user.ID)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				zap.S().Warnf("Couldn't get solved problem %d: %s\n", id, err)
			}
		} else {
			pbs = append(pbs, pb)
		}
	}
	return pbs
}

func (s *BaseAPI) insertProblem(ctx context.Context, problem *kilonova.Problem, authorID int) (int, *StatusError) {
	err := s.db.CreateProblem(ctx, problem, authorID)
	if err != nil {
		return -1, WrapError(err, "Couldn't create problem")
	}
	return problem.ID, nil
}

// CreateProblem is the simple way of creating a new problem. Just provide a title, an author and the type of input.
// The other stuff will be automatically set for sensible defaults.
func (s *BaseAPI) CreateProblem(ctx context.Context, title string, author *UserBrief, consoleInput bool) (int, *StatusError) {
	problem := &kilonova.Problem{
		Name:         title,
		ConsoleInput: consoleInput,
	}

	return s.insertProblem(ctx, problem, author.ID)
}

func (s *BaseAPI) AddProblemEditor(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripProblemAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add problem editor: sanity strip failed")
	}
	if err := s.db.AddProblemEditor(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add problem editor")
	}
	return nil
}

func (s *BaseAPI) AddProblemViewer(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripProblemAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add problem viewer: sanity strip failed")
	}
	if err := s.db.AddProblemViewer(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't add problem viewer")
	}
	return nil
}

func (s *BaseAPI) StripProblemAccess(ctx context.Context, pbid int, uid int) *StatusError {
	if err := s.db.StripProblemAccess(ctx, pbid, uid); err != nil {
		return WrapError(err, "Couldn't strip problem access")
	}
	return nil
}

func (s *BaseAPI) ProblemEditors(ctx context.Context, pbid int) ([]*kilonova.UserBrief, *StatusError) {
	users, err := s.db.ProblemEditors(ctx, pbid)
	if err != nil {
		return []*kilonova.UserBrief{}, WrapError(err, "Couldn't get problem editors")
	}
	return mapUsersBrief(users), nil
}

func (s *BaseAPI) ProblemViewers(ctx context.Context, pbid int) ([]*kilonova.UserBrief, *StatusError) {
	users, err := s.db.ProblemViewers(ctx, pbid)
	if err != nil {
		return []*kilonova.UserBrief{}, WrapError(err, "Couldn't get problem viewers")
	}
	return mapUsersBrief(users), nil
}

type ProblemStatistics struct {
	NumSolved    int `json:"num_solved"`
	NumAttempted int `json:"num_attempted"`

	SizeLeaderboard   *Submissions `json:"size_leaderboard"`
	MemoryLeaderboard *Submissions `json:"memory_leaderboard"`
	TimeLeaderboard   *Submissions `json:"time_leaderboard"`
}

func (s *BaseAPI) ProblemStatistics(ctx context.Context, problem *kilonova.Problem, lookingUser *UserBrief) (*ProblemStatistics, *StatusError) {
	if ok := s.IsProblemFullyVisible(lookingUser, problem); !ok {
		return nil, Statusf(401, "Looking user must be full problem viewer")
	}

	numSolved, err := s.db.ProblemStatisticsNumSolved(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get number of users that solved problem")
	}

	numAttempted, err := s.db.ProblemStatisticsNumAttempted(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get number of users that attempted problem")
	}

	sizeRaw, err := s.db.ProblemStatisticsSize(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by size")
	}
	size, err1 := s.fillSubmissions(ctx, -1, sizeRaw, lookingUser)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by size")
	}

	memoryRaw, err := s.db.ProblemStatisticsMemory(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by memory")
	}
	memory, err1 := s.fillSubmissions(ctx, -1, memoryRaw, lookingUser)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by memory")
	}

	timeRaw, err := s.db.ProblemStatisticsTime(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by time")
	}
	time, err1 := s.fillSubmissions(ctx, -1, timeRaw, lookingUser)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by time")
	}

	return &ProblemStatistics{
		NumSolved:    numSolved,
		NumAttempted: numAttempted,

		SizeLeaderboard:   size,
		MemoryLeaderboard: memory,
		TimeLeaderboard:   time,
	}, nil
}
