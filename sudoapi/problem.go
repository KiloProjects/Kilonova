package sudoapi

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"math"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/shopspring/decimal"
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

// `updater` is an optional parameter, specifying the author of the change. It handles the visibility change option.
// Visibility is the only parameter that can not be updated by a mere problem editor, it requires admin permissions!
// Please note that, if updater is not specified, the function won't attempt to ensure correct permissions for visibility.
func (s *BaseAPI) UpdateProblem(ctx context.Context, id int, args kilonova.ProblemUpdate, updater *kilonova.UserBrief) *StatusError {
	if args.Name != nil && *args.Name == "" {
		return Statusf(400, "Title can't be empty!")
	}
	if updater != nil && args.Visible != nil && !updater.IsAdmin() {
		return Statusf(403, "User can't update visibility!")
	}
	if args.ScoringStrategy != kilonova.ScoringTypeNone && args.ScoringStrategy != kilonova.ScoringTypeMaxSub && args.ScoringStrategy != kilonova.ScoringTypeSumSubtasks && args.ScoringStrategy != kilonova.ScoringTypeICPC {
		return Statusf(400, "Invalid scoring strategy!")
	}

	if err := s.db.UpdateProblem(ctx, id, args); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update problem")
	}

	return nil
}

func (s *BaseAPI) ToggleDeepPbListProblems(ctx context.Context, list *kilonova.ProblemList, deep bool, upd kilonova.ProblemUpdate) *kilonova.StatusError {
	var filter kilonova.ProblemFilter
	if deep {
		filter.DeepListID = &list.ID
	} else {
		filter.IDs = list.List
	}
	if err := s.db.BulkUpdateProblems(ctx, filter, kilonova.ProblemUpdate{Visible: upd.Visible, VisibleTests: upd.VisibleTests}); err != nil {
		return WrapError(err, "Couldn't update list problem visibility")
	}
	return nil
}

func (s *BaseAPI) DeleteProblem(ctx context.Context, problem *kilonova.Problem) *StatusError {
	// Try to delete tests first, so the contents also get deleted
	if err := s.DeleteTests(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
	}

	// Then, delete attachments, so they are fully removed from the database
	atts, err := s.ProblemAttachments(ctx, problem.ID)
	if err != nil {
		zap.S().Warn(err)
	} else {
		attIDs := []int{}
		for _, att := range atts {
			attIDs = append(attIDs, att.ID)
		}
		if _, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{IDs: attIDs, ProblemID: &problem.ID}); err != nil {
			zap.S().Warn(err)
		}
	}

	if err := s.db.DeleteProblem(ctx, problem.ID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete problem")
	}
	s.LogUserAction(ctx, "Removed problem", slog.Any("problem", problem))
	return nil
}

