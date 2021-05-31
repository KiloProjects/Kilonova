package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
)

func (s *API) createSubTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		VisibleID int    `json:"visible_id"`
		Score     int    `json:"score"`
		Tests     string `json:"tests"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if stk1, _ := s.db.SubTask(r.Context(), util.Problem(r).ID, args.VisibleID); stk1 != nil && stk1.ID != 0 {
		errorData(w, "SubTask with that ID already exists!", 400)
		return
	}

	ids, ok := DecodeIntString(args.Tests)
	if !ok {
		errorData(w, "Invalid test string", 400)
		return
	}
	if len(ids) == 0 {
		errorData(w, "No tests specified", 400)
		return
	}

	realIDs := []int{}
	for _, id := range ids {
		test, err := s.db.Test(r.Context(), util.Problem(r).ID, id)
		if err != nil {
			continue
		}
		realIDs = append(realIDs, test.ID)
	}

	stk := kilonova.SubTask{
		ProblemID: util.Problem(r).ID,
		VisibleID: args.VisibleID,
		Score:     args.Score,
		Tests:     realIDs,
	}

	if err := s.db.CreateSubTask(r.Context(), &stk); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, stk.ID)
}

func (s *API) updateSubTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		SubTaskID int     `json:"subtask_id"`
		NewID     *int    `json:"new_id"`
		Score     *int    `json:"score"`
		Tests     *string `json:"tests"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.SubTaskID == 0 {
		errorData(w, "SubTask ID must not be empty", 400)
		return
	}

	stk, err := s.db.SubTask(r.Context(), util.Problem(r).ID, args.SubTaskID)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	upd := kilonova.SubTaskUpdate{
		VisibleID: args.NewID,
		Score:     args.Score,
	}

	if args.Tests != nil {
		ids, ok := DecodeIntString(*args.Tests)
		if !ok {
			errorData(w, "One of the test IDs is not an int", 400)
			return
		}
		newIDs := make([]int, 0, len(ids))
		for _, id := range ids {
			test, err := s.db.Test(r.Context(), util.Problem(r).ID, id)
			if err != nil {
				errorData(w, "One of the tests does not exist", 400)
				return
			}
			newIDs = append(newIDs, test.ID)
		}
		upd.Tests = newIDs
	}

	if err := s.db.UpdateSubTask(r.Context(), stk.ID, upd); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updates SubTask")
}

func (s *API) bulkDeleteSubTasks(w http.ResponseWriter, r *http.Request) {
	var removedSubTasks int
	ids, ok := DecodeIntString(r.FormValue("subTasks"))
	if !ok || len(ids) == 0 {
		errorData(w, "Invalid input string", 400)
		return
	}
	for _, id := range ids {
		if stk, err := s.db.SubTask(r.Context(), util.Problem(r).ID, id); err == nil {
			if err := s.db.DeleteSubTask(r.Context(), stk.ID); err == nil {
				removedSubTasks++
			}
		}
	}
	if removedSubTasks != len(ids) {
		errorData(w, "Some SubTasks could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected subTasks")
}

func (s *API) bulkUpdateSubTaskScores(w http.ResponseWriter, r *http.Request) {
	var data map[int]int
	var updatedSubTasks int

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&data); err != nil {
		log.Println(err)
		errorData(w, "Could not decode body", 400)
		return
	}
	for k, v := range data {
		if stk, err := s.db.SubTask(r.Context(), util.Problem(r).ID, k); err == nil {
			if err := s.db.UpdateSubTask(r.Context(), stk.ID, kilonova.SubTaskUpdate{Score: &v}); err == nil {
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

func DecodeIntString(str string) ([]int, bool) {
	ids := []int{}
	strTestIDs := strings.Split(str, ",")
	for _, ss := range strTestIDs {
		if ss == "" {
			continue
		}
		val, err := strconv.Atoi(ss)
		if err != nil {
			return nil, false
		}
		ids = append(ids, val)
	}
	return ids, true
}
