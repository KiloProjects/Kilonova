// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"bytes"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/mdrenderer"
	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi"
)

var templates *template.Template

//go:embed static
var embedded embed.FS

//go:embed templ
var templateDir embed.FS

var fsys = hashfs.NewFS(embedded)

// Web is the struct representing this whole package
type Web struct {
	kn    *logic.Kilonova
	dm    kilonova.DataStore
	rd    kilonova.MarkdownRenderer
	debug bool

	userv   kilonova.UserService
	sserv   kilonova.SubmissionService
	pserv   kilonova.ProblemService
	tserv   kilonova.TestService
	stkserv kilonova.SubTaskService
	stserv  kilonova.SubTestService
	plserv  kilonova.ProblemListService
}

func (rt *Web) status(w http.ResponseWriter, r *http.Request, statusCode int, err string) {
	Status(w, &StatusParams{
		User:    util.User(r),
		Code:    statusCode,
		Message: err,
	})
}

// Handler returns a http.Handler
// TODO: Split routes in functions
func (rt *Web) Handler() http.Handler {
	r := chi.NewRouter()

	r.Mount("/static", hashfs.FileServer(fsys))

	r.Group(func(r chi.Router) {
		r.Use(rt.getUser)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			log.Println(index.Execute(w, &IndexParams{
				User:    util.User(r),
				Version: kilonova.Version,
				Config:  config.Index,
				ctx:     r.Context(),
				sserv:   rt.sserv,
				plserv:  rt.plserv,
				pserv:   rt.pserv,
				r:       rt.rd,
			}))
		})

		r.Route("/profile", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/", func(w http.ResponseWriter, r *http.Request) {

				pbs, err := kilonova.SolvedProblems(r.Context(), util.User(r).ID, rt.sserv, rt.pserv)
				if err != nil {
					Status(w, &StatusParams{util.User(r), 500, ""})
					return
				}

				Profile(w, &ProfileParams{
					User:         util.User(r),
					ContentUser:  util.User(r),
					UserProblems: pbs,
				})
			})
			r.Route("/{user}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					name := strings.TrimSpace(chi.URLParam(r, "user"))
					users, err := rt.userv.Users(r.Context(), kilonova.UserFilter{Name: &name})
					if err != nil || len(users) == 0 {
						if errors.Is(err, sql.ErrNoRows) || len(users) == 0 {
							Status(w, &StatusParams{util.User(r), 404, ""})
							return
						}
						fmt.Println(err)
						Status(w, &StatusParams{util.User(r), 500, ""})
						return
					}

					pbs, err := kilonova.SolvedProblems(r.Context(), users[0].ID, rt.sserv, rt.pserv)
					if err != nil {
						Status(w, &StatusParams{util.User(r), 500, ""})
						return
					}

					Profile(w, &ProfileParams{
						User:         util.User(r),
						ContentUser:  users[0],
						UserProblems: pbs,
					})
				})
			})
		})

		r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
			settings.Execute(w, &SimpleParams{util.User(r)})
		})

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				pbs.Execute(w, &IndexParams{
					User:    util.User(r),
					Version: kilonova.Version,
					ctx:     r.Context(),
					sserv:   rt.sserv,
					pserv:   rt.pserv,
				})
			})
			r.Route("/{pbid}", func(r chi.Router) {
				r.Use(rt.ValidateProblemID)
				r.Use(rt.ValidateVisible)
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					problem := util.Problem(r)

					buf, err := rt.rd.Render([]byte(problem.Description))
					if err != nil {
						log.Println("Rendering markdown:", err)
					}

					author, err := rt.userv.UserByID(r.Context(), problem.AuthorID)
					if err != nil {
						log.Println("Getting author:", err)
						rt.status(w, r, 500, "Couldn't get author")
						return
					}

					pb.Execute(w, &ProblemParams{
						User:          util.User(r),
						ProblemEditor: util.IsProblemEditor(util.User(r), util.Problem(r)),

						Problem: util.Problem(r),
						Author:  author,

						Markdown:  template.HTML(buf),
						Languages: config.Languages,
					})
				})
				r.Route("/edit", func(r chi.Router) {
					r.Use(rt.mustBeEditor)
					r.Get("/", func(w http.ResponseWriter, r *http.Request) {
						editIndex.Execute(w, &ProblemEditParams{util.User(r), util.Problem(r)})
					})
					r.Get("/desc", func(w http.ResponseWriter, r *http.Request) {
						editDesc.Execute(w, &ProblemEditParams{util.User(r), util.Problem(r)})
					})
					r.Get("/checker", func(w http.ResponseWriter, r *http.Request) {
						editChecker.Execute(w, &ProblemEditParams{util.User(r), util.Problem(r)})
					})
					r.Route("/test", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							testScores.Execute(w, &TestEditParams{util.User(r), util.Problem(r), &kilonova.Test{VisibleID: -2}, rt.tserv, rt.dm})
						})
						r.Get("/add", func(w http.ResponseWriter, r *http.Request) {
							testAdd.Execute(w, &TestEditParams{util.User(r), util.Problem(r), &kilonova.Test{VisibleID: -1}, rt.tserv, rt.dm})
						})
						r.With(rt.ValidateTestID).Get("/{tid}", func(w http.ResponseWriter, r *http.Request) {
							testEdit.Execute(w, &TestEditParams{util.User(r), util.Problem(r), util.Test(r), rt.tserv, rt.dm})
						})
					})
					r.Route("/subtasks", func(r chi.Router) {
						r.Get("/", func(w http.ResponseWriter, r *http.Request) {
							subtaskIndex.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), &kilonova.SubTask{VisibleID: -2}, r.Context(), rt.tserv, rt.stkserv})
						})
						r.Get("/add", func(w http.ResponseWriter, r *http.Request) {
							subtaskAdd.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), &kilonova.SubTask{VisibleID: -1}, r.Context(), rt.tserv, rt.stkserv})
						})
						r.With(rt.ValidateSubTaskID).Get("/{stid}", func(w http.ResponseWriter, r *http.Request) {
							subtaskEdit.Execute(w, &SubTaskEditParams{util.User(r), util.Problem(r), util.SubTask(r), r.Context(), rt.tserv, rt.stkserv})
						})
					})
				})
			})
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				subs.Execute(w, &SimpleParams{util.User(r)})
			})
			r.With(rt.ValidateSubmissionID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				sub.Execute(w, &SubParams{
					User:       util.User(r),
					Submission: util.Submission(r),
				})
			})
		})

		r.Route("/problem_lists", func(r chi.Router) {
			r.With(rt.mustBeProposer).Get("/", func(w http.ResponseWriter, r *http.Request) {
				log.Println(pbListIndex.Execute(w, &ProblemListParams{util.User(r), nil, r.Context(), rt.plserv, rt.pserv, rt.sserv, rt.rd}))
			})
			r.With(rt.mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				pbListCreate.Execute(w, &SimpleParams{util.User(r)})
			})
			r.With(rt.ValidateListID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				log.Println(pbListView.Execute(w, &ProblemListParams{util.User(r), util.ProblemList(r), r.Context(), rt.plserv, rt.pserv, rt.sserv, rt.rd}))
			})
		})

		r.Mount("/docs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := strings.TrimPrefix(r.URL.Path, "/")
			stat, err := fs.Stat(kilonova.Docs, p)
			_, err1 := fs.Stat(kilonova.Docs, p+".md")
			if err != nil && err1 != nil {
				if errors.Is(err, fs.ErrNotExist) && errors.Is(err1, fs.ErrNotExist) {
					Status(w, &StatusParams{util.User(r), 404, "Ce încerci să accesezi nu există"})
					return
				}
				log.Println("CAN'T STAT DOCS", err, err1)
				Status(w, &StatusParams{util.User(r), 500, "N-am putut da stat la path, contactați administratorul"})
				return
			} else if err1 == nil {
				file, err := kilonova.Docs.ReadFile(p + ".md")
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						Status(w, &StatusParams{util.User(r), 404, "Pagina nu există"})
						return
					}
					log.Println("CAN'T OPEN DOCS", err)
					Status(w, &StatusParams{util.User(r), 500, "N-am putut încărca pagina"})
					return
				}

				t, err := rt.rd.Render(file)
				if err != nil {
					log.Println("CAN'T RENDER DOCS")
					Status(w, &StatusParams{util.User(r), 500, "N-am putut randa pagina"})
					return
				}

				markdown.Execute(w, &MarkdownParams{util.User(r), template.HTML(t), "Docs"}) // TODO: Proper title
				return
			}

			if stat.IsDir() {
				file, err := kilonova.Docs.ReadFile(path.Join(p, "index.md"))
				if err != nil {
					entries, err := fs.ReadDir(kilonova.Docs, p)
					if err != nil {
						log.Println("Can't stat dir")
						Status(w, &StatusParams{util.User(r), 404, "Nu-i nimic aici"})
						return
					}
					var data strings.Builder
					for _, entry := range entries {
						data.WriteString("* [")
						data.WriteString(entry.Name())
						if entry.IsDir() {
							data.WriteRune('/')
						}
						data.WriteString("](/")
						data.WriteString(path.Join(p, strings.TrimSuffix(entry.Name(), ".md")))
						data.WriteString(")\n")
					}

					file = []byte(data.String())
				}
				t, err := rt.rd.Render(file)
				if err != nil {
					log.Println("CAN'T RENDER DOCS")
					Status(w, &StatusParams{util.User(r), 500, "N-am putut randa pagina"})
					return
				}

				markdown.Execute(w, &MarkdownParams{util.User(r), template.HTML(t), "Docs"}) // TODO: Proper title
			} else {
				file, err := kilonova.Docs.ReadFile(p)
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						http.Error(w, "Pagina nu există", 404)
						return
					}
					http.Error(w, "N-am putut încărca pagina", 500)
					return
				}
				http.ServeContent(w, r, path.Base(p), time.Now(), bytes.NewReader(file))
			}
		}))

		r.With(rt.mustBeAdmin).Route("/admin", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				AdminPanel(w, util.User(r))
			})
			r.Get("/kna", func(w http.ResponseWriter, r *http.Request) {
				knaPanel.Execute(w, &SimpleParams{util.User(r)})
			})
			r.Get("/makeKNA", func(w http.ResponseWriter, r *http.Request) {
				problems := r.FormValue("pbs")
				if problems == "" {
					http.Error(w, "No problems specified", 400)
					return
				}
				pbIDs, ok := api.DecodeIntString(problems)
				if !ok {
					http.Error(w, "Invalid problem string", 400)
					return
				}
				pbs := make([]*kilonova.Problem, 0, len(pbIDs))
				for _, id := range pbIDs {
					pb, err := rt.pserv.ProblemByID(r.Context(), id)
					if err != nil {
						log.Println(err)
						http.Error(w, "One of the problem IDs is invalid", 400)
						return
					}
					pbs = append(pbs, pb)
				}
				rd, err := kilonova.GenKNA(pbs, rt.tserv, rt.stkserv, rt.dm)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				defer rd.Close()
				w.Header().Add("Content-Type", "application/vnd.sqlite3")
				w.Header().Add("Content-Disposition", `attachment; filename="archive.kna"`)
				http.ServeContent(w, r, "archive.kna", time.Now(), rd)
			})
		})
		r.With(rt.mustBeAdmin).Get("/uitest", func(w http.ResponseWriter, r *http.Request) {
			testUI.Execute(w, &SimpleParams{util.User(r)})
		})

		r.With(rt.mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			login.Execute(w, &SimpleParams{util.User(r)})
		})
		r.With(rt.mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			signup.Execute(w, &SimpleParams{util.User(r)})
		})

		r.With(rt.mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			// i could redirect to /api/auth/logout, but it's easier to do it like this
			rt.kn.RemoveSessionCookie(w, r)
			http.Redirect(w, r, "/", http.StatusFound)
		})

		// Proposer panel
		r.Route("/proposer", func(r chi.Router) {
			r.Use(rt.mustBeProposer)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				proposerPanel.Execute(w, &SimpleParams{util.User(r)})
			})
			r.Route("/get", func(r chi.Router) {
				r.Get("/subtest_output/{st_id}", func(w http.ResponseWriter, r *http.Request) {
					id, err := strconv.Atoi(chi.URLParam(r, "st_id"))
					if err != nil {
						http.Error(w, "Bad ID", 400)
						return
					}
					subtest, err := rt.stserv.SubTest(r.Context(), id)
					if err != nil {
						http.Error(w, "Inexistent subtest", 400)
						return
					}
					sub, err := rt.sserv.SubmissionByID(r.Context(), subtest.SubmissionID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					pb, err := rt.pserv.ProblemByID(r.Context(), sub.ProblemID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					if !util.IsProblemEditor(util.User(r), pb) {
						http.Error(w, "You aren't allowed to do that!", 401)
						return
					}
					rc, err := rt.kn.DM.SubtestReader(subtest.ID)
					if err != nil {
						http.Error(w, "The subtest may have been purged as a routine data-saving process", 404)
						return
					}
					defer rc.Close()
					data, err := io.ReadAll(rc)
					if err != nil {
						http.Error(w, "Internal server error", 500)
						return
					}
					buf := bytes.NewReader(data)
					http.ServeContent(w, r, "subtest.out", time.Now(), buf)
				})
			})
		})

		// Email verification
		r.Route("/verify", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/resend", func(w http.ResponseWriter, r *http.Request) {
				u := util.User(r)
				if u.VerifiedEmail {
					Status(w, &StatusParams{util.User(r), 403, "Deja ai verificat email-ul!"})
					return
				}
				t := time.Now().Sub(u.EmailVerifSentAt.Time)
				if t < 5*time.Minute {
					text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
					Status(w, &StatusParams{util.User(r), 403, text})
					return
				}
				if err := rt.kn.SendVerificationEmail(u.Email, u.Name, u.ID); err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 500, "N-am putut retrimite email-ul de verificare"})
					return
				}

				now := time.Now()
				if err := rt.userv.UpdateUser(r.Context(), u.ID, kilonova.UserUpdate{EmailVerifSentAt: &now}); err != nil {
					log.Println("Couldn't update verification email timestamp:", err)
				}
				sentEmail.Execute(w, &SimpleParams{util.User(r)})
			})
			r.Get("/{vid}", func(w http.ResponseWriter, r *http.Request) {
				vid := chi.URLParam(r, "vid")
				if !rt.kn.CheckVerificationEmail(vid) {
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				uid, err := rt.kn.Verif.GetVerification(r.Context(), vid)
				if err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				user, err := rt.userv.UserByID(r.Context(), uid)
				if err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				// Do this to disable the popup
				if util.User(r) != nil && user.ID == util.User(r).ID {
					util.User(r).VerifiedEmail = true
				}

				if err := rt.kn.ConfirmVerificationEmail(vid, user); err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				VerifiedEmail(w, &VerifiedEmailParams{util.User(r), user})
			})
		})

	})

	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		file, err := embedded.Open("static/robots.txt")
		if err != nil {
			log.Println("Could not open robots.txt")
			return
		}
		http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		Status(w, &StatusParams{util.User(r), 404, ""})
	})

	return r
}

// NewWeb returns a new web instance
func NewWeb(kn *logic.Kilonova, ts kilonova.TypeServicer) *Web {
	rd := mdrenderer.NewLocalRenderer()
	//rd := mdrenderer.NewExternalRenderer("http://0.0.0.0:8040")
	return &Web{kn, kn.DM, rd, kn.Debug,
		ts.UserService(), ts.SubmissionService(), ts.ProblemService(), ts.TestService(), ts.SubTaskService(), ts.SubTestService(), ts.ProblemListService()}
}
