package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/gosimple/slug"
	"github.com/jinzhu/gorm"
)

// RegisterProblemRoutes registers the problem routes at /api/problem
func (s *API) RegisterProblemRoutes() chi.Router {
	r := chi.NewRouter()
	// /problem/getAll
	r.Get("/getAll", s.GetAllProblems)
	// /problem/getByID
	r.Get("/getByID", s.GetProblemByID)
	// /problem/create
	r.With(s.MustBeAuthed).Post("/create", s.InitProblem)
	r.With(s.MustBeAuthed).Route("/{id}", func(r chi.Router) {
		// TODO: Make sure it is the author or admin who does the change
		r.Use(s.ValidateProblemID)
		r.Route("/update", func(r chi.Router) {
			// /problem/{id}/update/title
			r.Post("/title", func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("A", r.ParseForm())
				val := r.FormValue("title")
				fmt.Println(val)
				spew.Dump(r.PostForm, r.Form)
				s.db.UpdateProblemField(s.getContextValue(r, "pbID").(uint), "name", val)
			})
			// /problem/{id}/update/description
			r.Post("/description", func(w http.ResponseWriter, r *http.Request) {
				val := r.FormValue("text")
				s.db.UpdateProblemField(s.getContextValue(r, "pbID").(uint), "text", val)
			})
			// /problem/{id}/update/addTest
			r.Post("/addTest", s.CreateTest)
			// /problem/{id}/update/limits
			r.Post("/limits", s.SetLimits)
			// /problem/{id}/update/updateText
			r.Post("/updateTest", func(w http.ResponseWriter, r *http.Request) {
				// TODO
			})
			// /problem/{id}/update/removeTests
			r.Post("/removeTests", func(w http.ResponseWriter, r *http.Request) {
				problem := common.ProblemFromContext(r)

				var tests = make([]common.Test, len(problem.Tests))
				copy(tests, problem.Tests)
				s.db.UpdateProblemField(problem.ID, "tests", []common.Test{})
				for _, t := range tests {
					fmt.Println("Trying to delete test with ID", t.ID)
					if err := s.db.DB.Delete(&t).Error; err != nil {
						// fixme(alexv): don't send the error
						s.ErrorData(w, err.Error(), 500)
					}
				}
			})
			r.Post("/setConsoleInput", func(w http.ResponseWriter, r *http.Request) {
				val := r.FormValue("isSet")
				problem := common.ProblemFromContext(r)
				if val == "true" {
					s.db.UpdateProblemField(problem.ID, gorm.ToColumnName("ConsoleInput"), true)
				} else if val == "false" {
					s.db.UpdateProblemField(problem.ID, gorm.ToColumnName("ConsoleInput"), false)
				} else {
					s.ErrorData(w, "Invalid value for `isSet` form value", http.StatusBadRequest)
				}
			})
			r.Post("/setTestName", func(w http.ResponseWriter, r *http.Request) {
				val := r.FormValue("testName")
				if val == "" {
					s.ErrorData(w, "You must set the `testName` form value", http.StatusBadRequest)
					return
				}
				problem := common.ProblemFromContext(r)
				s.db.UpdateProblemField(problem.ID, gorm.ToColumnName("TestName"), val)
			})
		})
		r.Route("/get", func(r chi.Router) {
			// /problem/{id}/get/tests
			r.Get("/tests", func(w http.ResponseWriter, r *http.Request) {
				s.ReturnData(w, "success", common.ProblemFromContext(r).Tests)
			})
			// /problem/{id}/get/test
			// required param: {id} - the wanted ID (should be the visible ID) (uint)
			r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				sid := r.FormValue("id")
				if sid == "" {
					s.ErrorData(w, "You must specify a test ID", http.StatusBadRequest)
					return
				}
				id, err := strconv.ParseUint(sid, 10, 32)
				if err != nil {
					s.ErrorData(w, "Invalid test ID", http.StatusBadRequest)
					return
				}
				for _, t := range common.ProblemFromContext(r).Tests {
					if t.VisibleID == uint(id) {
						s.ReturnData(w, "success", t)
						return
					}
				}
				s.ErrorData(w, "Test not found (did put the visible ID?)", http.StatusBadRequest)
			})
			// /problem/{id}/get/testData
			// URL params:
			//  - id - the test id
			//  - noIn - if not null, the input file won't be sent
			//  - noOut - if not null, the output file won't be sent
			r.Get("/testData", s.GetTestData)
		})
	})
	return r
}