// When editing Problems, please edit ScoredProblems as well
func (s *BaseAPI) Problems(ctx context.Context, filter kilonova.ProblemFilter) ([]*kilonova.Problem, *StatusError) {
	problems, err := s.db.Problems(ctx, filter)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

func (s *BaseAPI) ScoredProblems(ctx context.Context, filter kilonova.ProblemFilter, scoreUser *kilonova.UserBrief, editorUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	scoreUID := -1
	if scoreUser != nil {
		scoreUID = scoreUser.ID
	}
	editorUID := -1
	if editorUser != nil {
		editorUID = editorUser.ID
	}
	problems, err := s.db.ScoredProblems(ctx, filter, scoreUID, editorUID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get problems")
	}
	return problems, nil
}

type FullProblem struct {
	kilonova.ScoredProblem
	Tags []*kilonova.Tag `json:"tags"`

	SolvedBy    int `json:"solved_by"`
	AttemptedBy int `json:"attempted_by"`
}

// SearchProblems is like the functions above but returns more detailed results for problems
func (s *BaseAPI) SearchProblems(ctx context.Context, filter kilonova.ProblemFilter, scoreUser *kilonova.UserBrief, editorUser *kilonova.UserBrief) ([]*FullProblem, int, *StatusError) {
	pbs, err := s.ScoredProblems(ctx, filter, scoreUser, editorUser)
	if err != nil {
		return nil, -1, WrapError(err, "Couldn't get problems")
	}
	cnt, err1 := s.db.CountProblems(ctx, filter)
	if err1 != nil {
		return nil, -1, WrapError(err1, "Couldn't get problem count")
	}
	ids := make([]int, 0, len(pbs))
	for _, pb := range pbs {
		ids = append(ids, pb.ID)
	}

	tagMap, err1 := s.db.ManyProblemsTags(ctx, ids)
	if err1 != nil {
		return nil, -1, WrapError(err1, "Couldn't get problem tags")
	}

	// Tags must be only on Fully Visible problems
	var tagAll = true
	var fullyVisiblePbs = make(map[int]bool)
	if filter.Look {
		pbs, err := s.Problems(ctx, kilonova.ProblemFilter{IDs: ids, Look: true, LookFullyVisible: true, LookingUser: filter.LookingUser})
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				zap.S().Warn(err)
			}
		} else {
			tagAll = false
			for _, pb := range pbs {
				fullyVisiblePbs[pb.ID] = true
			}
		}
	}

	stats, err1 := s.db.ProblemsStatistics(ctx, ids)
	if err1 != nil {
		return nil, -1, WrapError(err1, "Couldn't get problem statistics")
	}

	fullPbs := make([]*FullProblem, 0, len(pbs))
	for _, pb := range pbs {
		stat, ok := stats[pb.ID]
		if !ok {
			zap.S().Warnf("Couldn't find stats for problem %d", pb.ID)
			// Attempt to get this working even in case of error
			stat = &db.ProblemStats{NumSolvedBy: -1, NumAttemptedBy: -1}
		}
		var tags = []*kilonova.Tag{}
		if _, ok := fullyVisiblePbs[pb.ID]; tagAll || ok {
			tags, ok = tagMap[pb.ID]
			if !ok {
				zap.S().Warnf("Couldn't find tags for problem %d", pb.ID)
				tags = []*kilonova.Tag{}
			}
		}
		fullPbs = append(fullPbs, &FullProblem{
			ScoredProblem: *pb,
			AttemptedBy:   stat.NumAttemptedBy,
			SolvedBy:      stat.NumSolvedBy,
			Tags:          tags,
		})
	}

	return fullPbs, cnt, nil
}

