package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) maxScore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID <= 0 {
		if util.UserBrief(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.UserBrief(r).ID
	}

	returnData(w, s.base.MaxScore(r.Context(), args.UserID, util.Problem(r).ID))
}

type scoreBreakdownRet struct {
	MaxScore int                           `json:"max_score"`
	Problem  *kilonova.Problem             `json:"problem"`
	Subtasks []*kilonova.SubmissionSubTask `json:"subtasks"`

	// ProblemEditor is true only if the request author is public
	// It does not take into consideration if the supplied user is the problem editor
	ProblemEditor bool `json:"problem_editor"`
	// Subtests are arranged from submission subtasks so something legible can be rebuilt to show more information on the subtasks
	Subtests []*kilonova.SubTest `json:"subtests"`
}

func (s *API) maxScoreBreakdown(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID int

		ContestID *int
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	// This endpoint may leak stuff that shouldn't be generally seen (like in contests), so restrict this option to editors only
	// It isn't used anywhere right now, but it might be useful in the future
	if !s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
		args.UserID = -1
	}
	if args.UserID <= 0 {
		if util.UserBrief(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.UserBrief(r).ID
	}

	if util.Problem(r).ScoringStrategy != kilonova.ScoringTypeSumSubtasks {
		errorData(w, "Breakdowns are only for problems with a sum of subtasks scoring strategy", 400)
		return
	}

	maxScore := -1
	if args.ContestID == nil {
		maxScore = s.base.MaxScore(r.Context(), args.UserID, util.Problem(r).ID)
	} else {
		maxScore = s.base.ContestMaxScore(r.Context(), args.UserID, util.Problem(r).ID, *args.ContestID)
	}

	stks, err := s.base.MaximumScoreSubTasks(r.Context(), util.Problem(r).ID, args.UserID, args.ContestID)
	if err != nil {
		err.WriteError(w)
		return
	}

	tests, err := s.base.MaximumScoreSubTaskTests(r.Context(), util.Problem(r).ID, args.UserID, args.ContestID)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, scoreBreakdownRet{
		MaxScore: maxScore,
		Problem:  util.Problem(r),
		Subtasks: stks,
		Subtests: tests,

		ProblemEditor: s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)),
	})
}

func (s *API) deleteProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.base.DeleteProblem(context.Background(), util.Problem(r).ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Deleted problem")
}

func (s *API) initProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Title        string `json:"title"`
		ConsoleInput bool   `json:"consoleInput"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	id, err := s.base.CreateProblem(r.Context(), args.Title, util.UserBrief(r), args.ConsoleInput)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

func (s *API) getProblems(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemFilter
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	args.Look = true
	args.LookingUser = util.UserBrief(r)

	problems, err := s.base.Problems(r.Context(), args)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, problems)
}

func (s *API) updateProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemUpdate
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if err := s.base.UpdateProblem(r.Context(), util.Problem(r).ID, args, util.UserBrief(r)); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Updated problem")
}

func (s *API) addProblemEditor(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"username"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.AddProblemEditor(r.Context(), util.Problem(r).ID, user.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Added problem editor")
}

func (s *API) addProblemViewer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"username"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		err.WriteError(w)
		return
	}

	if user.ID == util.UserBrief(r).ID {
		errorData(w, "You can't demote yourself to viewer rank!", 400)
		return
	}

	if err := s.base.AddProblemViewer(r.Context(), util.Problem(r).ID, user.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Added problem viewer")
}

func (s *API) stripProblemAccess(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID int `json:"user_id"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID == util.UserBrief(r).ID {
		errorData(w, "You can't strip your own access!", 400)
		return
	}

	if err := s.base.StripProblemAccess(r.Context(), util.Problem(r).ID, args.UserID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Stripped problem access")
}

func (s *API) getProblemAccessControl(w http.ResponseWriter, r *http.Request) {
	editors, err := s.base.ProblemEditors(r.Context(), util.Problem(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	viewers, err := s.base.ProblemViewers(r.Context(), util.Problem(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Editors []*kilonova.UserBrief `json:"editors"`
		Viewers []*kilonova.UserBrief `json:"viewers"`
	}{
		Editors: editors,
		Viewers: viewers,
	})
}
