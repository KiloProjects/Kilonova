package web

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func (rt *Web) index() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "index.html", "modals/pblist.html", "modals/pbs.html", "modals/contest_list.html")
	return func(w http.ResponseWriter, r *http.Request) {
		runningContests, err := rt.base.VisibleRunningContests(r.Context(), util.UserBrief(r))
		if err != nil {
			runningContests = []*kilonova.Contest{}
		}
		futureContests, err := rt.base.VisibleFutureContests(r.Context(), util.UserBrief(r))
		if err != nil {
			futureContests = []*kilonova.Contest{}
		}

		var pblist *kilonova.ProblemList
		if config.Frontend.RootProblemList > 0 {
			pblist, err = rt.base.ProblemList(r.Context(), config.Frontend.RootProblemList)
			if err != nil {
				zap.S().Warn(err)
				pblist = nil
			}
		}

		rt.runTempl(w, r, templ, &IndexParams{GenContext(r), futureContests, runningContests, pblist})
	}
}

func (rt *Web) problems() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "pbs.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) justRender(files ...string) http.HandlerFunc {
	templ := rt.parse(nil, files...)
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) pbListView() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "lists/view.html", "modals/pblist.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ProblemListParams{GenContext(r), util.ProblemList(r)})
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
			rt.statusPage(w, r, 500, "Couldn't fetch logs")
			return
		}

		numLogs, err1 := rt.base.GetLogCount(r.Context())
		if err1 != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch log count")
			return
		}

		numPages := numLogs / 50
		if numLogs%50 > 0 {
			numPages++
		}

		rt.runTempl(w, r, templ, &AuditLogParams{GenContext(r), logs, page, numPages})
	}
}

func (rt *Web) submission() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "submission.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &SubParams{GenContext(r), util.Submission(r)})
	}
}

func (rt *Web) submissions() http.HandlerFunc {
	templ := rt.parse(nil, "submissions.html")
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.canViewAllSubs(util.UserBrief(r)) {
			rt.statusPage(w, r, 401, "Cannot view all submissions")
			return
		}
		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

// canViewAllSubs is just for the text in the navbar and the submissions page
func (rt *Web) canViewAllSubs(user *kilonova.UserBrief) bool {
	if config.Features.AllSubs {
		return true
	}
	return rt.base.IsProposer(user)
}

func (rt *Web) paste() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "paste.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &PasteParams{GenContext(r), util.Paste(r)})
	}
}

func (rt *Web) appropriateDescriptionVariant(pb *kilonova.Problem, user *kilonova.UserBrief, variants []*kilonova.StatementVariant, prefLang string, prefFormat string) (string, string) {
	if len(variants) == 0 {
		return "", ""
	}
	// Search for the ideal scenario
	for _, v := range variants {
		if v.Language == prefLang && v.Format == prefFormat {
			return v.Language, v.Format
		}
	}
	// Then search if anything matches the language
	for _, v := range variants {
		if v.Language == prefLang {
			return v.Language, v.Format
		}
	}
	// Then search if anything matches the format
	for _, v := range variants {
		if v.Language == prefLang {
			return v.Language, v.Format
		}
	}
	// If nothing was found, then just return the first available variant
	return variants[0].Language, variants[0].Format
}