// NOTE: This function is assumed to be used only for the looking user.
// As such, freeze times are bypassed and the current scores are shown.
func (s *BaseAPI) ContestProblems(ctx context.Context, contest *kilonova.Contest, lookingUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if !s.CanViewContestProblems(ctx, lookingUser, contest) {
		return nil, Statusf(403, "User can't view contest problems")
	}
	userID := -1
	if lookingUser != nil {
		userID = lookingUser.ID
	}
	problems, err := s.db.ScoredContestProblems(ctx, contest.ID, userID, nil)
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
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
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

// Deprecated: TODO: Remove
func (s *BaseAPI) SolvedProblems(ctx context.Context, user *kilonova.UserBrief, lookingUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if user == nil {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.ScoredProblems(ctx, kilonova.ProblemFilter{
		SolvedBy: &user.ID,

		Look:        true,
		LookingUser: lookingUser,
	}, user, lookingUser)
}

// Deprecated: TODO: Remove
func (s *BaseAPI) AttemptedProblems(ctx context.Context, user *kilonova.UserBrief, lookingUser *kilonova.UserBrief) ([]*kilonova.ScoredProblem, *StatusError) {
	if user == nil {
		return []*kilonova.ScoredProblem{}, nil
	}
	return s.ScoredProblems(ctx, kilonova.ProblemFilter{
		AttemptedBy: &user.ID,

		Look:        true,
		LookingUser: lookingUser,
	}, user, lookingUser)
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
func (s *BaseAPI) CreateProblem(ctx context.Context, title string, author *kilonova.UserBrief, consoleInput bool) (*kilonova.Problem, *StatusError) {
	problem := &kilonova.Problem{
		Name:         title,
		ConsoleInput: consoleInput,
	}

	id, err := s.insertProblem(ctx, problem, author.ID)
	if err != nil {
		return nil, err
	}
	problem.ID = id

	return problem, nil
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

func (s *BaseAPI) ProblemChecklist(ctx context.Context, pbid int) (*kilonova.ProblemChecklist, *StatusError) {
	chk, err := s.db.ProblemChecklist(ctx, pbid)
	if err != nil || chk == nil {
		return nil, WrapError(err, "Couldn't get problem checklist")
	}
	return chk, nil
}

type ProblemStatistics struct {
	NumSolved    int `json:"num_solved"`
	NumAttempted int `json:"num_attempted"`

	SizeLeaderboard   *Submissions `json:"size_leaderboard"`
	MemoryLeaderboard *Submissions `json:"memory_leaderboard"`
	TimeLeaderboard   *Submissions `json:"time_leaderboard"`
}

func (s *BaseAPI) ProblemStatistics(ctx context.Context, problem *kilonova.Problem, lookingUser *kilonova.UserBrief) (*ProblemStatistics, *StatusError) {
	if ok := s.IsProblemFullyVisible(lookingUser, problem); !ok {
		return nil, Statusf(401, "Looking user must be full problem viewer")
	}

	numberStats, err := s.db.ProblemsStatistics(ctx, []int{problem.ID})
	if err != nil {
		return nil, WrapError(err, "Couldn't get attempted/solved user count")
	}
	if _, ok := numberStats[problem.ID]; !ok {
		return nil, Statusf(500, "Couldn't get attempted/solved user count for problem")
	}

	sizeRaw, err := s.db.ProblemStatisticsSize(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by size")
	}
	size, err1 := s.fillSubmissions(ctx, -1, sizeRaw, true, lookingUser, false)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by size")
	}

	memoryRaw, err := s.db.ProblemStatisticsMemory(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by memory")
	}
	memory, err1 := s.fillSubmissions(ctx, -1, memoryRaw, true, lookingUser, false)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by memory")
	}

	timeRaw, err := s.db.ProblemStatisticsTime(ctx, problem.ID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get statistics by time")
	}
	time, err1 := s.fillSubmissions(ctx, -1, timeRaw, true, lookingUser, false)
	if err1 != nil {
		return nil, WrapError(err1, "Couldn't get full statistics by time")
	}

	return &ProblemStatistics{
		NumSolved:    numberStats[problem.ID].NumSolvedBy,
		NumAttempted: numberStats[problem.ID].NumAttemptedBy,

		SizeLeaderboard:   size,
		MemoryLeaderboard: memory,
		TimeLeaderboard:   time,
	}, nil
}

type ProblemDiagnostic struct {
	// Use an slog.Level for the type (Info, Warn, Error)
	Level slog.Level

	// English diagnostic message
	Message string
}

func (s *BaseAPI) ProblemDiagnostics(ctx context.Context, problem *kilonova.Problem) ([]*ProblemDiagnostic, *StatusError) {
	diags := []*ProblemDiagnostic{}

	tests, err := s.Tests(ctx, problem.ID)
	if err != nil {
		return nil, err
	}

	subtasks, err := s.SubTasks(ctx, problem.ID)
	if err != nil {
		return nil, err
	}

	// Sum of maximum subtasks but no subtasks will error the max score attribute
	if problem.ScoringStrategy == kilonova.ScoringTypeSumSubtasks && len(subtasks) == 0 {
		diags = append(diags, &ProblemDiagnostic{
			Level:   slog.LevelError,
			Message: "Scoring Type is 'Sum of maximum Subtasks' but no Subtasks exist.",
		})
	}

	// Check if all tests are in at least one subtask
	testMap := make(map[int]bool)
	subtaskTestMap := make(map[int]bool)
	for _, test := range tests {
		testMap[test.ID] = true
	}
	for _, subtask := range subtasks {
		for _, test := range subtask.Tests {
			subtaskTestMap[test] = true
		}
	}
	if len(subtaskTestMap) > 0 && !maps.Equal(testMap, subtaskTestMap) {
		diags = append(diags, &ProblemDiagnostic{
			Level:   slog.LevelInfo,
			Message: "Not all tests belong to a Subtask.",
		})
	}

	var totalScore decimal.Decimal = problem.DefaultPoints.Copy()
	if len(subtasks) == 0 {
		for _, test := range tests {
			totalScore = totalScore.Add(test.Score)
		}
	} else {
		for _, subtask := range subtasks {
			totalScore = totalScore.Add(subtask.Score)
		}
	}

	// If score is not zero but also not 100, warn
	if math.Abs(totalScore.InexactFloat64()-100) > 0.01 && !totalScore.IsZero() {
		msg := "Total score is not 100"
		if problem.ScoringStrategy == kilonova.ScoringTypeICPC {
			msg += " (problem type is ICPC, however)"
		}
		diags = append(diags, &ProblemDiagnostic{
			Level:   slog.LevelWarn,
			Message: msg + ".",
		})
	}

	return diags, nil
}
