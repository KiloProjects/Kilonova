package api

import (
	"net/http"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/go-chi/chi"
	"github.com/gorilla/schema"
)

var decoder = schema.NewDecoder()

// API is the base struct for the project's API
type API struct {
	kn *logic.Kilonova

	userv  kilonova.UserService
	sserv  kilonova.SubmissionService
	pserv  kilonova.ProblemService
	tserv  kilonova.TestService
	stserv kilonova.SubTestService

	manager kilonova.DataStore

	testArchiveLock sync.Mutex
}

// New declares a new API instance
func New(kn *logic.Kilonova, db kilonova.TypeServicer) *API {
	return &API{kn: kn,
		userv: db.UserService(), sserv: db.SubmissionService(), pserv: db.ProblemService(), tserv: db.TestService(), stserv: db.SubTestService(),
		manager: kn.DM}
}

// Handler is the magic behind the API
func (s *API) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	r.With(s.MustBeAdmin).Route("/admin", func(r chi.Router) {

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)

		r.Route("/maintenance", func(r chi.Router) {
			r.Post("/resetWaitingSubs", s.resetWaitingSubs)
		})

		r.Get("/getAllUsers", s.getUsers)
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
				r.Post("/consoleInput", s.setInputType)
				r.Post("/sourceCredits", s.updateSourceCredits)
				r.Post("/authorCredits", s.updateAuthorCredits)
				r.Post("/limits", s.setLimits)
				r.Post("/defaultPoints", s.setDefaultPoints)

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
					// orphan:
					r.Post("/orphan", s.orphanTest)
				})
				r.Post("/orphanTests", s.purgeTests)
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
		r.Get("/get", s.filterSubs())
		// TODO: Allow getting multiple IDs
		r.Get("/getByID", s.getSubmissionByID())

		r.With(s.MustBeAuthed).Post("/setVisible", s.setSubmissionVisible)
		r.With(s.MustBeAuthed).Post("/submit", s.submissionSend)
	})
	r.Route("/user", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/setSubVisibility", s.setSubVisibility)
		r.With(s.MustBeAuthed).Post("/setBio", s.setBio())

		r.With(s.MustBeAuthed).Post("/resendEmail", s.resendVerificationEmail)

		r.Get("/getByName", s.getUserByName)
		r.With(s.MustBeAuthed).Get("/getSelf", s.getSelf)
		r.With(s.MustBeAuthed).Get("/getSelfSolvedProblems", s.getSelfSolvedProblems)

		r.With(s.MustBeAdmin).Route("/moderation", func(r chi.Router) {
			r.Post("/purgeBio", s.purgeBio)
			// TODO
			// r.Post("/nukeAccount", s.nukeAccount)
			// r.Post("/banAccount", s.banAccount)
		})

		r.Get("/getGravatar", s.getGravatar)
		r.With(s.MustBeAuthed).Get("/getSelfGravatar", s.getSelfGravatar)

		// TODO: Make this secure and maybe with email stuff
		r.With(s.MustBeAuthed).Post("/changeEmail", s.changeEmail)
		r.With(s.MustBeAuthed).Post("/changePassword", s.changePassword)
	})
	r.Route("/cdn", func(r chi.Router) {
		r.Use(s.MustBeProposer)

		r.Post("/saveFile", s.saveCDNFile)
		r.Post("/createDir", s.createCDNDir)
		r.Post("/deleteObject", s.deleteCDNObject)
		r.Get("/readDir", s.readCDNDirectory)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Method not allowed", 405)
	})

	return r
}
