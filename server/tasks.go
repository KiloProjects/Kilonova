package server

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/proto"
	"github.com/KiloProjects/Kilonova/internal/util"
	"gorm.io/gorm"
)

// GetTaskByID returns a task based on an ID
func (s *API) getTaskByID(w http.ResponseWriter, r *http.Request) {
	taskID, ok := getFormInt(w, r, "id")
	if !ok {
		return
	}

	task, err := s.db.GetTaskByID(uint(taskID))
	if err != nil {
		errorData(w, "Could not find task", http.StatusBadRequest)
		return
	}

	if !util.IsTaskVisible(*task, util.UserFromContext(r)) {
		task.SourceCode = ""
	}

	returnData(w, *task)
}

// getTasks returns all Tasks from the DB
// TODO: Pagination and filtering
func (s *API) getTasks(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	tasks, err := s.db.GetAllTasks()
	if err != nil {
		errorData(w, http.StatusText(500), 500)
		return
	}

	user := util.UserFromContext(r)
	for i := 0; i < len(tasks); i++ {
		if !util.IsTaskVisible(tasks[i], user) {
			tasks[i].SourceCode = ""
		}
	}
	returnData(w, tasks)
}

func (s *API) getTasksForProblem(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		PID uint
		UID uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	tasks, err := s.db.UserTasksOnProblem(args.UID, args.PID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	user, err := s.db.GetUserByID(args.UID)
	if err != nil {
		errorData(w, err, 500)
		return
	}

	for i := 0; i < len(tasks); i++ {
		if !util.IsTaskVisible(tasks[i], *user) {
			tasks[i].SourceCode = ""
		}
	}

	returnData(w, tasks)
}

func (s *API) getSelfTasksForProblem(w http.ResponseWriter, r *http.Request) {
	pid, ok := getFormInt(w, r, "pid")
	if !ok {
		return
	}
	uid := util.UserFromContext(r).ID
	tasks, err := s.db.UserTasksOnProblem(uint(uid), uint(pid))
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, tasks)
}

func (s *API) setTaskVisible(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Visible bool
		ID      uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	task, err := s.db.GetTaskByID(args.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Task not found", http.StatusNotFound)
			return
		}
		s.logger.Println(err)
		errorData(w, err, http.StatusNotFound)
		return
	}

	if !util.IsTaskEditor(*task, util.UserFromContext(r)) {
		errorData(w, "You are not allowed to do this", 403)
		return
	}

	if err := s.db.UpdateTaskVisibility(args.ID, args.Visible); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Updated visibility status")
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
	var args struct {
		Code      string
		Lang      string
		ProblemID uint
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	var user = util.UserFromContext(r)

	problem, err := s.db.GetProblemByID(args.ProblemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			errorData(w, "Problem not found", http.StatusBadRequest)
			return
		}
		return
	}

	if _, ok := proto.Languages[args.Lang]; ok == false {
		errorData(w, "Invalid language", http.StatusBadRequest)
		return
	}

	// figure out if the code is in a file or in a form value
	if args.Code == "" {
		if r.MultipartForm == nil {
			errorData(w, "No code sent", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			return
		}

		if problem.SourceSize != 0 && header.Size > problem.SourceSize {
			errorData(w, "File too large", http.StatusBadRequest)
			return
		}

		// Everything should be ok now
		c, err := ioutil.ReadAll(file)
		if err != nil {
			errorData(w, "Could not read file", http.StatusBadRequest)
			return
		}

		args.Code = string(c)
		if args.Code == "" {
			if r.MultipartForm == nil {
				errorData(w, "No code sent", http.StatusBadRequest)
				return
			}
		}
	}

	// create the evalTests
	var evalTests = make([]models.EvalTest, 0)
	for _, test := range problem.Tests {
		evTest := models.EvalTest{
			UserID: user.ID,
			Test:   test,
		}
		s.db.Save(&evTest)
		evalTests = append(evalTests, evTest)
	}

	// add the task to the DB
	task := models.Task{
		Tests:      evalTests,
		User:       user,
		Problem:    *problem,
		SourceCode: args.Code,
		Language:   args.Lang,
	}
	if err := s.db.Save(&task); err != nil {
		errorData(w, "Couldn't create test", 500)
		return
	}

	statusData(w, "success", task.ID, http.StatusCreated)
}
