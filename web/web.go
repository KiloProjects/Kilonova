// Package web is the client-side router that manages the website
// If the `server` package interacts with the DB, the `web` package interacts with the user
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"html"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/alecthomas/chroma/v2"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/benbjohnson/hashfs"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var (
	CCDisclaimer = config.GenFlag("feature.frontend.cc_disclaimer", true, "CC disclaimer in footer")

	AllSubsPage = config.GenFlag("feature.frontend.all_subs_page", true, "Anyone can view all submissions")

	FrontPageProblems = config.GenFlag("feature.frontend.front_page_pbs", true, "Show problems on front page")

	ForceLogin = config.GenFlag("behavior.force_authed", false, "Force authentication when accessing website")
)

//go:embed static
var embedded embed.FS

//go:embed templ
var templateDir embed.FS

var fsys = hashfs.NewFS(embedded)

// Web is the struct representing this whole package
type Web struct {
	debug bool

	// db *db.DB

	funcs template.FuncMap

	base *sudoapi.BaseAPI
}

func (rt *Web) statusPage(w http.ResponseWriter, r *http.Request, statusCode int, errMessage string) {
	if r.FormValue("logout") == "1" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	status := rt.parse(nil, "util/statusCode.html", "modals/login.html")
	rt.runTempl(w, r, status, &StatusParams{
		Ctx:     GenContext(r),
		Code:    statusCode,
		Message: errMessage,
	})
}

func (rt *Web) problemRouter(r chi.Router) {
	r.Use(rt.ValidateProblemID)
	r.Use(rt.ValidateProblemVisible)
	r.Get("/", rt.problem())
	r.Get("/submissions", rt.problemSubmissions())
	r.With(rt.mustBeAuthed).Get("/submit", rt.problemSubmit())
	r.With(rt.ValidateProblemFullyVisible).Get("/archive", rt.problemArchive())
	r.With(rt.mustBeProblemEditor).Route("/edit", rt.ProblemEditRouter)
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

	r.Group(func(r chi.Router) {
		// Util group. Will never be locked out

		r.Get("/static/chroma.css", rt.chromaCSS())
		r.Mount("/static", http.HandlerFunc(staticFileServer))

		// Email verification
		r.Route("/verify", func(r chi.Router) {
			r.With(rt.mustBeAuthed).Get("/resend", rt.resendEmail())
			r.Get("/{vid}", rt.verifyEmail())
		})

		// Password reset
		r.With(rt.mustBeVisitor).Get("/resetPassword/{reqid}", rt.resetPassword())

		r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			file, err := embedded.Open("static/robots.txt")
			if err != nil {
				zap.S().Warn("Could not open robots.txt")
				return
			}
			defer file.Close()
			http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
		})

		r.With(rt.mustBeVisitor).Get("/login", rt.justRender("auth/login.html", "modals/login.html"))
		r.With(rt.mustBeVisitor).Get("/signup", rt.justRender("auth/signup.html"))
		r.With(rt.mustBeVisitor).Get("/forgot_pwd", rt.justRender("auth/forgot_pwd_send.html"))

		r.With(rt.mustBeAuthed).Get("/logout", rt.logout)
	})

	r.Group(func(r chi.Router) {
		// Page group, can be locked out
		r.Use(rt.checkLockout())

		r.Get("/", rt.index())
		r.With(rt.mustBeAuthed).Get("/profile", rt.selfProfile())
		r.Get("/profile/{user}", rt.profile())
		r.With(rt.mustBeAuthed).Get("/settings", rt.justRender("settings.html"))
		r.Get("/donate", rt.donationPage())

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", rt.problems())
			r.Route("/{pbid}", rt.problemRouter)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Get("/", rt.blogPosts())
			r.Route("/{postslug}", rt.blogPostRouter)
		})
		// not /posts/create since there could be a post that is slugged "create"
		r.With(rt.mustBeProposer).Get("/createPost", rt.justRender("blogpost/create.html"))

		r.Route("/tags", func(r chi.Router) {
			r.Get("/", rt.tags())
			r.With(rt.ValidateTagID).Get("/{tagid}", rt.tag())
		})

		r.Route("/contests", func(r chi.Router) {
			r.Get("/", rt.contests())
			r.Get("/invite/{inviteID}", rt.contestInvite())
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
				r.Route("/problems/{pbid}", rt.problemRouter)
			})
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Get("/", rt.submissions())
			r.With(rt.ValidateSubmissionID).Get("/{id}", rt.submission())
		})

		r.With(rt.ValidatePasteID).Get("/pastes/{id}", rt.paste())

		r.Route("/problem_lists", func(r chi.Router) {
			r.Get("/", rt.pbListIndex())
			r.Get("/progress", rt.pbListProgressIndex())
			r.With(rt.ValidateListID).Get("/{id}/progress", rt.pbListProgressView())
			r.With(rt.ValidateListID).Get("/{id}", rt.pbListView())
		})

		r.With(rt.mustBeAdmin).Route("/admin", func(r chi.Router) {
			r.Get("/", rt.justRender("admin/admin.html"))
			r.Get("/users", rt.justRender("admin/users.html"))
			r.Get("/auditLog", rt.auditLog())
		})

		// Proposer panel
		r.With(rt.mustBeProposer).Get("/proposer", rt.justRender(
			"proposer/index.html",
			"proposer/createproblem.html", "proposer/importproblem.html",
			"proposer/createpblist.html", "proposer/createcontest.html",
		))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		rt.statusPage(w, r, 404, "")
	})

	return r
}

