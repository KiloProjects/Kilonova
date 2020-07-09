package server

import (
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
)

// RegisterTaskRoutes mounts the Task routes at /api/tasks
func (s *API) RegisterTaskRoutes() chi.Router {
	r := chi.NewRouter()
	// /tasks/get
	r.Get("/get", s.GetTasks)
	// /tasks/getByID
	r.Get("/getByID", s.GetTaskByID)

	// /tasks/submit
	r.With(s.MustBeAuthed).Post("/submit", s.SubmitTask)
	return r
}

// GetTaskByID returns a task based on an ID
func (s *API) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("id") == "" {
		s.ErrorData(w, "No ID specified", http.StatusBadRequest)
		return
	}
	taskID, err := strconv.ParseUint(r.FormValue("id"), 10, 32)
	if err != nil {
		s.ErrorData(w, "ID not uint", http.StatusBadRequest)
		return
	}
	task, err := s.db.GetTaskByID(uint(taskID))
	if err != nil {
		s.ErrorData(w, "Could not find test", http.StatusBadRequest)
		return
	}
	s.ReturnData(w, "success", task)
}

// GetTasks returns all Tasks from the DB
// TODO: Pagination and filtering
func (s *API) GetTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tasks, err := s.db.GetAllTasks()
	if err != nil {
		s.ReturnData(w, http.StatusText(500), 500)
		return
	}
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
	ipbid, _ := strconv.ParseUint(problemID, 10, 32)
	if problemID == "" {
		s.ErrorData(w, "No problem specified", http.StatusBadRequest)
		return
	}
	problem, err := s.db.GetProblemByID(uint(ipbid))
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			s.ErrorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		// shouldn't happen, but still log it
		s.errlog("/tasks/submit: Couldn't get problem by ID %d: %s", ipbid, err)
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
		s.db.Save(&evTest)
		evalTests = append(evalTests, evTest)
	}

	// add the task to the DB
	task := common.Task{
		Tests:      evalTests,
		User:       user,
		Problem:    *problem,
		SourceCode: code,
		Language:   language,
	}
	if err := s.db.Save(&task); err != nil {
		s.ErrorData(w, "Couldn't create test", 500)
		s.errlog("Could not create task: %v", err)
		return
	}

	s.StatusData(w, "success", task.ID, http.StatusCreated)
}
