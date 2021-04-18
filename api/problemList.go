package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
)

func (s *API) getProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.plserv.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, list)
}

func (s *API) filterProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ProblemListFilter
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	lists, err := s.plserv.ProblemLists(r.Context(), args)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, lists)
}

func (s *API) initProblemList(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	description := r.FormValue("description")
	problemIDs := r.FormValue("ids")
	inputIDs, ok := DecodeIntString(problemIDs)
	if !ok || len(inputIDs) == 0 {
		errorData(w, "Invalid id list", 400)
		return
	}

	var pbs []*kilonova.Problem
	var err error
	if util.User(r).Admin {
		pbs, err = s.pserv.Problems(r.Context(), kilonova.ProblemFilter{IDs: inputIDs})
	} else {
		pbs, err = s.pserv.Problems(r.Context(), kilonova.ProblemFilter{IDs: inputIDs, LookingUserID: &util.User(r).ID})
	}
	if err != nil {
		errorData(w, err, 500)
		return
	}

	spew.Dump(inputIDs, pbs)

	actualIDs := make([]int, 0, len(pbs))
	for _, pb := range pbs {
		actualIDs = append(actualIDs, pb.ID)
	}
	if len(actualIDs) == 0 {
		errorData(w, "Number of problems specified that you can see is 0", 400)
		return
	}

	var list kilonova.ProblemList
	list.Title = title
	list.Description = description
	list.AuthorID = util.User(r).ID
	list.List = actualIDs
	if err := s.plserv.CreateProblemList(r.Context(), &list); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, list.ID)
}

func (s *API) updateProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		List        *string `json:"list"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.plserv.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, "Couldn't find problem list", 400)
		return
	}

	if !(util.User(r).Admin || util.User(r).ID == list.AuthorID) {
		errorData(w, "You can't update this problem list!", 403)
		return
	}

	upd := kilonova.ProblemListUpdate{
		Title:       args.Title,
		Description: args.Description,
	}

	if args.List != nil {
		ll := strings.Split(*args.List, ",")
		l := make([]int, 0, len(ll))
		for _, str := range ll {
			val, err := strconv.Atoi(str)
			if err != nil {
				errorData(w, "Bad problem id list", 400)
				return
			}
			l = append(l, val)
		}
		upd.List = l
	}

	if err := s.plserv.UpdateProblemList(r.Context(), args.ID, upd); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated problem list")
}

func (s *API) deleteProblemList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct{ ID int }
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	list, err := s.plserv.ProblemList(r.Context(), args.ID)
	if err != nil {
		errorData(w, "Couldn't find problem list", 400)
		return
	}

	if !(util.User(r).Admin || util.User(r).ID == list.AuthorID) {
		errorData(w, "You can't delete this problem list!", 403)
		return
	}

	if err := s.plserv.DeleteProblemList(r.Context(), args.ID); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, "Removed problem list")
}
