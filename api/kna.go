package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createKNA(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Problems string `json:"problems"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	pbIDs, ok := DecodeIntString(args.Problems)
	if !ok {
		errorData(w, "Invalid problem string", 400)
		return
	}
	pbs := make([]*kilonova.Problem, 0, len(pbIDs))
	for _, id := range pbIDs {
		pb, err := s.pserv.ProblemByID(r.Context(), id)
		if err != nil {
			log.Println(err)
			errorData(w, "One of the problem IDs is invalid", 400)
			return
		}
		pbs = append(pbs, pb)
	}
	rd, err := kilonova.GenKNA(pbs, s.tserv, s.stkserv, s.kn.DM)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	defer rd.Close()

	var b bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &b)
	defer enc.Close()

	if _, err := io.Copy(enc, rd); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, b.String())
}

func (s *API) loadKNA(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(100 * 1024 * 1024); err != nil { // 100MB
		errorData(w, err, 500)
		return
	}
	file, _, err := r.FormFile("archive")
	if err != nil {
		errorData(w, err, 500)
		return
	}
	defer file.Close()

	id := util.User(r).ID
	sid := r.FormValue("author_id")
	if sid != "" {
		nid, err := strconv.Atoi(sid)
		if err == nil {
			id = nid
		}
	}

	if _, err := s.userv.UserByID(r.Context(), id); err != nil {
		errorData(w, "Invalid author", 400)
		return
	}

	pbs, err := kilonova.ReadKNA(file)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	var errorsHappened bool
	for _, pb := range pbs {
		pb.Name = s.nextProblemName(r.Context(), pb.Name)
		pb.AuthorID = id
		if err := s.pserv.CreateProblem(r.Context(), &pb.Problem); err != nil {
			errorsHappened = true
			log.Println(err)
			continue
		}

		testAssocs := make(map[int]int)

		for _, test := range pb.Tests {
			test.ProblemID = pb.ID
			oldID := test.ID
			if err := s.tserv.CreateTest(r.Context(), &test.Test); err != nil {
				errorsHappened = true
				log.Printf("Problem %d, test creation: %s\n", pb.ID, err)
				continue
			}
			testAssocs[oldID] = test.ID
			if err := s.manager.SaveTestInput(test.ID, bytes.NewReader(test.Input)); err != nil {
				errorsHappened = true
				log.Printf("Problem %d, test %d input: %s\n", pb.ID, test.ID, err)
				continue
			}
			if err := s.manager.SaveTestOutput(test.ID, bytes.NewReader(test.Output)); err != nil {
				errorsHappened = true
				log.Printf("Problem %d, test %d output: %s\n", pb.ID, test.ID, err)
				continue
			}
		}
		for _, stk := range pb.SubTasks {
			stk.ProblemID = pb.ID
			newTestIDs := []int{}
			for _, id := range stk.Tests {
				newID, ok := testAssocs[id]
				if ok {
					newTestIDs = append(newTestIDs, newID)
				}
			}
			stk.Tests = newTestIDs
			if err := s.stkserv.CreateSubTask(r.Context(), stk); err != nil {
				errorsHappened = true
				log.Printf("Problem %d, subtask creation: %s\n", pb.ID, err)
				continue
			}
		}
	}
	if !errorsHappened {
		returnData(w, "Archive successfully imported")
	} else {
		errorData(w, "Some errors happened while importing, check server logs", 500)
	}
}

func (s *API) nextProblemName(ctx context.Context, name string) string {
	pbs, err := s.pserv.Problems(ctx, kilonova.ProblemFilter{Name: &name})
	if err == nil && len(pbs) == 0 {
		return name
	}

	for i := 1; i <= 10; i++ {
		newName := fmt.Sprintf("%s (%d)", name, i)
		pbs, err := s.pserv.Problems(ctx, kilonova.ProblemFilter{Name: &newName})
		if err == nil && len(pbs) == 0 {
			return newName
		}
	}
	return kilonova.RandomString(10)
}
