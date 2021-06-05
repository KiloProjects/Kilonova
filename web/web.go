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
	"github.com/KiloProjects/kilonova/archive/kna"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
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
	dm    kilonova.DataStore
	rd    kilonova.MarkdownRenderer
	debug bool

	db     kilonova.DB
	mailer kilonova.Mailer
}

func statusPage(w http.ResponseWriter, r *http.Request, statusCode int, err string) {
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
			index.Execute(w, &IndexParams{
				User:    util.User(r),
				Version: kilonova.Version,
				Config:  config.Index,
				ctx:     r.Context(),
				db:      rt.db,
				r:       rt.rd,
			})
		})

		r.Route("/profile", func(r chi.Router) {
			r.With(mustBeAuthed).Get("/", func(w http.ResponseWriter, r *http.Request) {

				pbs, err := kilonova.SolvedProblems(r.Context(), util.User(r).ID, rt.db)
				if err != nil {
					Status(w, &StatusParams{util.User(r), 500, ""})
					return
				}

				Profile(w, &ProfileParams{
					User:         util.User(r),
					ContentUser:  util.User(r),
					UserProblems: util.FilterVisible(util.User(r), pbs),
				})
			})
			r.Route("/{user}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					name := strings.TrimSpace(chi.URLParam(r, "user"))
					users, err := rt.db.Users(r.Context(), kilonova.UserFilter{Name: &name})
					if err != nil || len(users) == 0 {
						if errors.Is(err, sql.ErrNoRows) || len(users) == 0 {
							Status(w, &StatusParams{util.User(r), 404, ""})
							return
						}
						fmt.Println(err)
						Status(w, &StatusParams{util.User(r), 500, ""})
						return
					}

					pbs, err := kilonova.SolvedProblems(r.Context(), users[0].ID, rt.db)
					if err != nil {
						Status(w, &StatusParams{util.User(r), 500, ""})
						return
					}

					Profile(w, &ProfileParams{
						User:         util.User(r),
						ContentUser:  users[0],
						UserProblems: util.FilterVisible(util.User(r), pbs),
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
					db:      rt.db,
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

					author, err := rt.db.User(r.Context(), problem.AuthorID)
					if err != nil {
						log.Println("Getting author:", err)
						statusPage(w, r, 500, "Couldn't get author")
						return
					}

					atts, err := rt.db.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
					if err != nil {
						if !errors.Is(err, sql.ErrNoRows) {
							log.Println(err)
						}
						atts = nil
					}

					if atts != nil {
						newAtts := make([]*kilonova.Attachment, 0, len(atts))
						for _, att := range atts {
							if att.Visible || util.IsProblemEditor(util.User(r), problem) {
								newAtts = append(newAtts, att)
							}
						}

						atts = newAtts
					}

					pb.Execute(w, &ProblemParams{
						User:          util.User(r),
						ProblemEditor: util.IsProblemEditor(util.User(r), util.Problem(r)),

						Problem:     util.Problem(r),
						Author:      author,
						Attachments: atts,

						Markdown:  template.HTML(buf),
						Languages: eval.Langs,
					})
				})
				r.Get("/attachments/{aid}", func(w http.ResponseWriter, r *http.Request) {
					name := chi.URLParam(r, "aid")
					att, err := rt.db.Attachments(r.Context(), true, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID, Name: &name})
					if err != nil || att == nil || len(att) == 0 {
						http.Error(w, "Atașamentul dorit nu există", 400)
						return
					}
					http.ServeContent(w, r, att[0].Name, time.Now(), bytes.NewReader(att[0].Data))
				})
				r.With(mustBeEditor).Mount("/edit", (&ProblemEditPart{rt.db, rt.dm}).Handler())
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
			r.With(mustBeProposer).Get("/", func(w http.ResponseWriter, r *http.Request) {
				pbListIndex.Execute(w, &ProblemListParams{util.User(r), nil, r.Context(), rt.db, rt.rd})
			})
			r.With(mustBeProposer).Get("/create", func(w http.ResponseWriter, r *http.Request) {
				pbListCreate.Execute(w, &SimpleParams{util.User(r)})
			})
			r.With(rt.ValidateListID).Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
				pbListView.Execute(w, &ProblemListParams{util.User(r), util.ProblemList(r), r.Context(), rt.db, rt.rd})
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

		r.With(mustBeAdmin).Route("/admin", func(r chi.Router) {
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
					pb, err := rt.db.Problem(r.Context(), id)
					if err != nil {
						log.Println(err)
						http.Error(w, "One of the problem IDs is invalid", 400)
						return
					}
					pbs = append(pbs, pb)
				}
				rd, err := kna.Generate(pbs, rt.db, rt.dm)
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

		r.With(rt.ValidateAttachmentID).Get("/attachments/{aid}", func(w http.ResponseWriter, r *http.Request) {
			pb, err := rt.db.Problem(r.Context(), util.Attachment(r).ProblemID)
			if err != nil {
				http.Error(w, "Couldn't get problem", 500)
				return
			}
			// TODO: Private attachments that can't be downloaded (for grader or something else)
			if !util.IsProblemVisible(util.User(r), pb) {
				http.Error(w, "403 Forbidden", 403)
				return
			}

			w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, util.Attachment(r).Name))
			http.ServeContent(w, r, util.Attachment(r).Name, time.Now(), bytes.NewReader(util.Attachment(r).Data))
		})

		r.With(mustBeAdmin).Get("/uitest", func(w http.ResponseWriter, r *http.Request) {
			testUI.Execute(w, &SimpleParams{util.User(r)})
		})

		r.With(mustBeVisitor).Get("/login", func(w http.ResponseWriter, r *http.Request) {
			login.Execute(w, &SimpleParams{util.User(r)})
		})
		r.With(mustBeVisitor).Get("/signup", func(w http.ResponseWriter, r *http.Request) {
			signup.Execute(w, &SimpleParams{util.User(r)})
		})

		r.With(mustBeAuthed).Get("/logout", func(w http.ResponseWriter, r *http.Request) {
			emptyCookie := &http.Cookie{
				Name:    "kn-sessionid",
				Value:   "",
				Path:    "/",
				Expires: time.Unix(0, 0),
			}
			http.SetCookie(w, emptyCookie)

			c, err := r.Cookie("kn-sessionid")
			if err != nil {
				return
			}
			rt.db.RemoveSession(r.Context(), c.Value)
			http.Redirect(w, r, "/", http.StatusFound)
		})

		// Proposer panel
		r.Route("/proposer", func(r chi.Router) {
			r.Use(mustBeProposer)
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
					subtest, err := rt.db.SubTest(r.Context(), id)
					if err != nil {
						http.Error(w, "Inexistent subtest", 400)
						return
					}
					sub, err := rt.db.Submission(r.Context(), subtest.SubmissionID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					pb, err := rt.db.Problem(r.Context(), sub.ProblemID)
					if err != nil {
						log.Println(err)
						http.Error(w, "Internal server error", 500)
						return
					}
					if !util.IsProblemEditor(util.User(r), pb) {
						http.Error(w, "You aren't allowed to do that!", 401)
						return
					}
					rc, err := rt.dm.SubtestReader(subtest.ID)
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
			r.With(mustBeAuthed).Get("/resend", func(w http.ResponseWriter, r *http.Request) {
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
				if err := kilonova.SendVerificationEmail(u.Email, u.Name, u.ID, rt.db, rt.mailer); err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 500, "N-am putut retrimite email-ul de verificare"})
					return
				}

				now := time.Now()
				if err := rt.db.UpdateUser(r.Context(), u.ID, kilonova.UserUpdate{EmailVerifSentAt: &now}); err != nil {
					log.Println("Couldn't update verification email timestamp:", err)
				}
				sentEmail.Execute(w, &SimpleParams{util.User(r)})
			})
			r.Get("/{vid}", func(w http.ResponseWriter, r *http.Request) {
				vid := chi.URLParam(r, "vid")
				if !kilonova.CheckVerificationEmail(rt.db, vid) {
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				uid, err := rt.db.GetVerification(r.Context(), vid)
				if err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				user, err := rt.db.User(r.Context(), uid)
				if err != nil {
					log.Println(err)
					Status(w, &StatusParams{util.User(r), 404, ""})
					return
				}

				// Do this to disable the popup
				if util.User(r) != nil && user.ID == util.User(r).ID {
					util.User(r).VerifiedEmail = true
				}

				if err := kilonova.ConfirmVerificationEmail(rt.db, vid, user); err != nil {
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
func NewWeb(debug bool, db kilonova.DB, dm kilonova.DataStore, mailer kilonova.Mailer) *Web {
	rd := mdrenderer.NewLocalRenderer()
	//rd := mdrenderer.NewExternalRenderer("http://0.0.0.0:8040")
	return &Web{dm, rd, debug, db, mailer}
}
