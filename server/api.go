package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/go-chi/chi"
)

// API is the base struct for the project's API
type API struct {
	ctx     context.Context
	db      *kndb.DB
	config  *common.Config
	manager datamanager.Manager
	logger  *log.Logger
}

// NewAPI declares a new API instance
func NewAPI(ctx context.Context, db *kndb.DB, config *common.Config, manager datamanager.Manager) *API {

	var logger *log.Logger
	if config.Debug {
		logger = log.New(os.Stdout, "[API]", log.LstdFlags)
	}
	return &API{ctx, db, config, manager, logger}
}

// GetRouter is the magic behind the API
func (s *API) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	r.Route("/admin", func(r chi.Router) {
		r.Use(s.MustBeAdmin)

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)

		r.Get("/getAllUsers", s.getUsers)
		r.Get("/getAllAdmins", s.getAdmins)
		r.Get("/getAllProposers", s.getProposers)

		r.Get("/dropAll", s.dropAll)
	})

	r.Route("/auth", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/logout", s.logout)
		r.With(s.MustBeVisitor).Post("/signup", s.signup)
		r.With(s.MustBeVisitor).Post("/login", s.login)
	})
	r.Route("/problem", func(r chi.Router) {
		r.Get("/getAll", s.getAllProblems)
		r.Get("/getByID", s.getProblemByID)

		r.With(s.MustBeProposer).Post("/create", s.initProblem)
		r.With(s.MustBeProposer).Route("/{id}", func(r chi.Router) {
			r.Use(s.validateProblemID)
			r.Use(s.validateProblemEditor)

			r.Route("/update", func(r chi.Router) {
				r.Post("/title", s.updateTitle)
				r.Post("/description", s.updateDescription)
				r.Post("/addTest", s.createTest)
				r.Post("/limits", s.setLimits)
				r.Post("/updateTest", s.updateTest)
				r.Post("/removeTests", s.purgeTests)
				r.Post("/setConsoleInput", s.setInputType)
				r.Post("/setTestName", s.setTestName)
				r.With(s.MustBeAdmin).Post("/setVisible", s.setProblemVisible)
			})
			r.Route("/get", func(r chi.Router) {
				r.Get("/tests", s.getTests)
				r.Get("/test", s.getTest)

				r.With(s.MustBeAuthed).Get("/selfMaxScore", s.maxScoreSelf)
				r.Get("/maxScore", s.maxScore)

				r.Get("/testData", s.getTestData)
			})
		})
	})
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/get", s.getTasks)
		r.Get("/getByID", s.getTaskByID)
		r.Get("/getForProblem", s.getTasksForProblem)
		r.With(s.MustBeAuthed).Get("/getSelfForProblem", s.getSelfTasksForProblem)

		r.With(s.MustBeAuthed).Post("/setVisible", s.setTaskVisible)
		r.With(s.MustBeAuthed).Post("/submit", s.submitTask)
	})
	r.Route("/user", func(r chi.Router) {
		r.Get("/getByName", s.getUserByName)
		r.With(s.MustBeAuthed).Get("/getSelf", s.getSelf)

		r.Get("/getGravatar", s.getGravatar)
		r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.getSelfGravatar)

		r.With(s.MustBeAuthed).Post("/changeEmail", s.changeEmail)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	return r
}
