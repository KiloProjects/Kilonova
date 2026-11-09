// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova/net/llm"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/statuscode"
	"github.com/go-chi/chi/v5"
)

//go:generate go run ../scripts/chroma_gen -o ./assets/chroma.css

//go:embed static
var embedded embed.FS

// Web is the struct representing this whole package
type Web struct {
	base *sudoapi.BaseAPI

	llmEnabled bool
}

func (rt *Web) statusPage(w http.ResponseWriter, r *http.Request, statusCode int, errMessage string) {
	if r.FormValue("logout") == "1" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if isHTMXRequest(r) {
		htmxErrorToast(w, r, errMessage)
		http.Error(w, errMessage, statusCode)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	rt.runLayout(w, r, &LayoutParams{
		Title:       fmt.Sprintf("Error %d | Kilonova", statusCode),
		Description: errMessage,
		Head:        ui.NoRobots(),
		Content:     statuscode.View(r.Context(), statusCode, errMessage),
	})
}

func (rt *Web) problemRouter(inContest bool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(rt.ValidateProblemID)
		r.Use(rt.ValidateProblemVisible)
		r.Get("/", rt.problem())
		r.Get("/submissions", rt.problemSubmissions())
		if !inContest {
			r.With(
				rt.checkFlag(flags.ProblemStatistics),
				rt.ValidateProblemFullyVisible,
			).Get("/statistics", rt.problemStatistics())
			r.Route("/externalResources", func(r chi.Router) {
				r.Use(rt.ValidateProblemFullyVisible)
				r.Use(rt.checkFlag(flags.ExternalResourcesEnabled))
				r.Get("/", rt.externalResources())
				r.With(rt.mustBeProblemEditor).Get("/create", rt.createExternalResourceView())
				r.With(rt.mustBeProblemEditor).Post("/create", rt.createExternalResource())
				// r.With(rt.ValidateExternalResourceID).Route("/externalResources/{resID}", func(r chi.Router) {
				//	r.Get("/", rt.externalResource())
				// })
			})
		}
		r.With(rt.mustBeAuthed).Get("/submit", rt.problemSubmit())
		r.With(rt.mustBeAuthed, rt.ValidateProblemFullyVisible).Get("/archive", rt.problemArchive())
		r.With(rt.mustBeProblemEditor).Route("/edit", rt.ProblemEditRouter)
	}
}

func (rt *Web) blogPostRouter(r chi.Router) {
	r.Use(rt.ValidateBlogPostSlug)
	r.Use(rt.ValidateBlogPostVisible)
	r.Get("/", rt.blogPost())
	r.Route("/edit", func(r chi.Router) {
		r.Use(rt.mustBePostEditor)
		r.Get("/", rt.editBlogPostIndex())
		r.Get("/attachments", rt.editBlogPostAtts())
	})
}

