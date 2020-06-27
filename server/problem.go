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
		r.Post("/addTest", func(w http.ResponseWriter, r *http.Request) {
			// TODO: Change to files, or make both options interchangeable
			score, err := strconv.Atoi(r.FormValue("score"))
			if err != nil {
				s.ErrorData(w, "Score not integer", http.StatusBadRequest)
				return
			}
			var test common.Test
			test.ProblemID = s.pbIDFromReq(r)
			test.Score = score
			s.db.Save(&test)
			s.manager.SaveTest(
				s.pbIDFromReq(r),
				test.ID,
				[]byte(r.FormValue("input")),
				[]byte(r.FormValue("output")),
			)
			s.ReturnData(w, "success", test.ID)
		})
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
	})
	return r
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
func (s *API) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			s.ErrorData(w, "invalid problem ID", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), common.PbID, uint(problemID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s API) pbIDFromReq(r *http.Request) uint {
	return s.getContextValue(r, "pbID").(uint)
}
