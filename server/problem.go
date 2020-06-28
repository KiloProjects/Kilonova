package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
)

// RegisterProblemRoutes registers the problem routes at /api/problem
func (s *API) RegisterProblemRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/getAll", s.GetAllProblems)
	r.Get("/getByID", s.GetProblemByID)
	r.With(s.MustBeAuthed).Post("/create", s.InitProblem)
	r.With(s.MustBeAuthed).Route("/update/{id}", func(r chi.Router) {
		// TODO: Make sure it is the author or admin who does the change
		r.Use(s.ValidateProblemID)
		r.Post("/title", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("A", r.ParseForm())
			val := r.FormValue("title")
			fmt.Println(val)
			spew.Dump(r.PostForm, r.Form)
			s.db.Model(&common.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).Update("name", val)
		})
		r.Post("/setText", func(w http.ResponseWriter, r *http.Request) {
			val := r.FormValue("text")
			s.db.Model(&common.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).Update("text", val)
		})
		r.Post("/addTest", s.CreateTest)
		r.Post("/description", func(w http.ResponseWriter, r *http.Request) {
			val := r.FormValue("description")
			s.db.Model(&common.Problem{}).Where("id = ?", s.getContextValue(r, "pbID")).UpdateColumn("text", val)
		})
		r.Post("/updateTest", func(w http.ResponseWriter, r *http.Request) {
			// TODO
		})
		r.Post("/removeTests", func(w http.ResponseWriter, r *http.Request) {
			var problem common.Problem
			s.db.Preload("Tests").First(&problem, s.pbIDFromReq(r))
			if err := s.db.Unscoped().Delete(&problem.Tests).Error; err != nil {
				s.ErrorData(w, err.Error(), http.StatusInternalServerError)
			}
		})
		r.Get("/getTests", func(w http.ResponseWriter, r *http.Request) {
			s.ReturnData(w, "success", s.pbFromReq(r).Tests)
		})
		// required param: {id} - the wanted ID (should be the visible ID) (uint)
		r.Get("/getTest", func(w http.ResponseWriter, r *http.Request) {
			sid := r.FormValue("id")
			if sid == "" {
				s.ErrorData(w, "You must specify a test ID", http.StatusBadRequest)
				return
			}
			id, err := strconv.ParseUint(sid, 10, 32)
			if err != nil {
				s.ErrorData(w, "Invalid test ID", http.StatusBadRequest)
			}
			for _, t := range s.pbFromReq(r).Tests {
				if t.VisibleID == uint(id) {
					s.ReturnData(w, "success", t)
					return
				}
			}
			s.ErrorData(w, "Test not found (did put the visible ID?)", http.StatusBadRequest)
		})
		r.Get("/getTestData", func(w http.ResponseWriter, r *http.Request) {
			sid := r.FormValue("id")
			if sid == "" {
				s.ErrorData(w, "You must specify a test ID", http.StatusBadRequest)
				return
			}
			id, err := strconv.ParseUint(sid, 10, 32)
			if err != nil {
				s.ErrorData(w, "Invalid test ID", http.StatusBadRequest)
			}

			in, out, err := s.manager.GetTest(s.pbIDFromReq(r), uint(id))
			if err != nil {
				s.ErrorData(w, err.Error(), http.StatusInternalServerError)
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
		})
	})
	return r
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
		}
	} else {
		// set it to be the largest visible id of a test + 1
		tests := s.pbFromReq(r).Tests
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
	s.manager.SaveTest(
		s.pbIDFromReq(r),
		uint(test.VisibleID),
		[]byte(r.FormValue("input")),
		[]byte(r.FormValue("output")),
	)
	s.ReturnData(w, "success", test.ID)
}

// InitProblem assigns an ID for the problem
func (s *API) InitProblem(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	if title == "" {
		s.ErrorData(w, "Title not provided", http.StatusBadRequest)
		return
	}
	var cnt int
	s.db.Model(&common.Problem{}).Where("lower(name) = lower(?)", title).Count(&cnt)
	if cnt > 0 {
		s.ErrorData(w, "Title already exists in DB", http.StatusBadRequest)
		return
	}
	var problem common.Problem
	problem.Name = title
	problem.User = common.UserFromContext(r)
	problem.UserID = common.UserFromContext(r).ID
	s.db.Create(&problem)
	s.ReturnData(w, "success", problem.ID)
}

// GetAllProblems returns all the problems from the DB
// TODO: Pagination
func (s *API) GetAllProblems(w http.ResponseWriter, r *http.Request) {
	var problems []common.Problem
	fmt.Println("start")
	s.db.Preload("Tests").Find(&problems)
	fmt.Println("end")
	s.ReturnData(w, "success", problems)
}

// GetProblemByID returns a problem from the DB specified by ID
func (s *API) GetProblemByID(w http.ResponseWriter, r *http.Request) {
	var problem common.Problem
	idstr := r.FormValue("id")
	id, err := strconv.Atoi(idstr)
	if err != nil {
		fmt.Fprintln(w, "Invalid ID")
	}
	s.db.Where("id = ?", id).Set("gorm:auto_preload", true).First(&problem)
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
		var problem common.Problem
		s.db.Preload("Tests").First(&problem, uint(problemID))
		if problem.ID == 0 {
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

func (s API) pbFromReq(r *http.Request) *common.Problem {
	return s.getContextValue(r, "problem").(*common.Problem)
}
