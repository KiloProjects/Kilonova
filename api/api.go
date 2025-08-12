package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"sync"

	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
)

var decoder *schema.Decoder

// API is the base struct for the project's API
type API struct {
	base *sudoapi.BaseAPI

	signupLock      sync.Mutex
	testArchiveLock sync.Mutex
}

// New declares a new API instance
func New(base *sudoapi.BaseAPI) *API {
	return &API{base: base}
}

func (s *API) HandlerV2() http.Handler {
	prefixURL, err := url.Parse(config.Common.HostPrefix)
	if err != nil {
		panic(err)
	}

	humaConf := huma.DefaultConfig("Kilonova", "2.0")
	humaConf.Servers = []*huma.Server{
		{URL: prefixURL.JoinPath("api/v2").String()},
	}
	humaConf.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		//"session": {
		//	Type: "apiKey",
		//	In:   "cookie",
		//	Name: "kn-sessionid",
		//},
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-Api-Key",
		},
		"oauth": {
			Type:             "openIdConnect",
			OpenIDConnectURL: prefixURL.JoinPath(oidc.DiscoveryEndpoint).String(),
		},
	}
	r := chi.NewRouter()
	api := humachi.New(r, humaConf)
	api.UseMiddleware(s.SetupSessionV2(api))
	api.UseMiddleware(s.CheckScopes(api))

	huma.Register(api, huma.Operation{
		OperationID: "get-problems",
		Method:      http.MethodGet,
		Path:        "/problems",
		Security:    []map[string][]string{},
	}, s.problemGet)

	problemsGroup := huma.NewGroup(api, "/problems")
	problemsGroup.UseMiddleware(s.validateProblemIDv2(problemsGroup))
	problemsGroup.UseSimpleModifier(func(o *huma.Operation) {
		o.Parameters = append(o.Parameters, &huma.Param{
			Name:     "problemID",
			In:       "path",
			Required: true,
			Example:  1,
		})
	})

	huma.Register(problemsGroup, huma.Operation{
		OperationID: "get-problems-by-id",
		Method:      http.MethodGet,
		Path:        "/{problemID}",
		Security:    []map[string][]string{},
	}, s.problemSingleGet)

	huma.Register(problemsGroup, huma.Operation{
		OperationID: "get-problems-languages",
		Method:      http.MethodGet,
		Path:        "/{problemID}/languages",
		Security:    []map[string][]string{},
	}, s.problemLanguagesV2)

	huma.Register(problemsGroup, huma.Operation{
		OperationID: "create-submission",
		Method:      http.MethodPost,
		Path:        "/{problemID}/submit",
		Security:    []map[string][]string{},
		Middlewares: huma.Middlewares{s.MustBeAuthedV2(problemsGroup)},
	}, s.createSubmissionV2)

	return r
}

