package api

import (
	"net/http"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/go-chi/chi"
	"github.com/gorilla/schema"
)

var decoder *schema.Decoder

// API is the base struct for the project's API
type API struct {
	kn      *logic.Kilonova
	db      kilonova.DB
	manager kilonova.DataStore

	testArchiveLock *sync.Mutex
}

// New declares a new API instance
func New(kn *logic.Kilonova, db kilonova.DB) *API {
	return &API{kn, db, kn.DM, &sync.Mutex{}}
}

// Handler is the magic behind the API
func (s *API) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	r.With(s.MustBeAdmin).Route("/admin", func(r chi.Router) {

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)
		r.Post("/updateIndex", s.updateIndex)

		r.Route("/maintenance", func(r chi.Router) {
			r.Post("/resetWaitingSubs", s.resetWaitingSubs)
			r.Post("/reevaluateSubmission", s.reevaluateSubmission)
		})

		r.Get("/getAllUsers", s.getUsers)
	})

	r.Route("/auth", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/logout", s.logout)
		r.With(s.MustBeVisitor).Post("/signup", s.signup)
		r.With(s.MustBeVisitor).Post("/login", s.login)
	})
	r.Route("/problem", func(r chi.Router) {
		r.Get("/get", s.getProblems)

		r.With(s.MustBeProposer).Post("/create", s.initProblem)
		r.Get("/maxScore", s.maxScore)

		r.Route("/{id}", func(r chi.Router) {
			r.Use(s.validateProblemID)
			r.Use(s.MustBeProposer)
			r.Use(s.validateProblemEditor)

			r.Route("/update", func(r chi.Router) {
				r.Post("/", s.updateProblem)

				r.Post("/addTest", s.createTest)
				r.Route("/test/{tID}", func(r chi.Router) {
					r.Use(s.validateTestID)
					r.Post("/data", s.saveTestData)
					r.Post("/id", s.updateTestID)
					r.Post("/score", s.updateTestScore)
					r.Post("/orphan", s.orphanTest)
				})

				r.Post("/addAttachment", s.createAttachment)
				//r.With(s.validateAttachmentID).Post("/attachment/{aID}/", s.updateAttachmentMetadata)
				r.Post("/bulkDeleteAttachments", s.bulkDeleteAttachments)

				r.Post("/bulkDeleteTests", s.bulkDeleteTests)
				r.Post("/bulkUpdateTestScores", s.bulkUpdateTestScores)
				r.Post("/orphanTests", s.purgeTests)
				r.Post("/processTestArchive", s.processTestArchive)

				r.Post("/addSubTask", s.createSubTask)
				r.Post("/updateSubTask", s.updateSubTask)
				r.Post("/bulkUpdateSubTaskScores", s.bulkUpdateSubTaskScores)
				r.Post("/bulkDeleteSubTasks", s.bulkDeleteSubTasks)

			})
			r.Route("/get", func(r chi.Router) {
				r.Get("/attachments", s.getAttachments)

				r.Get("/tests", s.getTests)
				r.Get("/test", s.getTest)

				r.Get("/testData", s.getTestData)
			})
			r.Post("/delete", s.deleteProblem)
		})
	})
	// The one from /web/web.go is good enough
	//r.Get("/getAttachment", s.getAttachment)
	r.Route("/submissions", func(r chi.Router) {
		r.Get("/get", s.filterSubs())
		r.Get("/getByID", s.getSubmissionByID())

		r.With(s.MustBeAuthed).Post("/setVisible", s.setSubmissionVisible)
		r.With(s.MustBeAuthed).Post("/setQuality", s.setSubmissionQuality)
		r.With(s.MustBeAuthed).Post("/submit", s.submissionSend)
		r.With(s.MustBeAdmin).Post("/delete", s.deleteSubmission)
	})
	r.Route("/user", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/setSubVisibility", s.setSubVisibility)
		r.With(s.MustBeAuthed).Post("/setBio", s.setBio())

		r.With(s.MustBeAuthed).Post("/resendEmail", s.resendVerificationEmail)

		r.Get("/getByName", s.getUserByName)
		r.With(s.MustBeAuthed).Get("/getSelf", s.getSelf)
		r.With(s.MustBeAuthed).Get("/getSelfSolvedProblems", s.getSelfSolvedProblems)
		r.With(s.MustBeAuthed).Get("/getSolvedProblems", s.getSolvedProblems)

		r.With(s.MustBeAdmin).Route("/moderation", func(r chi.Router) {
			r.Post("/purgeBio", s.purgeBio)
			// TODO
			// r.Post("/nukeUser", s.nukeUser)
			// r.Post("/banUser", s.banUser)
			r.Post("/deleteUser", s.deleteUser)
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
	r.Route("/problemList", func(r chi.Router) {
		r.Get("/get", s.getProblemList)
		r.Get("/filter", s.filterProblemList)
		r.With(s.MustBeProposer).Post("/create", s.initProblemList)
		r.With(s.MustBeAuthed).Post("/update", s.updateProblemList)
		r.With(s.MustBeAuthed).Post("/delete", s.deleteProblemList)
	})
	r.With(s.MustBeAdmin).Route("/kna", func(r chi.Router) {
		r.Get("/createArchive", s.createKNA)
		r.Post("/loadArchive", s.loadKNA)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Method not allowed", 405)
	})

	return r
}

func init() {
	decoder = schema.NewDecoder()
	decoder.SetAliasTag("json")
}