// SetLimits sets the limits for a problem
func (s *API) SetLimits(w http.ResponseWriter, r *http.Request) {
	pb := common.ProblemFromContext(r)
	lim := pb.Limits
	spew.Dump(lim)

	// in case limits is empty, set up the problem ID to save it to the DB
	lim.ProblemID = pb.ID

	if r.FormValue("memoryLimit") != "" {
		memoryLimit, err := strconv.ParseFloat(r.FormValue("memoryLimit"), 64)
		if err != nil || memoryLimit <= 0 {
			s.ErrorData(w, "memoryLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("memory limit:", memoryLimit)
		lim.MemoryLimit = memoryLimit
	}
	if r.FormValue("stackLimit") != "" {
		stackLimit, err := strconv.ParseFloat(r.FormValue("stackLimit"), 64)
		if err != nil || stackLimit <= 0 {
			s.ErrorData(w, "stackLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("stack limit:", stackLimit)
		lim.StackLimit = stackLimit
	}
	if r.FormValue("timeLimit") != "" {
		timeLimit, err := strconv.ParseFloat(r.FormValue("timeLimit"), 64)
		if err != nil || timeLimit <= 0 {
			s.ErrorData(w, "timeLimit must be a valid float", http.StatusBadRequest)
			return
		}
		fmt.Println("time limit:", timeLimit)
		lim.TimeLimit = timeLimit
	}
	if err := s.db.Save(&lim); err != nil {
		s.errlog("API: /problem/%d/update/limits, db saving error: %s\n", pb.ID, err)
		s.ErrorData(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", "Limits saved")
}

// CreateTest inserts a new test to the problem
func (s *API) CreateTest(w http.ResponseWriter, r *http.Request) {
	// TODO: Change to files, or make both options interchangeable
	score, err := strconv.Atoi(r.FormValue("score"))
	if err != nil {
		s.ErrorData(w, "Score not integer", http.StatusBadRequest)
		return
	}
	var visibleID uint64
	if vID := r.FormValue("visibleID"); vID != "" {
		visibleID, err = strconv.ParseUint(vID, 10, 32)
		if err != nil {
			s.ErrorData(w, "Visible ID not int", http.StatusBadRequest)
			return
		}
	} else {
		// set it to be the largest visible id of a test + 1
		tests := common.ProblemFromContext(r).Tests
		var max uint = 0
		for _, t := range tests {
			if max < t.VisibleID {
				max = t.VisibleID
			}
		}
		if max == 0 {
			visibleID = 1
		} else {
			visibleID = uint64(max)
		}
	}

	var test common.Test
	test.ProblemID = s.pbIDFromReq(r)
	test.Score = score
	test.VisibleID = uint(visibleID)
	s.db.Save(&test)
	if err := s.manager.SaveTest(
		s.pbIDFromReq(r),
		uint(test.VisibleID),
		[]byte(r.FormValue("input")),
		[]byte(r.FormValue("output")),
	); err != nil {
		fmt.Println("Hăăăăăă", err)
	}
	s.ReturnData(w, "success", test.ID)
}

// InitProblem assigns an ID for the problem
// TODO: Handle limits
func (s *API) InitProblem(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		s.ErrorData(w, "Title not provided", http.StatusBadRequest)
		return
	}

	cistr := r.FormValue("consoleInput")
	var consoleInput bool
	if cistr != "" {
		ci, err := strconv.ParseBool(cistr)
		if err != nil {
			s.ErrorData(w, "Invalid `consoleInput` form value", http.StatusBadRequest)
			return
		}
		consoleInput = ci
	}

	if s.db.ProblemExists(title) {
		s.ErrorData(w, "Problem with specified title already exists in DB", http.StatusBadRequest)
		return
	}

	var problem common.Problem
	problem.Name = title
	problem.User = common.UserFromContext(r)
	problem.ConsoleInput = consoleInput
	problem.TestName = slug.Make(title)
	// default limits
	problem.Limits = common.Limits{
		MemoryLimit: 64,  // 64MB
		StackLimit:  16,  // 16MB
		TimeLimit:   0.1, // 0.1s
	}

	if err := s.db.Save(&problem); err != nil {
		s.errlog("API: /problem/create, db saving error: %s\n", err)
		s.ReturnData(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", problem.ID)
}

// GetAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) GetAllProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := s.db.GetAllProblems()
	if err != nil {
		s.errlog("%s", err)
		s.ReturnData(w, http.StatusText(500), 500)
		return
	}
	s.ReturnData(w, "success", problems)
}

// GetProblemByID returns a problem from the DB specified by ID
func (s *API) GetProblemByID(w http.ResponseWriter, r *http.Request) {
	idstr := r.FormValue("id")
	id, err := strconv.ParseUint(idstr, 10, 64)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
		return
	}
	problem, err := s.db.GetProblemByID(uint(id))
	if err != nil {
		s.ErrorData(w, "Problem with ID doesn't exist", http.StatusBadRequest)
		return
	}
	s.ReturnData(w, "success", problem)
}

// ValidateProblemID pre-emptively returns if there isnt a valid problem ID in the URL params
// Also, it fetches the problem from the DB and makes sure it exists
func (s *API) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			s.ErrorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		problem, err := s.db.GetProblemByID(uint(problemID))
		if err != nil {
			s.ErrorData(w, "problem does not exist", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), common.PbID, uint(problemID))
		ctx = context.WithValue(ctx, common.ProblemKey, problem)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s API) pbIDFromReq(r *http.Request) uint {
	return s.getContextValue(r, "pbID").(uint)
}

// GetTestData returns the test data from a specifed test of a specified problem
func (s API) GetTestData(w http.ResponseWriter, r *http.Request) {
	sid := r.FormValue("id")
	if sid == "" {
		s.ErrorData(w, "You must specify a test ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		s.ErrorData(w, "Invalid test ID", http.StatusBadRequest)
		return
	}

	in, out, err := s.manager.GetTest(s.pbIDFromReq(r), uint(id))
	if err != nil {
		s.ErrorData(w, err.Error(), 500)
		return
	}

	var ret struct {
		In  string `json:"in"`
		Out string `json:"out"`
	}
	if r.FormValue("noIn") == "" {
		ret.In = string(in)
	}
	if r.FormValue("noOut") == "" {
		ret.Out = string(out)
	}
	s.ReturnData(w, "success", ret)
	return
}