// Handler returns a http.Handler
func (rt *Web) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(rt.initSession)
	r.Use(rt.initLanguage)
	r.Use(rt.initTheme)
	r.Use(rt.recover)
	csrf := http.NewCrossOriginProtection()

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		rt.statusPage(w, r, 404, "")
	})

	r.Group(func(r chi.Router) {
		// Util group. Will never be locked out

		r.Mount("/static", http.HandlerFunc(staticFileServer))

		// Email verification
		r.Route("/verify", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/resend", rt.resendEmail())
			r.Get("/{vid}", rt.verifyEmail())
		})

		r.Route("/links", func(r chi.Router) {
			r.Get("/discord", rt.redirectDiscord)
		})

		// Password reset
		r.With(rt.mustBeVisitor, csrf.Handler).Get("/resetPassword/{reqid}", rt.resetPassword())

		r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, embedded, "static/robots.txt")
		})

		r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, embedded, "static/favicons/favicon.ico")
		})
		r.Get("/profile/{user}/avatar", rt.profilePicture)
		r.Get("/profile/{user}/gravatar", rt.gravatar)
		r.Get("/profile/{user}/discordAvatar", rt.discordAvatar)

		r.Get("/termsOfService", rt.termsOfService)
		r.Get("/privacyPolicy", rt.privacyPolicy)

		r.With(csrf.Handler).Get("/login", rt.getLogin)
		r.With(csrf.Handler).Post("/login", rt.handleLogin)
		r.With(rt.mustBeVisitor, csrf.Handler).Get("/signup", rt.signup)
		r.With(rt.mustBeVisitor, csrf.Handler).Get("/forgot_pwd", rt.forgotPassword)

		r.With(rt.mustBeAuthed, csrf.Handler).Post("/logout", rt.logout)
		r.With(rt.mustBeAuthed, csrf.Handler).Get("/logout", rt.confirmLogout)

		rt.base.RegisterOIDCRoutes(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(csrf.Handler)

		// Page group, can be locked out
		r.Use(rt.checkLockout())

		r.Get("/", rt.index())
		r.With(rt.mustBeAuthed).Get("/link/discord", rt.discordLink())
		r.With(rt.mustBeAuthed).Get("/profile", rt.selfProfile())
		r.With(rt.mustBeAuthed).Get("/profile/linked", rt.selfLinkStatus())
		r.With(rt.mustBeAuthed).Get("/profile/sessions", rt.selfSessions())
		r.Get("/profile/{user}", rt.profile())
		r.Get("/profile/{user}/submissions", rt.userSubmissions())
		r.With(rt.mustBeAuthed).Get("/profile/{user}/linked", rt.linkStatus())
		r.With(rt.mustBeAuthed).Get("/profile/{user}/sessions", rt.userSessions())
		r.With(rt.mustBeAuthed).Get("/settings", rt.userSettings())
		r.With(rt.mustBeAuthed).Get("/settings/integrations", rt.authorizedApps)
		r.With(rt.mustBeAuthed).Post("/settings/integrations/{clientID}/revoke", rt.revokeAuthorizedApp)
		r.With(rt.mustManageOAuthClients).Route("/settings/oauth", func(r chi.Router) {
			r.Get("/", rt.oauthClientsSettings)
			r.Post("/", rt.createOAuthClient)
			r.Post("/{clientID}/delete", rt.deleteOAuthClient)
		})
		r.With(rt.checkFlag(flags.DonationsEnabled)).Get("/donate", rt.donationPage())
		r.Get("/grader", rt.graderInfo())

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", rt.problems())
			r.Get("/random", rt.randomProblem())
			r.Route("/{pbid}", rt.problemRouter(false))
		})

		r.Route("/posts", func(r chi.Router) {
			r.Get("/", rt.blogPosts())
			r.Route("/{postslug}", rt.blogPostRouter)
		})
		// not /posts/create since there could be a post that is slugged "create"
		r.With(rt.mustBeProposer).Get("/createPost", rt.createPost())

		r.With(rt.canViewTags).Route("/tags", func(r chi.Router) {
			r.Get("/", rt.tags())
			r.With(rt.ValidateTagID).Get("/{tagid}", rt.tag())
		})

		r.Route("/contests", func(r chi.Router) {
			r.Get("/", rt.contests())
			r.Get("/official", rt.contests())
			r.Get("/virtual", rt.contests())
			r.Get("/personal", rt.contests())
			r.With(rt.mustBeAuthed).Get("/create", rt.createContest())
			r.With(rt.mustBeAuthed).Get("/invite/{inviteID}", rt.contestInvite())
			r.Get("/invite/{inviteID}/qr", rt.contestInviteQRCode)
			r.Route("/{contestID}", func(r chi.Router) {
				r.Use(rt.ValidateContestID)
				r.Use(rt.ValidateContestVisible)
				r.Get("/", rt.contest())

				// Communication holds both questions and announcements
				r.Get("/communication", rt.contestCommunication())

				r.Get("/leaderboard", rt.contestLeaderboard())

				r.Route("/manage", func(r chi.Router) {
					r.Use(rt.mustBeContestEditor)
					r.Get("/edit", rt.contestEdit())
					r.Get("/registrations", rt.contestRegistrations())
				})
				r.Route("/problems/{pbid}", rt.problemRouter(true))
			})
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", rt.submissions())
			r.Route("/{subID}", func(r chi.Router) {
				r.Use(rt.ValidateSubmissionID)
				r.Get("/", rt.submission)
				r.Get("/download", rt.downloadSubmission)
				r.Post("/paste", rt.createPaste)
				r.Delete("/", rt.deleteSubmission)
				r.Post("/reevaluate", rt.reevaluateSubmission)
			})
		})

		r.Route("/pastes/{pasteID}", func(r chi.Router) {
			r.Use(rt.checkFlag(flags.PastesEnabled))
			r.Use(rt.ValidatePasteID)
			r.Get("/", rt.paste)
			r.Delete("/", rt.deletePaste)
			r.Get("/download", rt.downloadPaste())
		})

		r.Route("/problem_lists", func(r chi.Router) {
			r.Get("/", rt.pbListIndex())
			r.Get("/progress", rt.pbListProgressIndex())
			r.With(rt.ValidateListID).Get("/{id}/progress", rt.pbListProgressView())
			r.With(rt.ValidateListID).Get("/{id}", rt.pbListView())
			r.With(rt.mustBeAdmin, rt.ValidateListID).Post("/{id}/updateSources", rt.updateProblemSources())
		})

		r.With(rt.mustBeAdmin).Route("/admin", func(r chi.Router) {
			r.Get("/", rt.adminPanel)
			r.Get("/users", rt.usersList())
			r.Get("/user_gen", rt.userGen)
			r.Get("/contest_gen", rt.contestGen)
			r.With(csrf.Handler).Post("/contest_gen", rt.contestGenRun)
			r.Get("/auditLog", rt.auditLog())
			r.Get("/debug", rt.debugPage())
			r.Get("/sessions", rt.sessionsFilter())
			r.Get("/oauth", rt.adminOAuthClients)
		})

		// Proposer panel
		r.With(rt.mustBeProposer).Get("/proposer", rt.proposerPanel)
	})

	return r
}

