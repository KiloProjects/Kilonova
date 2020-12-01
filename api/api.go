package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/go-chi/chi"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// API is the base struct for the project's API
type API struct {
	ctx             context.Context
	manager         datamanager.Manager
	db              *db.Queries
	testArchiveLock sync.Mutex
}

// New declares a new API instance
func New(ctx context.Context, manager datamanager.Manager, kdb *db.Queries) *API {
	return &API{ctx, manager, kdb, sync.Mutex{}}
}

// GetRouter is the magic behind the API
func (s *API) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	r.Route("/admin", func(r chi.Router) {
		r.Use(s.MustBeAdmin)

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)

		r.Get("/getAllUsers", s.getUsers)
		r.Get("/getAllAdmins", s.getAdmins)
		r.Get("/getAllProposers", s.getProposers)
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
		r.Route("/{id}", func(r chi.Router) {
			r.Use(s.validateProblemID)

			r.Route("/update", func(r chi.Router) {
				r.Use(s.MustBeProposer)
				r.Use(s.validateProblemEditor)

				r.Post("/title", s.updateTitle)
				r.Post("/description", s.updateDescription)
				r.Post("/setConsoleInput", s.setInputType)
				r.Post("/limits", s.setLimits)

				r.Post("/addTest", s.createTest)
				r.Route("/test/{tID}", func(r chi.Router) {
					// test update stuff
					r.Use(s.validateTestID)
					// data:
					r.Post("/data", s.saveTestData)
					// visible id:
					r.Post("/id", s.updateTestID)
					// score:
					r.Post("/score", s.updateTestScore)
				})
				r.Post("/removeTests", s.purgeTests)
				r.Post("/setTestName", s.setTestName)
				r.Post("/processTestArchive", s.processTestArchive)

				r.With(s.MustBeAdmin).Post("/setVisible", s.setProblemVisible)
			})
			r.Route("/get", func(r chi.Router) {
				r.With(s.MustBeAuthed).Get("/selfMaxScore", s.maxScoreSelf)
				r.Group(func(r chi.Router) {
					r.Use(s.MustBeProposer)
					r.Use(s.validateProblemEditor)

					r.Get("/tests", s.getTests)
					r.Get("/test", s.getTest)

					r.Get("/maxScore", s.maxScore)

					r.Get("/testData", s.getTestData)
				})
			})
		})
	})
	r.Route("/submissions", func(r chi.Router) {
		r.Get("/get", s.getSubmissions)
		r.Get("/getByID", s.getSubmissionByID)
		r.Get("/getForProblem", s.getSubmissionsForProblem)
		r.With(s.MustBeAuthed).Get("/getSelfForProblem", s.getSelfSubmissionsForProblem)

		r.With(s.MustBeAuthed).Post("/setVisible", s.setSubmissionVisible)
		r.With(s.MustBeAuthed).Post("/submit", s.submissionSend)
	})
	r.Route("/user", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/setBio", s.setBio)
		r.With(s.MustBeAuthed).Post("/setSubVisibility", s.setSubVisibility)
		r.With(s.MustBeAdmin).Post("/purgeBio", s.purgeBio)

		r.Get("/getByName", s.getUserByName)
		r.With(s.MustBeAuthed).Get("/getSelf", s.getSelf)

		r.Get("/getGravatar", s.getGravatar)
		r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.getSelfGravatar)

		r.With(s.MustBeAuthed).Post("/changeEmail", s.changeEmail)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Method not allowed", 405)
	})

	return r
}
