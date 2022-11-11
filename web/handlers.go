package web

import (
	"bytes"
	"context"
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
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (rt *Web) index() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "index.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &IndexParams{GenContext(r), kilonova.Version, kilonova.IndexDescription})
	}
}

func (rt *Web) problems() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "pbs.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runTempl(w, r, templ, &IndexParams{GenContext(r), kilonova.Version, kilonova.IndexDescription})
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

func (rt *Web) auditLog() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "admin/audit_log.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.FormValue("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			page = 0
		}

		logs, err1 := rt.base.GetAuditLogs(r.Context(), 50, (page-1)*50)
		if err1 != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch logs", false)
			return
		}

		numLogs, err1 := rt.base.GetLogCount(r.Context())
		if err1 != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch log count", false)
			return
		}

		numPages := numLogs / 50
		if numLogs%50 > 0 {
			numPages++
		}

		runTempl(w, r, templ, &AuditLogParams{GenContext(r), logs, numPages})
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

		author, err1 := rt.base.UserBrief(r.Context(), problem.AuthorID)
		if err1 != nil || author == nil {
			log.Println("Getting author:", err, author, problem.AuthorID)
			rt.statusPage(w, r, 500, "Couldn't get author", false)
			return
		}

		atts, err1 := rt.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
		if err1 != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || util.IsProblemEditor(util.UserBrief(r), problem) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		langs := eval.Langs
		if evalSettings, err := rt.base.ProblemSettings(r.Context(), util.Problem(r).ID); err != nil {
			log.Println("Getting problem settings:", err, util.Problem(r).ID)
			rt.statusPage(w, r, 500, "Couldn't get problem settings", false)
		} else if evalSettings.OnlyCPP {
			newLangs := make(map[string]eval.Language)
			for name, lang := range langs {
				if strings.HasPrefix(name, "cpp") {
					newLangs[name] = lang
				}
			}
			langs = newLangs
		} else if evalSettings.OutputOnly {
			newLangs := make(map[string]eval.Language)
			newLangs["outputOnly"] = langs["outputOnly"]
			langs = newLangs
		}

		runTempl(w, r, templ, &ProblemParams{
			Ctx:           GenContext(r),
			ProblemEditor: util.IsProblemEditor(util.UserBrief(r), util.Problem(r)),

			Problem:     util.Problem(r),
			Author:      author,
			Attachments: atts,

			Markdown:  template.HTML(buf),
			Languages: langs,
		})
	}
}

func (rt *Web) selfProfile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pbs, err := rt.base.SolvedProblems(r.Context(), util.UserBrief(r).ID)
		if err != nil {
			rt.Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}
		runTempl(w, r, templ, &ProfileParams{GenContext(r), util.UserFull(r), pbs})
	}
}

func (rt *Web) profile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(chi.URLParam(r, "user")))
		if err != nil {
			zap.S().Warn(err)
			rt.Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}
		if user == nil {
			rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		pbs, err1 := rt.base.SolvedProblems(r.Context(), user.ID)
		if err1 != nil {
			rt.Status(w, &StatusParams{GenContext(r), 500, "", false})
			return
		}

		runTempl(w, r, templ, &ProfileParams{GenContext(r), user, util.FilterVisible(util.UserBrief(r), pbs)})
	}
}

func (rt *Web) resendEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "util/sent.html")
	return func(w http.ResponseWriter, r *http.Request) {
		u := util.UserFull(r)
		if u.VerifiedEmail {
			rt.Status(w, &StatusParams{GenContext(r), 403, "Deja ai verificat emailul!", false})
			return
		}
		t := time.Since(u.EmailVerifResent)
		if t < 5*time.Minute {
			text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
			rt.Status(w, &StatusParams{GenContext(r), 403, text, false})
			return
		}
		if err := rt.base.SendVerificationEmail(context.Background(), u.ID, u.Name, u.Email); err != nil {
			log.Println(err)
			rt.Status(w, &StatusParams{GenContext(r), 500, "N-am putut retrimite email-ul de verificare", false})
			return
		}

		runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) verifyEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "verified-email.html")
	return func(w http.ResponseWriter, r *http.Request) {
		vid := chi.URLParam(r, "vid")
		if !rt.base.CheckVerificationEmail(r.Context(), vid) {
			rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		uid, err := rt.base.GetVerificationUser(r.Context(), vid)
		if err != nil {
			log.Println(err)
			rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		user, err1 := rt.base.UserBrief(r.Context(), uid)
		if err1 != nil {
			log.Println(err1)
			rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		if err := rt.base.ConfirmVerificationEmail(vid, user); err != nil {
			log.Println(err)
			rt.Status(w, &StatusParams{GenContext(r), 404, "", false})
			return
		}

		// rebuild session for user to disable popup
		rt.initSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runTempl(w, r, templ, &VerifiedEmailParams{GenContext(r), user})
		}))
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
	rt.base.RemoveSession(r.Context(), c.Value)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (rt *Web) problemAttachment(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "aid")
	att, err := rt.base.AttachmentByName(r.Context(), util.Problem(r).ID, name)
	if err != nil || att == nil {
		http.Error(w, "The attachment doesn't exist", 400)
		return
	}
	if att.Private && !util.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
		http.Error(w, "You aren't allowed to download the attachment!", 400)
		return
	}

	attData, err := rt.base.AttachmentData(r.Context(), att.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Couldn't get attachment data", 500)
		return
	}

	// If markdown file and client asks for HTML format, render the markdown
	if path.Ext(name) == ".md" && r.FormValue("format") == "html" {
		data, err := rt.rd.Render(attData)
		if err != nil {
			zap.S().Warn(err)
			http.Error(w, "Could not render file", 500)
			return
		}
		http.ServeContent(w, r, att.Name+".html", time.Now(), bytes.NewReader(data))
		return
	}

	http.ServeContent(w, r, att.Name, time.Now(), bytes.NewReader(attData))
}