// NewWeb returns a new web instance
func NewWeb(base *sudoapi.BaseAPI, llmProvider llm.Provider) *Web {
	return &Web{base: base, llmEnabled: llmProvider != nil}
}

func staticFileServer(w http.ResponseWriter, r *http.Request) {
	name := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	// Don't let http.ServeFileFS list the embedded directories.
	if stat, err := fs.Stat(embedded, name); err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(name, "static/misc/") {
		// Vite content-hashes everything it writes in there, so it can never go stale.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	w.Header().Del("Content-Type")
	http.ServeFileFS(w, r, embedded, name)
}

func isHTMXRequest(r *http.Request) bool {
	return (r.Header.Get("HX-Request") == "true" || r.FormValue("force_htmx") == "true") && r.Header.Get("HX-Boosted") != "true"
}

type htmxShowToast struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func setHTMXToast(w http.ResponseWriter, r *http.Request, level string, message string) {
	data := struct {
		ShowToast htmxShowToast `json:"showToast"`
	}{
		htmxShowToast{level, message},
	}
	result, err := json.Marshal(data)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not marshal JSON", slog.Any("err", err))
	}
	w.Header().Set("HX-Trigger", string(result))
}

func htmxSuccessToast(w http.ResponseWriter, r *http.Request, message string) {
	setHTMXToast(w, r, "success", message)
}

func htmxInfoToast(w http.ResponseWriter, r *http.Request, message string) {
	setHTMXToast(w, r, "info", message)
}

func htmxErrorToast(w http.ResponseWriter, r *http.Request, message string) {
	setHTMXToast(w, r, "error", message)
}