func (rt *Web) problem() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "problem/summary.html", "problem/topbar.html", "modals/contest_sidebar.html", "modals/pb_submit_form.html")
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)

		var statement = []byte("This problem doesn't have a statement.")

		lang := r.FormValue("pref_lang")
		if lang == "" {
			lang = util.Language(r)
		}
		format := r.FormValue("pref_format")
		if format == "" {
			format = "md"
		}

		variants, err := rt.base.ProblemDescVariants(r.Context(), problem.ID, rt.base.IsProblemEditor(util.UserBrief(r), problem))
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Warn("Couldn't get problem desc variants", err)
		}

		foundLang, foundFmt := rt.appropriateDescriptionVariant(problem, util.UserBrief(r), variants, lang, format)

		switch foundFmt {
		case "md":
			data, _, err := rt.base.ProblemRawDesc(r.Context(), problem.ID, foundLang, foundFmt)
			if err != nil {
				zap.S().Warn("Error getting markdown data")
				data = []byte("Error fetching markdown.")
			}
			buf, err := rt.base.RenderMarkdown(data)
			if err != nil {
				zap.S().Warn("Error rendering markdown:", err)
			} else {
				statement = buf
			}
		case "pdf":
			url := fmt.Sprintf("/problems/%d/attachments/statement-%s.%s", problem.ID, foundLang, foundFmt)
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh;"></embed>`,
				url, kilonova.GetText(util.Language(r), "desc_link"), url,
			))
		case "":
		default:
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="/problems/%d/attachments/statement-%s.%s">%s</a>`,
				problem.ID, foundLang, foundFmt,
				kilonova.GetText(util.Language(r), "desc_link"),
			))
		}

		atts, err := rt.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || rt.base.IsProblemEditor(util.UserBrief(r), problem) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		langs := eval.Langs
		if evalSettings, err := rt.base.ProblemSettings(r.Context(), util.Problem(r).ID); err != nil {
			if !errors.Is(err, context.Canceled) {
				zap.S().Warn("Error getting problem settings:", err, util.Problem(r).ID)
			}
			rt.statusPage(w, r, 500, "Couldn't get problem settings")
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

		rt.runTempl(w, r, templ, &ProblemParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "pb_statement", -1),

			Problem:     util.Problem(r),
			Attachments: atts,

			Statement: template.HTML(statement),
			Languages: langs,
			Variants:  variants,

			SelectedLang:   foundLang,
			SelectedFormat: foundFmt,
		})
	}
}

func (rt *Web) problemSubmissions() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "problem/pb_submissions.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ProblemTopbarParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "pb_submissions", -1),

			Problem: util.Problem(r),
		})
	}
}

func (rt *Web) problemSubmit() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "problem/pb_submit.html", "problem/topbar.html", "modals/contest_sidebar.html", "modals/pb_submit_form.html")
	return func(w http.ResponseWriter, r *http.Request) {
		langs := eval.Langs
		if evalSettings, err := rt.base.ProblemSettings(r.Context(), util.Problem(r).ID); err != nil {
			zap.S().Warn("Error getting problem settings:", err, util.Problem(r).ID)
			rt.statusPage(w, r, 500, "Couldn't get problem settings")
			return
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

		rt.runTempl(w, r, templ, &ProblemTopbarParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "pb_submit", -1),

			Languages: langs,
			Problem:   util.Problem(r),
		})
	}
}

func (rt *Web) contests() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/index.html", "modals/contest_list.html")
	return func(w http.ResponseWriter, r *http.Request) {
		contests, err := rt.base.VisibleContests(r.Context(), util.UserBrief(r))
		if err != nil {
			rt.statusPage(w, r, 400, "")
			return
		}
		rt.runTempl(w, r, templ, &ContestsIndexParams{
			Ctx:      GenContext(r),
			Contests: contests,
		})
	}
}

func (rt *Web) contest() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/view.html", "problem/topbar.html", "modals/pbs.html", "modals/contest_sidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "contest_general", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestEdit() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/edit.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "contest_edit", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestCommunication() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/communication.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "contest_communication", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestRegistrations() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/registrations.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "contest_registrations", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestLeaderboard() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "contest/leaderboard.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.topbar(r, "contest_leaderboard", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestLeaderboardCSV(w http.ResponseWriter, r *http.Request) {
	ld, err := rt.base.ContestLeaderboard(r.Context(), util.Contest(r).ID)
	if err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}
	var buf bytes.Buffer
	wr := csv.NewWriter(&buf)

	// Header
	header := []string{"username"}
	for _, pb := range ld.ProblemOrder {
		name, ok := ld.ProblemNames[pb]
		if !ok {
			zap.S().Warn("Invalid rt.base.ContestLeaderboard output")
			http.Error(w, "Invalid internal data", 500)
			continue
		}
		header = append(header, name)
	}
	header = append(header, "total")
	if err := wr.Write(header); err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't write CSV", 500)
		return
	}
	for _, entry := range ld.Entries {
		line := []string{entry.User.Name}
		for _, pb := range ld.ProblemOrder {
			score, ok := entry.ProblemScores[pb]
			if !ok {
				line = append(line, "-")
			} else {
				line = append(line, strconv.Itoa(score))
			}
		}

		line = append(line, strconv.Itoa(entry.TotalScore))
		if err := wr.Write(line); err != nil {
			zap.S().Warn(err)
			http.Error(w, "Couldn't write CSV", 500)
			return
		}
	}

	wr.Flush()
	if err := wr.Error(); err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't write CSV", 500)
		return
	}

	http.ServeContent(w, r, "leaderboard.csv", time.Now(), bytes.NewReader(buf.Bytes()))
}

func (rt *Web) selfProfile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		solvedPbs, err := rt.base.SolvedProblems(r.Context(), util.UserBrief(r))
		if err != nil {
			solvedPbs = []*kilonova.ScoredProblem{}
		}
		attemptedPbs, err := rt.base.AttemptedProblems(r.Context(), util.UserBrief(r))
		if err != nil {
			attemptedPbs = []*kilonova.ScoredProblem{}
		}
		rt.runTempl(w, r, templ, &ProfileParams{
			GenContext(r),
			util.UserFull(r),
			solvedPbs,
			attemptedPbs,
		})
	}
}

func (rt *Web) profile() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "profile.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(chi.URLParam(r, "user")))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			zap.S().Warn(err)
			rt.statusPage(w, r, 500, "")
			return
		}
		if user == nil {
			rt.statusPage(w, r, 404, "")
			return
		}

		solvedPbs, err := rt.base.SolvedProblems(r.Context(), user.Brief())
		if err != nil {
			solvedPbs = []*kilonova.ScoredProblem{}
		}
		attemptedPbs, err := rt.base.AttemptedProblems(r.Context(), user.Brief())
		if err != nil {
			attemptedPbs = []*kilonova.ScoredProblem{}
		}
		rt.runTempl(w, r, templ, &ProfileParams{
			GenContext(r),
			user,
			rt.base.FilterVisibleProblems(util.UserBrief(r), solvedPbs),
			rt.base.FilterVisibleProblems(util.UserBrief(r), attemptedPbs),
		})
	}
}

func (rt *Web) resendEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "util/sent.html")
	return func(w http.ResponseWriter, r *http.Request) {
		u := util.UserFull(r)
		if u.VerifiedEmail {
			rt.statusPage(w, r, 403, "Deja ai verificat emailul!")
			return
		}
		t := time.Since(u.EmailVerifResent)
		if t < 5*time.Minute {
			text := fmt.Sprintf("Trebuie să mai aștepți %s până poți retrimite email de verificare", (5*time.Minute - t).Truncate(time.Millisecond))
			rt.statusPage(w, r, 403, text)
			return
		}
		if err := rt.base.SendVerificationEmail(context.Background(), u.ID, u.Name, u.Email); err != nil {
			zap.S().Warn(err)
			rt.statusPage(w, r, 500, "N-am putut retrimite emailul de verificare")
			return
		}

		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) verifyEmail() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "verified-email.html")
	return func(w http.ResponseWriter, r *http.Request) {
		vid := chi.URLParam(r, "vid")
		if !rt.base.CheckVerificationEmail(r.Context(), vid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetVerificationUser(r.Context(), vid)
		if err != nil {
			zap.S().Warn(err)
			rt.statusPage(w, r, 404, "")
			return
		}

		user, err1 := rt.base.UserBrief(r.Context(), uid)
		if err1 != nil {
			zap.S().Warn(err1)
			rt.statusPage(w, r, 404, "")
			return
		}

		if err := rt.base.ConfirmVerificationEmail(vid, user); err != nil {
			zap.S().Warn(err)
			rt.statusPage(w, r, 404, "")
			return
		}

		// rebuild session for user to disable popup
		rt.initSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rt.runTempl(w, r, templ, &VerifiedEmailParams{GenContext(r), user})
		})).ServeHTTP(w, r)
	}
}

func (rt *Web) resetPassword() func(http.ResponseWriter, *http.Request) {
	templ := rt.parse(nil, "auth/forgot_pwd_reset.html")
	return func(w http.ResponseWriter, r *http.Request) {
		reqid := chi.URLParam(r, "reqid")
		if !rt.base.CheckPasswordResetRequest(r.Context(), reqid) {
			rt.statusPage(w, r, 404, "")
			return
		}

		uid, err := rt.base.GetPwdResetRequestUser(r.Context(), reqid)
		if err != nil {
			zap.S().Warn(err)
			rt.statusPage(w, r, 404, "")
			return
		}

		user, err1 := rt.base.UserFull(r.Context(), uid)
		if err1 != nil {
			zap.S().Warn(err1)
			rt.statusPage(w, r, 404, "")
			return
		}

		rt.runTempl(w, r, templ, &PasswordResetParams{GenContext(r), user, reqid})
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
	if att.Private && !rt.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
		http.Error(w, "You aren't allowed to download the attachment!", 400)
		return
	}

	w.Header().Add("X-Robots-Tag", "noindex, nofollow, noarchive")

	attData, err := rt.base.AttachmentData(r.Context(), att.ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get attachment data", 500)
		return
	}

	// If markdown file and client asks for HTML format, render the markdown
	if path.Ext(name) == ".md" && r.FormValue("format") == "html" {
		data, err := rt.base.RenderMarkdown(attData)
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
			rt.statusPage(w, r, 404, "Ce încerci să accesezi nu există")
			return
		} else if err1 == nil {
			file, err := kilonova.Docs.ReadFile(p + ".md")
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					rt.statusPage(w, r, 404, "Pagina nu există")
					return
				}
				zap.S().Warn("Can't open docs", err)
				rt.statusPage(w, r, 500, "N-am putut încărca pagina")
				return
			}

			t, err1 := rt.base.RenderMarkdown(file)
			if err1 != nil {
				zap.S().Warn("Can't render docs", err1)
				rt.statusPage(w, r, 500, "N-am putut randa pagina")
				return
			}

			rt.runTempl(w, r, templ, &MarkdownParams{GenContext(r), template.HTML(t), p})
			return
		}

		if stat.IsDir() {
			rt.statusPage(w, r, 400, "Can't read dir")
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
		zap.S().Warn(err1)
		http.Error(w, "You aren't allowed to do that", 500)
		return
	}

	if !rt.base.IsProblemEditor(util.UserBrief(r), sub.Problem) {
		http.Error(w, "You aren't allowed to do that!", http.StatusUnauthorized)
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

func (rt *Web) runTempl(w io.Writer, r *http.Request, templ *template.Template, data any) {
	templ, err := templ.Clone()
	if err != nil {
		fmt.Fprintf(w, "Error cloning template, report to admin: %s", err)
		return
	}

	// "cache" most util.* calls
	lang := util.Language(r)
	authedUser := util.UserBrief(r)

	// Add request-specific functions
	templ.Funcs(template.FuncMap{
		"getText": func(line string, args ...any) template.HTML {
			return template.HTML(kilonova.GetText(lang, line, args...))
		},
		"language": func() string {
			return lang
		},
		"isDarkMode": func() bool {
			return util.Theme(r) == kilonova.PreferredThemeDark
		},
		"authed": func() bool {
			return authedUser != nil
		},
		"authedUser": func() *kilonova.UserBrief {
			return authedUser
		},
		"currentProblem": func() *kilonova.Problem {
			return util.Problem(r)
		},
		"isContestEditor": func(c *kilonova.Contest) bool {
			return rt.base.IsContestEditor(authedUser, c)
		},
		"contestQuestions": func(c *kilonova.Contest) []*kilonova.ContestQuestion {
			questions, err := rt.base.ContestUserQuestions(context.Background(), c.ID, authedUser.ID)
			if err != nil {
				return []*kilonova.ContestQuestion{}
			}
			return questions
		},
		"canViewAllSubs": func() bool {
			return rt.canViewAllSubs(authedUser)
		},
		"contestRegistration": func(c *kilonova.Contest) *kilonova.ContestRegistration {
			if authedUser == nil || c == nil {
				return nil
			}
			reg, err := rt.base.ContestRegistration(context.Background(), c.ID, authedUser.ID)
			if err != nil {
				if !errors.Is(err, kilonova.ErrNotFound) {
					zap.S().Warn(err)
				}
				return nil
			}
			return reg
		},
		"problemFullyVisible": func() bool {
			return rt.base.IsProblemFullyVisible(util.UserBrief(r), util.Problem(r))
		},
	})

	if err := templ.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Error executing template, report to admin: %s", err)
		if !strings.Contains(err.Error(), "broken pipe") {
			zap.S().WithOptions(zap.AddCallerSkip(1)).Warnf("Error executing template: %q %q %#v", err, r.URL.Path, util.UserBrief(r))
		}
	}
}