func (rt *Web) docs() http.HandlerFunc {
	templ := rt.parse(nil, "util/mdrender.html")
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		stat, err := fs.Stat(kilonova.Docs, p)
		_, err1 := fs.Stat(kilonova.Docs, p+".md")
		if err != nil && err1 != nil {
			if errors.Is(err, fs.ErrNotExist) && errors.Is(err1, fs.ErrNotExist) {
				rt.Status(w, &StatusParams{GenContext(r), 404, "Ce încerci să accesezi nu există", false})
				return
			}
			log.Println("CAN'T STAT DOCS", err, err1)
			rt.Status(w, &StatusParams{GenContext(r), 500, "N-am putut da stat la path, contactați administratorul", false})
			return
		} else if err1 == nil {
			file, err := kilonova.Docs.ReadFile(p + ".md")
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					rt.Status(w, &StatusParams{GenContext(r), 404, "Pagina nu există", false})
					return
				}
				log.Println("CAN'T OPEN DOCS", err)
				rt.Status(w, &StatusParams{GenContext(r), 500, "N-am putut încărca pagina", false})
				return
			}

			t, err := rt.rd.Render(file)
			if err != nil {
				log.Println("CAN'T RENDER DOCS")
				rt.Status(w, &StatusParams{GenContext(r), 500, "N-am putut randa pagina", false})
				return
			}

			runTempl(w, r, templ, &MarkdownParams{GenContext(r), template.HTML(t), p}) // TODO: Proper title
			return
		}

		if stat.IsDir() {
			file, err := kilonova.Docs.ReadFile(path.Join(p, "index.md"))
			if err != nil {
				entries, err := fs.ReadDir(kilonova.Docs, p)
				if err != nil {
					log.Println("Can't stat dir")
					rt.Status(w, &StatusParams{GenContext(r), 404, "Nu-i nimic aici", false})
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
				rt.Status(w, &StatusParams{GenContext(r), 500, "N-am putut randa pagina", false})
				return
			}

			runTempl(w, r, templ, &MarkdownParams{GenContext(r), template.HTML(t), p}) // TODO: Proper title
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

func (rt *Web) subtestOutput(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "st_id"))
	if err != nil {
		http.Error(w, "Bad ID", 400)
		return
	}
	subtest, err1 := rt.base.SubTest(r.Context(), id)
	if err1 != nil {
		http.Error(w, "Invalid subtest", 400)
		return
	}
	sub, err1 := rt.base.Submission(r.Context(), subtest.SubmissionID, util.UserBrief(r))
	if err1 != nil {
		log.Println(err1)
		http.Error(w, "You aren't allowed to do that", 500)
		return
	}

	if !util.IsProblemEditor(util.UserBrief(r), sub.Problem) {
		log.Println(err)
		http.Error(w, "You aren't allowed to do that!", 401)
		return
	}

	rc, err := rt.base.SubtestReader(subtest.ID)
	if err != nil {
		http.Error(w, "The subtest may have been purged as a routine data-saving process", 404)
		return
	}
	defer rc.Close()
	http.ServeContent(w, r, "subtest.out", time.Now(), rc)
}

func runTempl(w io.Writer, r *http.Request, templ executor, data interface{}) {
	if err := templ.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Error executing template, report to admin: %s", err)
	}
}
