package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/shopspring/decimal"
)

func (s *API) createSubTask(w http.ResponseWriter, r *http.Request) {
	var args struct {
		VisibleID int   `json:"visible_id"`
		Score     int   `json:"score"`
		Tests     []int `json:"tests"`
	}
	if err := parseJSONBody(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if stk1, _ := s.base.SubTask(r.Context(), util.Problem(r).ID, args.VisibleID); stk1 != nil && stk1.ID != 0 {
		errorData(w, "SubTask with that ID already exists!", 400)
		return
	}

	if len(args.Tests) == 0 {
		errorData(w, "No tests specified", 400)
		return
	}

	realIDs := []int{}
	for _, id := range args.Tests {
		test, err := s.base.Test(r.Context(), util.Problem(r).ID, id)
		if err != nil {
			continue
		}
		realIDs = append(realIDs, test.ID)
	}

	stk := kilonova.SubTask{
		ProblemID: util.Problem(r).ID,
		VisibleID: args.VisibleID,
		Score:     decimal.NewFromInt(int64(args.Score)),
		Tests:     realIDs,
	}

	if err := s.base.CreateSubTask(r.Context(), &stk); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, stk.ID)
}

func (s *API) updateSubTask(w http.ResponseWriter, r *http.Request) {
	var args struct {
		SubTaskID *int     `json:"subtask_id"`
		NewID     *int     `json:"new_id"`
		Score     *float64 `json:"score"`
		Tests     []int    `json:"tests"`
	}
	if err := parseJSONBody(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if args.SubTaskID == nil {
		errorData(w, "SubTask ID must not be empty", 400)
		return
	}

	stk, err := s.base.SubTask(r.Context(), util.Problem(r).ID, *args.SubTaskID)
	if err != nil {
		statusError(w, err)
		return
	}

	var score *decimal.Decimal
	if args.Score != nil {
		val := decimal.NewFromFloat(*args.Score)
		score = &val
	}

	if err := s.base.UpdateSubTask(r.Context(), stk.ID, kilonova.SubTaskUpdate{
		VisibleID: args.NewID,
		Score:     score,
	}); err != nil {
		statusError(w, err)
		return
	}

	if args.Tests != nil {
		newIDs := make([]int, 0, len(args.Tests))
		for _, id := range args.Tests {
			test, err := s.base.Test(r.Context(), util.Problem(r).ID, id)
			if err != nil {
				errorData(w, "One of the tests does not exist", 400)
				return
			}
			newIDs = append(newIDs, test.ID)
		}

		if err := s.base.UpdateSubTaskTests(r.Context(), stk.ID, newIDs); err != nil {
			statusError(w, err)
			return
		}
	}

	returnData(w, "Updated SubTask")
}

func (s *API) bulkDeleteSubTasks(w http.ResponseWriter, r *http.Request) {
	var removedSubTasks int
	var subtaskIDs []int
	if err := parseJSONBody(r, &subtaskIDs); err != nil {
		statusError(w, err)
		return
	}

	for _, id := range subtaskIDs {
		if stk, err := s.base.SubTask(r.Context(), util.Problem(r).ID, id); err == nil {
			if err := s.base.DeleteSubTask(r.Context(), stk.ID); err == nil {
				removedSubTasks++
			}
		}
	}
	if removedSubTasks != len(subtaskIDs) {
		errorData(w, "Some SubTasks could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected subTasks")
}

func (s *API) bulkUpdateSubTaskScores(w http.ResponseWriter, r *http.Request) {
	var data map[int]decimal.Decimal
	var updatedSubTasks int

	if err := parseJSONBody(r, &data); err != nil {
		statusError(w, err)
		return
	}
	for k, v := range data {
		v := v
		if stk, err := s.base.SubTask(r.Context(), util.Problem(r).ID, k); err == nil {
			if err := s.base.UpdateSubTask(r.Context(), stk.ID, kilonova.SubTaskUpdate{Score: &v}); err == nil {
				updatedSubTasks++
			} else {
				spew.Dump(err)
			}
		} else {
			spew.Dump(stk, err)
		}
	}

	if updatedSubTasks != len(data) {
		errorData(w, "Some subTasks could not be updated", 500)
		return
	}
	returnData(w, "Updated all subTasks")
}