func (rt *Web) parse(optFuncs template.FuncMap, files ...string) *template.Template {
	if optFuncs == nil {
		return parse(rt.funcs, files...)
	}
	for k, v := range rt.funcs {
		optFuncs[k] = v
	}
	return parse(optFuncs, files...)
}

// NewWeb returns a new web instance
func NewWeb(debug bool, base *sudoapi.BaseAPI) *Web {
	funcs := template.FuncMap{
		"pLanguages": func() map[string]*WebLanguage {
			return webLanguages
		},
		"problemSettings": func(problemID int) *kilonova.ProblemEvalSettings {
			settings, err := base.ProblemSettings(context.Background(), problemID)
			if err != nil {
				zap.S().Warn(err)
				return nil
			}
			return settings
		},
		"problemList": func(id int) *kilonova.ProblemList {
			list, err := base.ProblemList(context.Background(), id)
			if err != nil {
				return nil
			}
			return list
		},
		"unassociatedProblems": func(user *kilonova.UserBrief) []*kilonova.ScoredProblem {
			pbs, err := base.ScoredProblems(context.Background(), kilonova.ProblemFilter{
				LookingUser: user, Look: true, Unassociated: true,
			}, user)
			if err != nil {
				return nil
			}
			return pbs
		},
		"contestProblems": func(user *kilonova.UserBrief, c *kilonova.Contest) []*kilonova.ScoredProblem {
			pbs, err := base.ContestProblems(context.Background(), c, user)
			if err != nil {
				return []*kilonova.ScoredProblem{}
			}
			return pbs
		},
		"problemContests": func(user *kilonova.UserBrief, pb *kilonova.Problem) []*kilonova.Contest {
			// TODO: Once there will be more contests, this will need to be optimized out to exclude ended ones
			// At the moment, however, this is not a priority
			contests, err := base.ProblemRunningContests(context.Background(), pb.ID)
			if err != nil {
				zap.S().Warn(err)
				return nil
			}
			actualContests := make([]*kilonova.Contest, 0, len(contests))
			for _, contest := range contests {
				if base.CanSubmitInContest(user, contest) {
					actualContests = append(actualContests, contest)
				}
			}
			return actualContests
		},
		"decimalFromInt": decimal.NewFromInt,
		"subScore": func(pb *kilonova.Problem, user *kilonova.UserBrief) string {
			if user == nil {
				return ""
			}
			score := base.MaxScore(context.Background(), user.ID, pb.ID)
			if score.IsNegative() {
				return "-"
			}
			return score.StringFixed(pb.ScorePrecision)
		},
		"actualMaxScore": func(pb *kilonova.Problem, user *kilonova.UserBrief) decimal.Decimal {
			return base.MaxScore(context.Background(), user.ID, pb.ID)
		},
		"spbMaxScore": func(pb *kilonova.ScoredProblem) template.HTML {
			if pb.ScoreUserID == nil {
				return ""
			}
			if pb.MaxScore == nil || pb.MaxScore.IsNegative() {
				return "-"
			}
			if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
				if pb.MaxScore.Equal(decimal.NewFromInt(100)) {
					return `<i class="fas fa-fw fa-check"></i>`
				} else {
					return `<i class="fas fa-fw fa-xmark"></i>`
				}
			}
			return template.HTML(pb.MaxScore.StringFixed(pb.ScorePrecision))
		},
		"checklistMaxScore": func(pb *kilonova.ScoredProblem) string {
			if pb.ScoreUserID == nil {
				return "-1"
			}
			if pb.MaxScore == nil || pb.MaxScore.IsNegative() {
				return "-1"
			}
			return pb.MaxScore.StringFixed(pb.ScorePrecision)
		},
		"computeChecklistSpan": computeChecklistSpan,
		"scoreStep": func(pb *kilonova.Problem) string {
			return decimal.NewFromInt(1).Shift(-pb.ScorePrecision).String()
		},
		"contestAnnouncements": func(c *kilonova.Contest) []*kilonova.ContestAnnouncement {
			announcements, err := base.ContestAnnouncements(context.Background(), c.ID)
			if err != nil {
				return []*kilonova.ContestAnnouncement{}
			}
			return announcements
		},
		"isUSACOstyle": func(c *kilonova.Contest) bool {
			return c.PerUserTime > 0
		},
		"startedUSACO": func(c *kilonova.Contest, reg *kilonova.ContestRegistration) bool {
			if c.PerUserTime == 0 {
				return false
			}
			return reg.IndividualStartTime != nil
		},
		"endedUSACO": func(c *kilonova.Contest, reg *kilonova.ContestRegistration) bool {
			if c.PerUserTime == 0 {
				return false
			}
			return reg.IndividualStartTime != nil && reg.IndividualEndTime.Before(time.Now())
		},
		"remainingContestTime": func(c *kilonova.Contest, reg *kilonova.ContestRegistration) time.Time {
			if c.PerUserTime == 0 || reg == nil || reg.IndividualStartTime == nil {
				return c.EndTime
			}
			if time.Now().Before(*reg.IndividualEndTime) {
				return *reg.IndividualEndTime
			}
			return c.EndTime
		},
		"allContestQuestions": func(c *kilonova.Contest) []*kilonova.ContestQuestion {
			questions, err := base.ContestQuestions(context.Background(), c.ID)
			if err != nil {
				return []*kilonova.ContestQuestion{}
			}
			return questions
		},
		"listProblems": func(user *kilonova.UserBrief, list *kilonova.ProblemList) []*kilonova.ScoredProblem {
			pbs, err := base.ProblemListProblems(context.Background(), list.List, user)
			if err != nil {
				return nil
			}
			return pbs
		},
		"pbParentPblists": func(problem *kilonova.Problem) []*kilonova.ProblemList {
			lists, err := base.ProblemParentLists(context.Background(), problem.ID, false)
			if err != nil {
				return nil
			}
			if len(lists) > 5 {
				lists = lists[:5]
			}
			return lists
		},
		"pblistParent": func(pblist *kilonova.ProblemList) []*kilonova.ProblemList {
			lists, err := base.PblistParentLists(context.Background(), pblist.ID)
			if err != nil {
				return nil
			}
			if len(lists) > 5 {
				lists = lists[:5]
			}
			return lists
		},
		"renderMarkdown": func(body any) template.HTML {
			var bd []byte
			switch body := body.(type) {
			case string:
				bd = []byte(body)
			case []byte:
				bd = body
			case template.HTML:
				bd = []byte(body)
			default:
				zap.S().Fatal("Unknown renderMarkdown type")
			}
			val, err := base.RenderMarkdown(bd, nil)
			if err != nil {
				zap.S().Warn(err)
				return "[Error rendering markdown]"
			}
			return template.HTML(val)
		},
		"hasField": func(v any, name string) bool {
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if rv.Kind() != reflect.Struct {
				return false
			}
			return rv.FieldByName(name).IsValid()
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"unescapeHTML": func(s string) string {
			return html.UnescapeString(s)
		},
		"serverTime": func() string {
			return time.Now().Format(time.RFC3339Nano)
		},
		"serverTimeFooter": func() string {
			return time.Now().Format("15:04:05")
		},
		"syntaxHighlight": func(code, lang string) (string, error) {
			fmt := chtml.New(chtml.WithClasses(true), chtml.TabWidth(4))
			lm := lexers.Get(strings.TrimFunc(lang, unicode.IsDigit))
			if lm == nil {
				lm = lexers.Fallback
			}
			lm = chroma.Coalesce(lm)
			var buf bytes.Buffer
			it, err := lm.Tokenise(nil, code)
			if err != nil {
				return "", err
			}
			if err := fmt.Format(&buf, styles.Get("github"), it); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		"submissionEditor": func(user *kilonova.UserBrief, sub *kilonova.Submission) bool {
			return base.IsSubmissionEditor(sub, user)
		},
		"pasteEditor": func(user *kilonova.UserBrief, paste *kilonova.SubmissionPaste) bool {
			return base.IsPasteEditor(paste, user)
		},
		"genProblemsParams": func(pbs []*kilonova.ScoredProblem, showScore, showPublished bool) *ProblemListingParams {
			return &ProblemListingParams{pbs, showScore, showPublished, -1}
		},
		"genContestProblemsParams": func(pbs []*kilonova.ScoredProblem, contest *kilonova.Contest) *ProblemListingParams {
			return &ProblemListingParams{pbs, true, true, contest.ID}
		},
		"genPblistParams": func(pblist *kilonova.ProblemList, open bool) *PblistParams {
			return &PblistParams{pblist, open}
		},
		"numSolvedPbs": func(pbs []*kilonova.ScoredProblem) int {
			var cnt int
			hundred := decimal.NewFromInt(100)
			for _, pb := range pbs {
				if pb.MaxScore != nil && pb.MaxScore.Equal(hundred) {
					cnt++
				}
			}
			return cnt
		},
		"user": func(uid int) *kilonova.UserBrief {
			user, err := base.UserBrief(context.Background(), uid)
			if err != nil {
				return nil
			}
			return user
		},
		"problemEditors": func(problem *kilonova.Problem) []*kilonova.UserBrief {
			users, err := base.ProblemEditors(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return users
		},
		"problemViewers": func(problem *kilonova.Problem) []*kilonova.UserBrief {
			users, err := base.ProblemViewers(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return users
		},
		"problemTests": func(problem *kilonova.Problem) []*kilonova.Test {
			tests, err := base.Tests(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return tests
		},
		"problemSubtasks": func(problem *kilonova.Problem) []*kilonova.SubTask {
			sts, err := base.SubTasks(context.Background(), problem.ID)
			if err != nil {
				return nil
			}
			return sts
		},
		"ispdflink": func(link string) bool {
			u, err := url.Parse(link)
			if err != nil {
				return false
			}
			return path.Ext(u.Path) == ".pdf"
		},
		"encodeJSON": func(data any) (string, error) {
			d, err := json.Marshal(data)
			return base64.StdEncoding.EncodeToString(d), err
		},
		"KBtoMB": func(kb int) float64 {
			return math.Round(float64(kb)/1024.0*100) / 100.0
		},
		"hashedName": fsys.HashName,
		"version":    func() string { return kilonova.Version },
		"debug":      func() bool { return config.Common.Debug },

		"formatCanonical": func(path string) string {
			rez, err := url.JoinPath(config.Common.HostPrefix, path)
			if err != nil {
				zap.S().Warn("Malformed host prefix")
				return path
			}
			return rez
		},

		"tagsByType": func(g string) []*kilonova.Tag {
			tags, err := base.TagsByType(context.Background(), kilonova.TagType(g))
			if err != nil {
				zap.S().Warnf("Couldn't get tags of type %q", g)
				return nil
			}
			return tags
		},
		"problemTags": func(pb *kilonova.Problem) []*kilonova.Tag {
			if pb == nil {
				return nil
			}
			tags, err := base.ProblemTags(context.Background(), pb.ID)
			if err != nil {
				zap.S().Warn("Couldn't get problem tags: ", err)
				return nil
			}
			return tags
		},
		"authorsFromTags": func(tags []*kilonova.Tag) string {
			names := []string{}
			for _, tag := range tags {
				if tag.Type == kilonova.TagTypeAuthor {
					names = append(names, tag.Name)
				}
			}
			return strings.Join(names, ", ")
		},
		"filterTags": func(tags []*kilonova.Tag, tp string, negate bool) []*kilonova.Tag {
			rez := make([]*kilonova.Tag, 0, len(tags))
			for _, tag := range tags {
				if (negate && tag.Type != kilonova.TagType(tp)) || (!negate && tag.Type == kilonova.TagType(tp)) {
					rez = append(rez, tag)
				}
			}
			return rez
		},

		"intFlag": func(name string) int {
			val, ok := config.GetFlagVal[int](name)
			if !ok {
				zap.S().Warnf("Flag with name %q is not int", name)
			}
			return val
		},
		"boolFlag": func(name string) bool {
			val, ok := config.GetFlagVal[bool](name)
			if !ok {
				zap.S().Warnf("Flag with name %q is not bool", name)
			}
			return val
		},

		// for admin configuration
		"defaultLang":  func() string { return config.Common.DefaultLang },
		"testMaxMemMB": func() int { return config.Common.TestMaxMemKB / 1024 },
		"globalMaxMem": func() int64 { return config.Eval.GlobalMaxMem / 1024 },
		"numWorkers":   func() int { return config.Eval.NumConcurrent },

		"bannedHotProblems": func() []int { return config.Frontend.BannedHotProblems },

		"boolFlags":   config.GetFlags[bool],
		"stringFlags": config.GetFlags[string],
		"intFlags":    config.GetFlags[int],

		"validUsername": func(name string) bool { return base.CheckValidUsername(name) == nil },

		// for problem edit page
		"maxMemMB": func() float64 { return float64(config.Common.TestMaxMemKB) / 1024.0 },

		"intList": func(ids []int) string {
			if ids == nil {
				return ""
			}
			var b strings.Builder
			for i, id := range ids {
				b.WriteString(strconv.Itoa(id))
				if i != len(ids)-1 {
					b.WriteRune(',')
				}
			}
			return b.String()
		},
		"problemIDs": func(pbs []*kilonova.ScoredProblem) []int {
			ids := make([]int, 0, len(pbs))
			for _, pb := range pbs {
				ids = append(ids, pb.ID)
			}
			return ids
		},
		"shallowPblistIDs": func(lists []*kilonova.ShallowProblemList) []int {
			rez := []int{}
			for _, l := range lists {
				rez = append(rez, l.ID)
			}
			return rez
		},
		"formatStmtLang": func(lang string) string {
			switch lang {
			case "en":
				return "🇬🇧 English"
			case "ro":
				return "🇷🇴 Română"
			default:
				return lang
			}
		},
		"formatStmtFmt": func(fmt string) string {
			switch fmt {
			case "pdf":
				return "PDF"
			case "md":
				return "Markdown"
			case "tex":
				return "LaTeX"
			default:
				return fmt
			}
		},
		"httpstatus":         http.StatusText,
		"dump":               spew.Sdump,
		"canJoinContest":     base.CanJoinContest,
		"canSubmitInContest": base.CanSubmitInContest,
		"contestDuration": func(c *kilonova.Contest) string {
			d := c.EndTime.Sub(c.StartTime).Round(time.Minute)
			return d.String()
		},
		"usacoDuration": func(c *kilonova.Contest) string {
			return (time.Duration(c.PerUserTime) * time.Second).String()
		},
		"getText": func(key string, vals ...any) string {
			zap.S().Error("Uninitialized `getText`")
			return "FATAL ERR"
		},
		"reqPath": func() string {
			zap.S().Error("Uninitialized `reqPath`")
			return "/"
		},
		"language": func() string {
			zap.S().Error("Uninitialized `language`")
			return "en"
		},
		"isDarkMode": func() bool {
			zap.S().Error("Uninitialized `isDarkMode`")
			return true
		},
		"authed": func() bool {
			zap.S().Error("Uninitialized `authed`")
			return false
		},
		"authedUser": func() *kilonova.UserBrief {
			zap.S().Error("Uninitialized `authedUser`")
			return nil
		},
		"isAdmin": func() bool {
			zap.S().Error("Uninitialized `isAdmin`")
			return false
		},
		"isContestEditor": func(c *kilonova.Contest) bool {
			zap.S().Error("Uninitialized `isContestEditor`")
			return false
		},
		"contestLeaderboardVisible": func(c *kilonova.Contest) bool {
			zap.S().Error("Uninitialized `contestLeaderboardVisible`")
			return false
		},
		"contestProblemsVisible": func(c *kilonova.Contest) bool {
			zap.S().Error("Uninitialized `contestProblemsVisible`")
			return false
		},
		"contestQuestions": func(c *kilonova.Contest) []*kilonova.ContestQuestion {
			zap.S().Error("Uninitialized `contestQuestions`")
			return nil
		},
		"currentProblem": func() *kilonova.Problem {
			zap.S().Error("Uninitialized `currentProblem`")
			return nil
		},
		"canViewAllSubs": func() bool {
			zap.S().Error("Uninitialized `canViewAllSubs`")
			return false
		},
		"contestRegistration": func(c *kilonova.Contest) *kilonova.ContestRegistration {
			zap.S().Error("Uninitialized `contestRegistration`")
			return nil
		},
		"problemFullyVisible": func() bool {
			zap.S().Error("Uninitialized `problemFullyVisible`")
			return false
		},
		"numSolvedPblist": func(listID int) int {
			zap.S().Error("Uninitialized `numSolvedPblist`")
			return -1
		},
	}
	return &Web{debug, funcs, base}
}

var webLanguages map[string]*WebLanguage

type WebLanguage struct {
	Disabled bool   `json:"disabled"`
	Name     string `json:"name"`
	// Extensions []string `json:"extensions"`
}

func init() {
	webLanguages = make(map[string]*WebLanguage)
	for name, lang := range eval.Langs {
		webLanguages[name] = &WebLanguage{
			Disabled: lang.Disabled,
			Name:     lang.PrintableName,
			// Extensions: lang.Extensions,
		}
	}
}

// staticFileServer is a modification of the original hashfs
// This may cause problems if the misc/ directory is updated, but that should be done on rare occasions (since it's mostly fonts)
func staticFileServer(w http.ResponseWriter, r *http.Request) {
	// Clean up filename based on URL path.
	filename := r.URL.Path
	if filename == "/" {
		filename = "."
	} else {
		filename = strings.TrimPrefix(filename, "/")
	}
	filename = path.Clean(filename)

	// Read file from attached file system.
	f, err := fsys.Open(filename)
	if os.IsNotExist(err) {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Fetch file info. Disallow directories from being displayed.
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	} else if fi.IsDir() {
		http.Error(w, "403 Forbidden", http.StatusForbidden)
		return
	}

	trueName, orgHash := hashfs.ParseName(filename)
	_, trueHash := hashfs.ParseName(fsys.HashName(trueName))

	// Cache the file aggressively if the file contains a hash.
	if orgHash != "" {
		w.Header().Set("Cache-Control", `public, max-age=31536000`)
		w.Header().Set("ETag", "\""+trueHash+"\"")
	}

	// Cache the file not-so-aggressively if the file is in the misc directory and has no hash.
	// 2 hours should be good enough
	if orgHash == "" && strings.HasPrefix(trueName, "static/misc/") {
		w.Header().Set("Cache-Control", `public, max-age=7200`)
		w.Header().Set("ETag", "\""+trueHash+"\"")
	}

	// Flush header and write content.
	switch f := f.(type) {
	case io.ReadSeeker:
		http.ServeContent(w, r, filename, fi.ModTime(), f)
	default:
		// Set content length.
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))

		// Flush header and write content.
		w.WriteHeader(http.StatusOK)
		if r.Method != "HEAD" {
			io.Copy(w, f)
		}
	}
}
