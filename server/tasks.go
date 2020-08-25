package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/KiloProjects/Kilonova/common"
	"gorm.io/gorm"
)

// GetTaskByID returns a task based on an ID
func (s *API) getTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("id") == "" {
		errorData(w, "No ID specified", http.StatusBadRequest)
		return
	}
	taskID, err := strconv.ParseUint(r.FormValue("id"), 10, 32)
	if err != nil {
		errorData(w, "ID not uint", http.StatusBadRequest)
		return
	}
	task, err := s.db.GetTaskByID(uint(taskID))
	if err != nil {
		errorData(w, "Could not find task", http.StatusBadRequest)
		return
	}

	if !common.IsTaskVisible(*task, common.UserFromContext(r)) {
		task.SourceCode = ""
	}

	returnData(w, "success", *task)
}

// getTasks returns all Tasks from the DB
// TODO: Pagination and filtering
func (s *API) getTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tasks, err := s.db.GetAllTasks()
	if err != nil {
		returnData(w, http.StatusText(500), 500)
		return
	}

	user := common.UserFromContext(r)
	for i := 0; i < len(tasks); i++ {
		if !common.IsTaskVisible(tasks[i], user) {
			tasks[i].SourceCode = ""
		}
	}
	returnData(w, "success", tasks)
}

func (s *API) setTaskVisible(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("visible") == "" {
		errorData(w, "`visible` not specified", http.StatusBadRequest)
		return
	}

	if r.FormValue("id") == "" {
		errorData(w, "`id` not specified", http.StatusBadRequest)
		return
	}

	b, err := strconv.ParseBool(r.FormValue("visible"))
	if err != nil {
		errorData(w, "`visible` not valid bool", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseUint(r.FormValue("id"), 10, 0)
	if err != nil {
		errorData(w, "`id` not valid unsigned int", http.StatusBadRequest)
		return
	}

	task, err := s.db.GetTaskByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Task not found", http.StatusNotFound)
			return
		}
		fmt.Println(err)
		errorData(w, err.Error(), http.StatusNotFound)
		return
	}

	if !common.IsTaskEditor(*task, common.UserFromContext(r)) {
		errorData(w, "You are not allowed to do this", 403)
		return
	}

	if err := s.db.UpdateTaskVisibility(uint(id), b); err != nil {
		errorData(w, err.Error(), 500)
		return
	}

	returnData(w, "success", "Updated visibility status")
}

// submitTask registers a task to be sent to the Eval handler
// Required values:
//	- code=[sourcecode] - source code of the task, mutually exclusive with file uploads
//  - file=[file] - multipart file, mutually exclusive with the code param
//  - lang=[language] - language key like in common.Languages
//  - problemID=[problem] - problem ID that the task will be associated with
// Note that the `code` param is prioritized over file upload
func (s *API) submitTask(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var code = r.FormValue("code")
	var language = r.FormValue("lang")
	var user = common.UserFromContext(r)

	// try to read problem
	var problemID = r.FormValue("problemID")
	ipbid, _ := strconv.ParseUint(problemID, 10, 32)
	if problemID == "" {
		errorData(w, "No problem specified", http.StatusBadRequest)
		return
	}
	problem, err := s.db.GetProblemByID(uint(ipbid))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		// shouldn't happen, but still log it
		//s.errlog("/tasks/submit: Couldn't get problem by ID %d: %s", ipbid, err)
		return
	}

	// validate language
	// TODO: Move language info from protoServer (also move protoServer to another repo idk)
	//if _, ok := protoServer.Languages[language]; ok == false {
	//	errorData(w, "Invalid language", http.StatusBadRequest)
	//	return
	//}

	// figure out if the code is in a file or in a form value
	if code == "" {
		if r.MultipartForm == nil {
			errorData(w, "No code sent", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			//s.errlog("Could not read multipart file: %v", err.Error())
			return
		}

		if problem.SourceSize != 0 && header.Size > problem.SourceSize {
			errorData(w, "File too large", http.StatusBadRequest)
		}

		// Everything should be ok now
		c, err := ioutil.ReadAll(file)
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			//s.errlog("Could not read file: %v", err)
			return
		}

		code = string(c)
		if code == "" {
			if r.MultipartForm == nil {
				errorData(w, "No code sent", http.StatusBadRequest)
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
		errorData(w, "Couldn't create test", 500)
		//s.errlog("Could not create task: %v", err)
		return
	}

	statusData(w, "success", task.ID, http.StatusCreated)
}