// HandlerV1 is the magic behind the API
func (s *API) HandlerV1() http.Handler {
	r := chi.NewRouter()
	r.Use(s.SetupSession)
	r.Use(s.filterUserAgent)

	r.With(s.MustBeAdmin).Route("/admin", func(r chi.Router) {

		r.Post("/setAdmin", s.setAdmin)
		r.Post("/setProposer", s.setProposer)

		r.Post("/updateConfig", webMessageWrapper("Updated config. Some changes may only apply after a restart", s.base.UpdateConfig))
		r.Post("/updateFlags", s.updateBoolFlags)

		r.Route("/maintenance", func(r chi.Router) {
			r.Post("/resetWaitingSubs", webMessageWrapper("Reset waiting subs", func(ctx context.Context, _ struct{}) error {
				return s.base.ResetWaitingSubmissions(ctx)
			}))
			r.Post("/mdCacheWarmup", webMessageWrapper("Warmed up cache.", func(ctx context.Context, _ struct{}) error { return s.base.WarmupStatementCache(ctx) }))
			r.Route("/bucket/{bname}", func(r chi.Router) {
				r.Use(s.validateBucket)
				r.Post("/cleanCache", webMessageWrapper("Reset bucket cache", func(ctx context.Context, _ struct{}) error {
					b := util.BucketContext(ctx)
					if !b.Cache() {
						return kilonova.Statusf(403, "Refusing to remove non-cache bucket")
					}
					if err := b.ResetCache(); err != nil {
						slog.WarnContext(ctx, "Could not bucket cache", slog.Any("bucket", b), slog.Any("reason", err))
						return fmt.Errorf("could not reset cache: %w", err)
					}
					return nil
				}))
				r.Post("/evictObjects", webWrapper(func(ctx context.Context, _ struct{}) (string, error) {
					b := util.BucketContext(ctx)
					if b.Persistent() {
						return "", kilonova.Statusf(403, "Refusing to remove important bucket")
					}
					s.base.LogUserAction(ctx, "Attempted running bucket eviction", slog.Any("bucket", b))
					numDeleted, err := b.RunEvictionPolicy(ctx, s.base.EvictionLogger())
					if err != nil {
						slog.WarnContext(ctx, "Could not evict bucket objects", slog.Any("bucket", b), slog.Any("reason", err))
						return "", fmt.Errorf("could not evict objects: %w", err)
					}
					return fmt.Sprintf("Deleted %d objects", numDeleted), nil
				}))
				r.Post("/stats", webWrapper(func(ctx context.Context, args struct {
					Refresh bool `json:"refresh"`
				}) (*datastore.BucketStats, error) {
					return util.BucketContext(ctx).Statistics(args.Refresh), nil
				}))
			})
		})

		r.Post("/exportSubmissions", webWrapper(s.exportSubmissions))

		r.Post("/addDonation", s.addDonation)
		r.Post("/endSubscription", s.endSubscription)

		r.Get("/getAllUsers", s.getAllUsers)
	})

	r.Route("/webhook", func(r chi.Router) {
		r.Post("/bmac_event", s.bmacEvent)
		r.Get("/discord_callback", s.base.HandleDiscordCallback)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Mount("/captcha", s.base.CaptchaImageHandler())

		r.With(s.MustBeAuthed).Post("/logout", s.logout)
		r.With(s.MustBeVisitor).Post("/signup", s.signup)
		r.With(s.MustBeVisitor).Post("/login", s.login)

		r.With(s.MustBeAuthed).Post("/extendSession", s.extendSession)

		r.With(s.MustBeVisitor).Post("/forgotPassword", s.sendForgotPwdMail())
		r.Post("/resetPassword", s.resetPassword)
	})
	r.Route("/problem", func(r chi.Router) {
		r.Post("/get", webWrapper(s.getProblems))
		r.Post("/search", webWrapper(s.searchProblems))

		r.With(s.MustBeProposer).Post("/create", s.initProblem)

		r.With(s.MustBeProposer).Post("/import", s.importProblemArchive)

		r.Route("/{problemID}", func(r chi.Router) {
			r.Use(s.validateProblemID)
			r.Use(s.validateProblemVisible)

			r.Get("/", webWrapper(s.getProblem))
			r.Get("/maxScore", s.maxScore)
			r.Get("/maxScoreBreakdown", s.maxScoreBreakdown)
			r.Get("/languages", webWrapper(s.problemLanguages))
			r.With(s.validateProblemFullyVisible).Get("/tags", webWrapper(s.problemTags))

			r.Group(func(r chi.Router) {
				r.Use(s.validateProblemEditor)
				r.Route("/update", func(r chi.Router) {
					r.Post("/", webMessageWrapper("Updated problem", s.updateProblem))

					r.Post("/addTest", s.createTest)
					r.Route("/test/{tID}", func(r chi.Router) {
						r.Use(s.validateTestID)
						r.Post("/data", s.saveTestData)
						r.Post("/info", s.updateTestInfo)
						r.Post("/delete", webMessageWrapper("Removed test", s.deleteTest))
					})

					r.Post("/tags", webMessageWrapper("Updated tags", s.updateProblemTags))

					r.Post("/addEditor", s.addProblemEditor)
					r.Post("/addViewer", s.addProblemViewer)
					r.Post("/stripAccess", webMessageWrapper("Stripped problem access", s.stripProblemAccess))

					r.Post("/addAttachment", s.createAttachment)
					r.Post("/attachmentData", s.updateAttachmentData)
					r.Post("/bulkDeleteAttachments", s.bulkDeleteAttachments)
					r.Post("/bulkUpdateAttachmentInfo", s.bulkUpdateAttachmentInfo)

					r.Post("/translateStatement", s.translateProblemStatement())

					r.Post("/bulkDeleteTests", s.bulkDeleteTests)
					r.Post("/bulkUpdateTestScores", s.bulkUpdateTestScores)
					r.Post("/processTestArchive", s.processTestArchive)

					r.Post("/addSubTask", s.createSubTask)
					r.Post("/updateSubTask", s.updateSubTask)
					r.Post("/bulkUpdateSubTaskScores", s.bulkUpdateSubTaskScores)
					r.Post("/bulkDeleteSubTasks", s.bulkDeleteSubTasks)
				})

				r.Post("/reevaluateSubs", webMessageWrapper("Reevaluating submissions", func(ctx context.Context, _ struct{}) error {
					return s.base.ResetProblemSubmissions(context.WithoutCancel(ctx), util.ProblemContext(ctx))
				}))

				r.Post("/delete", s.deleteProblem)
			})

			r.Route("/get", func(r chi.Router) {
				r.Get("/attachments", webWrapper(func(ctx context.Context, _ struct{}) ([]*kilonova.Attachment, error) {
					return s.base.ProblemAttachments(ctx, util.ProblemContext(ctx).ID)
				}))
				r.With(s.validateAttachmentID).Get("/attachment/{aID}", webWrapper(s.getFullAttachment))
				r.With(s.validateAttachmentName).Get("/attachmentByName/{aName}", webWrapper(s.getFullAttachment))

				r.With(s.validateProblemEditor).Get("/checklist", webWrapper(func(ctx context.Context, _ struct{}) (*kilonova.ProblemChecklist, error) {
					return s.base.ProblemChecklist(ctx, util.ProblemContext(ctx).ID)
				}))

				r.Get("/accessControl", webWrapper(s.getProblemAccessControl))

				r.Get("/tests", webWrapper(s.getTests))
				r.Get("/test", webWrapper(s.getTest))
			})
		})
	})
	r.Route("/blogPosts", func(r chi.Router) {
		r.Get("/fromUser", s.userBlogPosts)
		r.Get("/bySlug", s.blogPostBySlug)
		r.With(s.MustBeProposer).Post("/create", s.createBlogPost)
		r.Route("/{bpID}", func(r chi.Router) {
			r.Use(s.validateBlogPostID)
			r.Use(s.validateBlogPostVisible)
			r.Get("/", webWrapper(s.blogPostByID))

			r.Route("/update", func(r chi.Router) {
				r.Use(s.validateBlogPostEditor)
				r.Post("/", s.updateBlogPost)

				r.Post("/addAttachment", s.createAttachment)
				r.Post("/attachmentData", s.updateAttachmentData)
				r.Post("/bulkDeleteAttachments", s.bulkDeleteAttachments)
				r.Post("/bulkUpdateAttachmentInfo", s.bulkUpdateAttachmentInfo)
			})

			r.Route("/get", func(r chi.Router) {
				r.Get("/attachments", webWrapper(func(ctx context.Context, _ struct{}) ([]*kilonova.Attachment, error) {
					return s.base.BlogPostAttachments(ctx, util.BlogPostContext(ctx).ID)
				}))
				r.With(s.validateAttachmentID).Get("/attachment/{aID}", webWrapper(s.getFullAttachment))
				r.With(s.validateAttachmentName).Get("/attachmentByName/{aName}", webWrapper(s.getFullAttachment))
			})
			r.With(s.validateBlogPostEditor).Post("/delete", webMessageWrapper("Removed blog post", s.deleteBlogPost))
		})
	})
	r.Route("/submissions", func(r chi.Router) {
		r.Get("/get", s.filterSubs())
		r.Get("/getByID", s.getSubmissionByID())

		r.Route("/{subID}", func(r chi.Router) {
			r.Use(s.validateSubmissionID)

			r.With(s.MustBeAuthed).Post("/createPaste", s.createPaste)
			r.With(s.MustBeAuthed).Post("/delete", webMessageWrapper("Deleted submission", func(ctx context.Context, _ struct{}) error {
				// Check submission permissions
				if !(util.UserBriefContext(ctx).Admin || util.SubmissionContext(ctx).ProblemEditor) {
					return kilonova.Statusf(403, "You cannot delete this submission!")
				}

				return s.base.DeleteSubmission(ctx, util.SubmissionContext(ctx).ID)
			}))
			r.With(s.MustBeAuthed).Post("/reevaluate", webMessageWrapper("Reset submission", func(ctx context.Context, _ struct{}) error {
				// Check submission permissions
				if !(util.UserBriefContext(ctx).Admin || util.SubmissionContext(ctx).ProblemEditor) {
					return kilonova.Statusf(403, "You cannot reevaluate this submission!")
				}

				return s.base.ResetSubmission(context.WithoutCancel(ctx), util.SubmissionContext(ctx).ID)
			}))
		})

		r.With(s.MustBeAuthed).Post("/submit", s.createSubmission)
	})
	r.Route("/paste/{pasteID}", func(r chi.Router) {
		r.Get("/", s.getPaste)
		r.With(s.MustBeAuthed).Post("/delete", s.deletePaste)
	})
	r.Route("/tags", func(r chi.Router) {
		r.Get("/", s.getTags)

		r.Get("/getByID", webWrapper(func(ctx context.Context, args struct {
			ID int `json:"id"`
		}) (*kilonova.Tag, error) {
			return s.base.TagByID(ctx, args.ID)
		}))
		r.Get("/getByName", webWrapper(func(ctx context.Context, args struct {
			Name string `json:"name"`
		}) (*kilonova.Tag, error) {
			return s.base.TagByName(ctx, args.Name)
		}))
		r.With(s.MustBeAdmin).Post("/delete", webMessageWrapper("Deleted tag", func(ctx context.Context, args struct {
			ID int `json:"id"`
		}) error {
			tag, err := s.base.TagByID(ctx, args.ID)
			if err != nil {
				return err
			}
			return s.base.DeleteTag(ctx, tag)
		}))

		r.With(s.MustBeProposer).Post("/create", s.createTag)
		r.With(s.MustBeAdmin).Post("/merge", webMessageWrapper("Merged tags", func(ctx context.Context, args struct {
			ToKeep    int `json:"to_keep"`
			ToReplace int `json:"to_replace"`
		}) error {
			return s.base.MergeTags(ctx, args.ToKeep, []int{args.ToReplace}) // TODO: Many tags
		}))
		r.With(s.MustBeProposer).Post("/update", s.updateTag)
	})
	r.Route("/user", func(r chi.Router) {

		r.With(s.MustBeAuthed).Post("/resendEmail", s.resendVerificationEmail)

		userRouter := chi.NewMux()
		userRouter.Get("/", func(w http.ResponseWriter, r *http.Request) { returnData(w, util.ContentUserBrief(r)) })
		userRouter.Get("/solvedProblems", s.getSolvedProblems)
		userRouter.Get("/gravatar", s.getGravatar)
		userRouter.Get("/avatar", s.getAvatar)
		userRouter.Get("/discordAvatar", s.getDiscordAvatar)
		userRouter.With(s.selfOrAdmin).Post("/deauthAll", s.deauthAllSessions)

		userRouter.With(s.selfOrAdmin).Post("/setBio", s.setBio())
		userRouter.With(s.selfOrAdmin).Post("/setAvatarType", s.setAvatarType())
		userRouter.With(s.selfOrAdmin).Post("/setPreferredLanguage", s.setPreferredLanguage())
		userRouter.With(s.selfOrAdmin).Post("/setPreferredTheme", s.setPreferredTheme())

		userRouter.Route("/moderation", func(r chi.Router) {
			r.Use(s.MustBeAdmin)
			r.Post("/manage", s.manageUser)
			r.Post("/deleteUser", s.deleteUser)
			r.Post("/refreshPassword", webWrapper(s.refreshPassword))
		})

		r.With(s.MustBeAuthed, s.authedContentUser).Mount("/self", userRouter)
		r.With(s.validateUserID).Mount("/byID/{cUID}", userRouter)
		r.With(s.validateUsername).Mount("/byName/{cUName}", userRouter)

		r.With(s.MustBeAuthed).Post("/updateName", s.updateUsername)

		r.With(s.MustBeAdmin).Post("/generateUser", s.generateUser)

		// TODO: Make this secure and maybe with email stuff
		r.With(s.MustBeAuthed).Post("/changeEmail", s.changeEmail)
		r.With(s.MustBeAuthed).Post("/changePassword", s.changePassword)
	})
	r.Route("/problemList", func(r chi.Router) {
		r.Get("/filter", s.problemLists)
		r.Get("/byName", s.problemListByName)
		r.With(s.MustBeProposer).Post("/create", s.initProblemList)

		r.Route("/{pblistID}", func(r chi.Router) {
			r.Use(s.validateProblemListID)
			r.Get("/", webWrapper(s.getProblemList))
			r.Get("/complex", s.getComplexProblemList)

			r.With(s.MustBeAuthed).Post("/update", s.updateProblemList)
			r.With(s.MustBeAuthed).Post("/delete", s.deleteProblemList)

			r.With(s.MustBeAdmin).Post("/toggleProblems", s.togglePblistProblems)
		})
	})

	r.Route("/contest", func(r chi.Router) {
		r.With(s.MustBeAuthed).Post("/create", s.createContest)

		r.With(s.MustBeAuthed).Post("/acceptInvitation", webMessageWrapper("Registered for contest", s.acceptContestInvitation))
		r.With(s.MustBeAuthed).Post("/updateInvitation", webMessageWrapper("Updated invitation", s.updateContestInvitation))

		r.Route("/{contestID}", func(r chi.Router) {
			r.Use(s.validateContestID)
			r.Use(s.validateContestVisible)

			r.Get("/", webWrapper(s.getContest))
			r.Get("/problems", s.getContestProblems)
			r.With(s.MustBeAuthed).Get("/problemRemainingCount", s.getRemainingSubmissionCount)

			r.Get("/leaderboard", s.contestLeaderboard)

			r.Get("/questions", webWrapper(s.contestUserQuestions))
			r.With(s.validateContestEditor).Get("/allQuestions", webWrapper(s.contestAllQuestions))
			r.With(s.validateContestParticipant).Post("/askQuestion", s.askContestQuestion)
			r.With(s.validateContestEditor).Post("/answerQuestion", s.answerContestQuestion)

			r.Get("/announcements", webWrapper(s.contestAnnouncements))
			r.With(s.validateContestEditor).Post("/createAnnouncement", webMessageWrapper("Created announcement", s.createContestAnnouncement))
			r.With(s.validateContestEditor).Post("/updateAnnouncement", webMessageWrapper("Updated announcement", s.updateContestAnnouncement))
			r.With(s.validateContestEditor).Post("/deleteAnnouncement", webMessageWrapper("Removed announcement", s.deleteContestAnnouncement))

			r.With(s.MustBeAuthed).Post("/register", s.registerForContest)
			r.With(s.MustBeAuthed).Post("/startRegistration", s.startContestRegistration)
			r.With(s.validateContestEditor).Post("/runMOSS", webMessageWrapper("Sent submissions to MOSS. It should be done soon", s.runMOSS))

			r.With(s.validateContestEditor).Get("/invitations", webWrapper(func(ctx context.Context, _ struct{}) ([]*kilonova.ContestInvitation, error) {
				return s.base.ContestInvitations(ctx, util.ContestContext(ctx).ID)
			}))
			r.With(s.validateContestEditor).Post("/createInvitation", webWrapper(func(ctx context.Context, args struct {
				MaxUses int `json:"max_uses"`
			}) (string, error) {
				var cnt *int
				if args.MaxUses > 0 {
					cnt = &args.MaxUses
				}
				return s.base.CreateContestInvitation(ctx, util.ContestContext(ctx).ID, util.UserBriefContext(ctx), cnt)
			}))

			r.With(s.MustBeAuthed).Get("/checkRegistration", webWrapper(s.checkRegistration))
			r.With(s.validateContestEditor).Get("/registrations", s.contestRegistrations)
			r.With(s.validateContestEditor).Post("/kickUser", s.stripContestRegistration)
			r.With(s.MustBeAdmin).Post("/forceRegister", s.forceRegisterForContest)
			r.With(s.validateContestEditor).Post("/delete", webMessageWrapper("Deleted contest", func(ctx context.Context, _ struct{}) error {
				return s.base.DeleteContest(ctx, util.ContestContext(ctx))
			}))

			r.Route("/update", func(r chi.Router) {
				r.Use(s.validateContestEditor)

				r.Post("/", s.updateContest)
				r.Post("/problems", s.updateContestProblems)

				r.Post("/addEditor", s.addContestEditor)
				r.Post("/addTester", s.addContestTester)
				r.Post("/stripAccess", s.stripContestAccess)
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Endpoint not found", 404)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		errorData(w, "Method not allowed", 405)
	})

	return r
}

func webWrapper[T1, T2 any](handler func(context.Context, T1) (T2, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var query T1
		if err := parseRequest(r, &query); err != nil {
			statusError(w, err)
			return
		}
		rez, err := handler(r.Context(), query)
		if err != nil {
			statusError(w, err)
			return
		}
		returnData(w, rez)
	}
}

func webMessageWrapper[T1 any](successString string, handler func(context.Context, T1) error) http.HandlerFunc {
	return webWrapper(func(ctx context.Context, args T1) (string, error) {
		if err := handler(ctx, args); err != nil {
			return "", err
		}
		return successString, nil
	})
}

func init() {
	decoder = schema.NewDecoder()
	decoder.SetAliasTag("json")
}

func returnData(w http.ResponseWriter, retData any) {
	kilonova.StatusData(w, "success", retData, 200)
}

func errorData(w http.ResponseWriter, retData any, errCode int) {
	kilonova.StatusData(w, "error", retData, errCode)
}

func statusError(w http.ResponseWriter, err error) {
	kilonova.StatusData(w, "error", err.Error(), kilonova.ErrorCode(err))
}

func parseJSONBody[T any](r *http.Request, output *T) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(output); err != nil {
		return kilonova.Statusf(400, "Invalid JSON input.")
	}
	return nil
}

func parseRequest[T any](r *http.Request, output *T) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	t, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		contentType = "application/octet-stream"
	} else {
		contentType = t
	}

	if contentType == "application/json" {
		return parseJSONBody(r, output)
	}

	if err := r.ParseForm(); err != nil {
		slog.InfoContext(r.Context(), "Form parse error", slog.Any("err", err))
		return kilonova.Statusf(http.StatusBadRequest, "Could not parse form")
	}
	if err := decoder.Decode(output, r.Form); err != nil {
		return kilonova.Statusf(http.StatusBadRequest, "Invalid query parameters: %s", err)
	}
	return nil
}
