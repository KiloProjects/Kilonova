package server

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
)

// RegisterTaskRoutes mounts the Task routes at /api/tasks
func (s *API) RegisterTaskRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/get", s.GetTasks)

	r.With(s.MustBeAuthed).Post("/submit", s.SubmitTask)
	return r
}

// GetTasks returns all Tasks from the DB
// TODO: Pagination and filtering
func (s *API) GetTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var tasks []common.Task
	s.db.Find(&tasks)
	s.ReturnData(w, "success", tasks)
}

// SubmitTask registers a task to be sent to the Eval handler
// Required values:
//	- code=[sourcecode] - source code of the task, mutually exclusive with file uploads
//  - file=[file] - multipart file, mutually exclusive with the code param
//  - lang=[language] - language key like in common.Languages
//  - problemID=[problem] - problem ID that the task will be associated with
// Note that the `code` param is prioritized over file upload
func (s *API) SubmitTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var code = r.PostFormValue("code")
	var language = r.PostFormValue("lang")
	var user = common.UserFromContext(r)

	// try to read problem
	var problemID = r.PostFormValue("problemID")
	ipbid, _ := strconv.Atoi(problemID)
	if problemID == "" {
		s.ErrorData(w, "No problem specified", http.StatusBadRequest)
		return
	}
	var problem common.Problem
	if s.db.Preload("Tests").First(&problem, ipbid).RecordNotFound() {
		s.ErrorData(w, "Problem not found", http.StatusInternalServerError)
		return
	}

	// validate language
	if _, ok := common.Languages[language]; ok == false {
		s.ErrorData(w, "Invalid language", http.StatusBadRequest)
		return
	}

	// figure out if the code is in a file or in a form value
	if code == "" {
		if r.MultipartForm == nil {
			s.ErrorData(w, "No code sent", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			s.ErrorData(w, "Could not read file", http.StatusBadRequest)
			s.errlog("Could not read multipart file: %v", err.Error())
			return
		}

		if problem.SourceSize != 0 && header.Size > problem.SourceSize {
			s.ErrorData(w, "File too large", http.StatusBadRequest)
		}

		// Everything should be ok now
		c, err := ioutil.ReadAll(file)
		if err != nil {
			s.ErrorData(w, "Could not read file", http.StatusBadRequest)
			s.errlog("Could not read file: %v", err)
			return
		}

		code = string(c)
		if code == "" {
			if r.MultipartForm == nil {
				s.ErrorData(w, "No code sent", http.StatusBadRequest)
				return
			}
		}
	}

	// create the evalTests
	var evalTests = make([]common.EvalTest, 0)
	for _, test := range problem.Tests {
		evTest := common.EvalTest{
			UserID: user.ID,
			Test:   test,
		}
		s.db.Create(&evTest)
		evalTests = append(evalTests, evTest)
	}

	// add the task to the DB
	task := common.Task{
		Tests:      evalTests,
		User:       user,
		Problem:    problem,
		SourceCode: code,
		Language:   language,
	}
	if err := s.db.Create(&task).Error; err != nil {
		s.ErrorData(w, "Couldn't create test", http.StatusInternalServerError)
		s.errlog("Could not create task: %v", err)
		return
	}

	s.StatusData(w, "success", task.ID, http.StatusCreated)
}
