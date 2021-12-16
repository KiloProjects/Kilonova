package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/archive/kna"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/internal/verification"
	"github.com/go-chi/chi"
)

func (rt *Web) index() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "index.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &IndexParams{GenContext(r), kilonova.Version, config.Index})
	}
}

func (rt *Web) problems() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "pbs.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &IndexParams{GenContext(r), kilonova.Version, config.Index})
	}
}

func (rt *Web) justRender(files ...string) http.HandlerFunc {
	templ := rt.parse(nil, files...)
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) pbListView() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "lists/view.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &ProblemListParams{GenContext(r), util.ProblemList(r)})
	}
}

func (rt *Web) admin() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "admin/admin.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &AdminParams{GenContext(r), config.Index.Description, config.Index.ShowProblems, kilonova.SerializeIntList(config.Index.Lists)})
	}
}

func (rt *Web) submission() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "submission.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &SubParams{GenContext(r), util.Submission(r)})
	}
}

func (rt *Web) problem() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "pb.html")
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)

		buf, err := rt.rd.Render([]byte(problem.Description))
		if err != nil {
			log.Println("Rendering markdown:", err)
		}

		author, err := rt.db.User(r.Context(), problem.AuthorID)
		if err != nil {
			log.Println("Getting author:", err)
			statusPage(w, r, 500, "Couldn't get author", false)
			return
		}

		atts, err := rt.db.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
		if err != nil || len(atts) == 0 {
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

		runTempl(w, r, templ, &ProblemParams{
			Ctx:           GenContext(r),
			ProblemEditor: util.IsProblemEditor(util.User(r), util.Problem(r)),

			Problem:     util.Problem(r),
			Author:      author,
			Attachments: atts,

			Markdown:  template.HTML(buf),
			Languages: eval.Langs,
		})
	}
}

func (rt *Web) selfProfile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pbs, err := rt.db.FullSolvedProblems(r.Context(), util.User(r).ID)
		if err != nil {
			Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}
		runTempl(w, r, templ, &ProfileParams{GenContext(r), util.User(r), pbs})
	}
}

func (rt *Web) profile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html")
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(chi.URLParam(r, "user"))
		users, err := rt.db.Users(r.Context(), kilonova.UserFilter{Name: &name})
		if err != nil {
			fmt.Println(err)
			Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}
		if len(users) == 0 {
			Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		pbs, err := rt.db.FullSolvedProblems(r.Context(), users[0].ID)
		if err != nil {
			Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}

		runTempl(w, r, templ, &ProfileParams{GenContext(r), users[0], util.FilterVisible(util.User(r), pbs)})
	}
}

func (rt *Web) resendEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "util/sent.html")
	return func(w http.ResponseWriter, r *http.Request) {
		u := util.User(r)
		if u.VerifiedEmail {
			Status(w, &StatusParams{GenContext(r), 403, "Deja ai verificat email-ul!", false})
			return
		}
		t := time.Now().Sub(u.EmailVerifSentAt.Time)
		if t < 5*time.Minute {
			text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
			Status(w, &StatusParams{GenContext(r), 403, text, false})
			return
		}
		if err := verification.SendVerificationEmail(u, rt.db, rt.mailer); err != nil {
			log.Println(err)
			Status(w, &StatusParams{GenContext(r), 500, "N-am putut retrimite email-ul de verificare", false})
			return
		}

		now := time.Now()
		if err := rt.db.UpdateUser(r.Context(), u.ID, kilonova.UserUpdate{EmailVerifSentAt: &now}); err != nil {
			log.Println("Couldn't update verification email timestamp:", err)
		}
		runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) verifyEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "verified-email.html")
	return func(w http.ResponseWriter, r *http.Request) {
		vid := chi.URLParam(r, "vid")
		if !verification.CheckVerificationEmail(rt.db, vid) {
			Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		uid, err := rt.db.GetVerification(r.Context(), vid)
		if err != nil {
			log.Println(err)
			Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		user, err := rt.db.User(r.Context(), uid)
		if err != nil {
			log.Println(err)
			Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		// Do this to disable the popup
		if util.User(r) != nil && user.ID == util.User(r).ID {
			util.User(r).VerifiedEmail = true
		}

		if err := verification.ConfirmVerificationEmail(rt.db, vid, user); err != nil {
			log.Println(err)
			Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		runTempl(w, r, templ, &VerifiedEmailParams{GenContext(r), user})
	}
}

func (rt *Web) logout(w http.ResponseWriter, r *http.Request) {
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
}

func (rt *Web) problemAttachment(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "aid")
	att, err := rt.db.Attachments(r.Context(), true, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID, Name: &name})
	if err != nil || att == nil || len(att) == 0 {
		http.Error(w, "The attachment doesn't exist", 400)
		return
	}
	if att[0].Private && !util.IsProblemEditor(util.User(r), util.Problem(r)) {
		http.Error(w, "You aren't allowed to download the attachment!", 400)
		return
	}
	http.ServeContent(w, r, att[0].Name, time.Now(), bytes.NewReader(att[0].Data))
}

func (rt *Web) genKNA(w http.ResponseWriter, r *http.Request) {
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
		if err != nil || pb == nil {
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
}

func (rt *Web) docs() http.HandlerFunc {
	templ := rt.parse(nil, "util/mdrender.html")
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		stat, err := fs.Stat(kilonova.Docs, p)
		_, err1 := fs.Stat(kilonova.Docs, p+".md")
		if err != nil && err1 != nil {
			if errors.Is(err, fs.ErrNotExist) && errors.Is(err1, fs.ErrNotExist) {
				Status(w, &StatusParams{GenContext(r), 404, "Ce încerci să accesezi nu există", false})
				return
			}
			log.Println("CAN'T STAT DOCS", err, err1)
			Status(w, &StatusParams{GenContext(r), 500, "N-am putut da stat la path, contactați administratorul", false})
			return
		} else if err1 == nil {
			file, err := kilonova.Docs.ReadFile(p + ".md")
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					Status(w, &StatusParams{GenContext(r), 404, "Pagina nu există", false})
					return
				}
				log.Println("CAN'T OPEN DOCS", err)
				Status(w, &StatusParams{GenContext(r), 500, "N-am putut încărca pagina", false})
				return
			}

			t, err := rt.rd.Render(file)
			if err != nil {
				log.Println("CAN'T RENDER DOCS")
				Status(w, &StatusParams{GenContext(r), 500, "N-am putut randa pagina", false})
				return
			}

			runTempl(w, r, templ, &MarkdownParams{GenContext(r), template.HTML(t), "Docs"}) // TODO: Proper title
			return
		}

		if stat.IsDir() {
			file, err := kilonova.Docs.ReadFile(path.Join(p, "index.md"))
			if err != nil {
				entries, err := fs.ReadDir(kilonova.Docs, p)
				if err != nil {
					log.Println("Can't stat dir")
					Status(w, &StatusParams{GenContext(r), 404, "Nu-i nimic aici", false})
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
				Status(w, &StatusParams{GenContext(r), 500, "N-am putut randa pagina", false})
				return
			}

			runTempl(w, r, templ, &MarkdownParams{GenContext(r), template.HTML(t), "Docs"}) // TODO: Proper title
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
	}
}

func runTempl(w io.Writer, r *http.Request, templ *template.Template, data interface{}) {
	if err := templ.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Error executing template, report to admin: %s", err)
	}
}
