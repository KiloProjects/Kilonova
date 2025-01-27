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
	"log/slog"
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/integrations/maxmind"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/alecthomas/chroma/v2"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/benbjohnson/hashfs"
	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	CCDisclaimer    = config.GenFlag("frontend.footer.cc_disclaimer", true, "CC disclaimer in footer")
	DiscordInviteID = config.GenFlag("frontend.footer.discord_id", "Qa6Ytgh", "Invite ID for Discord server")

	AllSubsPage       = config.GenFlag("feature.frontend.all_subs_page", true, "Anyone can view all submissions")
	ViewOtherProfiles = config.GenFlag("feature.frontend.view_other_profiles", true, "Allow anyone to view other profiles")

	FrontPageLatestProblems = config.GenFlag("feature.frontend.front_page_latest_pbs", true, "Show list with latest published problems on front page")
	FrontPageProblems       = config.GenFlag("feature.frontend.front_page_pbs", true, "Show problems on front page")
	FrontPagePbDetails      = config.GenFlag("feature.frontend.front_page_pbs_links", true, "On the front page problems, show links to other resources")
	FrontPageRandomProblem  = config.GenFlag("feature.frontend.front_page_random_pb", true, "On the front page problems, show buttons to draw a random problem")

	FrontPageAnnouncement = config.GenFlag("frontend.front_page_announcement", "default", `Custom front page announcement ("default" = default text)`)

	SidebarContests = config.GenFlag("feature.frontend.front_page_csidebar", true, "Show contests in sidebar on the front page")
	ShowTrending    = config.GenFlag("frontend.front_page.show_trending", true, "Show trending problems on the front page sidebar")

	ForceLogin = config.GenFlag("behavior.force_authed", false, "Force authentication when accessing website")

	GoatCounterDomain = config.GenFlag("feature.analytics.goat_prefix", "https://goat.kilonova.ro", "URL prefix for GoatCounter analytics")
	TwiplaID          = config.GenFlag("feature.analytics.twipla_id", "", "ID for TWIPLA Analytics integration")
	FaroID            = config.GenFlag("feature.analytics.faro_id", "", "ID for Grafana Faro integration")

	NavbarBranding = config.GenFlag("frontend.navbar.branding", "Kilonova", "Branding in navbar")

	FeedbackURL    = config.GenFlag("feature.frontend.feedback_url", "", "Feedback URL for main page")
	QuickSearchBox = config.GenFlag("feature.frontend.quick_search", false, "Quick search box on main page")
)

//go:generate go run ../scripts/chroma_gen -o ./static/chroma.css

//go:embed static
var embedded embed.FS

//go:embed templ
var templateDir embed.FS

var fsys = hashfs.NewFS(embedded)

// Web is the struct representing this whole package
type Web struct {
	funcs template.FuncMap

	base *sudoapi.BaseAPI
}

func (rt *Web) statusPage(w http.ResponseWriter, r *http.Request, statusCode int, errMessage string) {
	if r.FormValue("logout") == "1" {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if isHTMXRequest(r) {
		http.Error(w, errMessage, statusCode)
		return
	}
	status := rt.parse(nil, "util/statusCode.html", "modals/login.html")
	rt.runTempl(w, r, status, &StatusParams{
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
				slog.WarnContext(r.Context(), "Could not open robots.txt")
				return
			}
			defer file.Close()
			http.ServeContent(w, r, "robots.txt", time.Now(), file.(io.ReadSeeker))
		})

		r.Get("/termsOfService", rt.justRender("util/termsOfService.html"))
		r.Get("/privacyPolicy", rt.justRender("util/privacyPolicy.html"))

		r.With(rt.mustBeVisitor).Get("/login", rt.justRender("auth/login.html", "modals/login.html"))
		r.With(rt.mustBeVisitor).Get("/signup", rt.justRender("auth/signup.html"))
		r.With(rt.mustBeVisitor).Get("/forgot_pwd", rt.justRender("auth/forgot_pwd_send.html"))

		r.With(rt.mustBeAuthed).Get("/logout", rt.logout)
	})

	r.Group(func(r chi.Router) {
		// Page group, can be locked out
		r.Use(rt.checkLockout())

		r.Get("/", rt.index())
		r.With(rt.mustBeAuthed).Get("/link/discord", rt.discordLink())
		r.With(rt.mustBeAuthed).Get("/profile", rt.selfProfile())
		r.With(rt.mustBeAuthed).Get("/profile/linked", rt.selfLinkStatus())
		r.With(rt.mustBeAuthed).Get("/profile/sessions", rt.selfSessions())
		r.Get("/profile/{user}", rt.profile())
		r.With(rt.mustBeAuthed).Get("/profile/{user}/linked", rt.linkStatus())
		r.With(rt.mustBeAuthed).Get("/profile/{user}/sessions", rt.userSessions())
		r.With(rt.mustBeAuthed).Get("/settings", rt.justRender("settings.html"))
		r.Get("/donate", rt.donationPage())
		r.Get("/grader", rt.graderInfo())

		r.Route("/problems", func(r chi.Router) {
			r.Get("/", rt.problems())
			r.Get("/random", rt.randomProblem())
			r.Route("/{pbid}", rt.problemRouter)
		})

		r.Route("/posts", func(r chi.Router) {
			r.Get("/", rt.blogPosts())
			r.Route("/{postslug}", rt.blogPostRouter)
		})
		// not /posts/create since there could be a post that is slugged "create"
		r.With(rt.mustBeProposer).Get("/createPost", rt.justRender("blogpost/create.html"))

		r.With(rt.canViewTags).Route("/tags", func(r chi.Router) {
			r.Get("/", rt.tags())
			r.With(rt.ValidateTagID).Get("/{tagid}", rt.tag())
		})

		r.Route("/contests", func(r chi.Router) {
			r.Get("/", rt.contests())
			r.With(rt.mustBeAuthed).Get("/create", rt.createContest())
			r.With(rt.mustBeAuthed).Get("/invite/{inviteID}", rt.contestInvite())
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
			r.Get("/user_gen", rt.justRender("admin/user_gen.html"))
			r.Get("/auditLog", rt.auditLog())
			r.Get("/debug", rt.debugPage())
			r.Get("/sessions", rt.sessionsFilter())
			r.Get("/problemQueue", rt.problemQueue())
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

func (rt *Web) parseModal(optFuncs template.FuncMap, files ...string) *template.Template {
	if optFuncs == nil {
		return parseModal(rt.funcs, files...)
	}
	for k, v := range rt.funcs {
		optFuncs[k] = v
	}
	return parseModal(optFuncs, files...)
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
func NewWeb(base *sudoapi.BaseAPI) *Web {
	ctx := context.Background()
	hostURL, err := url.Parse(config.Common.HostPrefix)
	if err != nil {
		hostURL, _ = url.Parse("localhost:8080")
		slog.ErrorContext(ctx, "Invalid host prefix", slog.Any("err", err))
	}

	funcs := template.FuncMap{
		"problemSettings": func(problemID int) *kilonova.ProblemEvalSettings {
			settings, err := base.ProblemSettings(ctx, problemID)
			if err != nil {
				slog.WarnContext(ctx, "Could not get problem settings", slog.Any("err", err))
				return nil
			}
			return settings
		},
		"problemList": func(id int) *kilonova.ProblemList {
			list, err := base.ProblemList(ctx, id)
			if err != nil {
				return nil
			}
			return list
		},
		"unassociatedProblems": func(user *kilonova.UserBrief) []*kilonova.ScoredProblem {
			pbs, err := base.ScoredProblems(ctx, kilonova.ProblemFilter{
				LookingUser: user, Look: true, Unassociated: true,
			}, user, user)
			if err != nil {
				return nil
			}
			return pbs
		},
		"contestProblems": func(user *kilonova.UserBrief, c *kilonova.Contest) []*kilonova.ScoredProblem {
			pbs, err := base.ContestProblems(ctx, c, user)
			if err != nil {
				return []*kilonova.ScoredProblem{}
			}
			return pbs
		},
		"problemFromList": func(pbs []*kilonova.ScoredProblem, id int) *kilonova.ScoredProblem {
			for _, pb := range pbs {
				if pb.ID == id {
					return pb
				}
			}
			return nil
		},
		"problemContests": func(user *kilonova.UserBrief, pb *kilonova.Problem) []*kilonova.Contest {
			contests, err := base.ProblemRunningContests(ctx, pb.ID)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get running contests", slog.Any("err", err))
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
		"formatScore": func(pb *kilonova.Problem, score *decimal.Decimal) template.HTML {
			if score == nil || score.IsNegative() {
				return "-"
			}
			if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
				if score.Equal(decimal.NewFromInt(100)) {
					return `<i class="fas fa-fw fa-check"></i>`
				}
				return `<i class="fas fa-fw fa-xmark"></i>`
			}
			return template.HTML(removeTrailingZeros(score.StringFixed(pb.ScorePrecision)))
		},
		"subScore": func(pb *kilonova.Problem, user *kilonova.UserBrief) template.HTML {
			if user == nil {
				return ""
			}
			score := base.MaxScore(ctx, user.ID, pb.ID)
			if score.IsNegative() {
				return "-"
			}
			if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
				if score.Equal(decimal.NewFromInt(100)) {
					return `<i class="fas fa-fw fa-check"></i>`
				}
				return `<i class="fas fa-fw fa-xmark"></i>`
			}
			return template.HTML(removeTrailingZeros(score.StringFixed(pb.ScorePrecision)) + "p")
		},
		"actualMaxScore": func(pb *kilonova.Problem, user *kilonova.UserBrief) decimal.Decimal {
			return base.MaxScore(ctx, user.ID, pb.ID)
		},
		"spbMaxScore": func(pb *kilonova.ScoredProblem, summaryDisplay bool) template.HTML {
			if pb.ScoreUserID == nil {
				return ""
			}
			if pb.MaxScore == nil || pb.MaxScore.IsNegative() {
				return "-"
			}
			if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
				if pb.MaxScore.Equal(decimal.NewFromInt(100)) {
					return `<i class="fas fa-fw fa-check"></i>`
				}
				return `<i class="fas fa-fw fa-xmark"></i>`
			}
			val := removeTrailingZeros(pb.MaxScore.StringFixed(pb.ScorePrecision))
			if summaryDisplay {
				// Add the unit at the end
				val += "p"
			}
			return template.HTML(val)
		},
		"checklistMaxScore": func(pb *kilonova.ScoredProblem) string {
			if pb.ScoreUserID == nil {
				return "-1"
			}
			if pb.MaxScore == nil || pb.MaxScore.IsNegative() {
				return "-1"
			}
			return removeTrailingZeros(pb.MaxScore.StringFixed(pb.ScorePrecision))
		},
		"computeChecklistSpan": computeChecklistSpan,
		"scoreStep": func(pb *kilonova.Problem) string {
			return decimal.NewFromInt(1).Shift(-pb.ScorePrecision).String()
		},
		"contestAnnouncements": func(c *kilonova.Contest) []*kilonova.ContestAnnouncement {
			announcements, err := base.ContestAnnouncements(ctx, c.ID)
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
			questions, err := base.ContestQuestions(ctx, c.ID)
			if err != nil {
				return []*kilonova.ContestQuestion{}
			}
			return questions
		},
		"listProblems": func(user *kilonova.UserBrief, list *kilonova.ProblemList) []*kilonova.ScoredProblem {
			pbs, err := base.ProblemListProblems(ctx, list.List, user)
			if err != nil {
				return nil
			}
			return pbs
		},
		"pbParentPblists": func(problem *kilonova.Problem) []*kilonova.ProblemList {
			slog.ErrorContext(ctx, "Uninitialized `pbParentPblists`")
			return nil
		},
		"pblistParent": func(pblist *kilonova.ProblemList) []*kilonova.ProblemList {
			lists, err := base.PblistParentLists(ctx, pblist.ID)
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
				slog.ErrorContext(ctx, "Unknown renderMarkdown type")
				return "[Fatal error rendering markdown]"
			}
			val, err := base.RenderMarkdown(bd, nil)
			if err != nil {
				slog.WarnContext(ctx, "Error rendering markdown", slog.Any("err", err))
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
		"unescapeHTML": html.UnescapeString,
		"escapeHTML":   html.EscapeString,
		"serverTime": func() string {
			return time.Now().Format(time.RFC3339Nano)
		},
		"serverTimeFooter": func() string {
			return time.Now().Format("15:04:05")
		},
		"syntaxHighlight": func(code []byte, lang string) (string, error) {
			fmt := chtml.New(chtml.WithClasses(true), chtml.TabWidth(4))
			if lang == "pascal" {
				lang = "pas"
			}
			if lang == "nodejs" {
				lang = "js"
			}
			lm := lexers.Get(strings.TrimFunc(lang, unicode.IsDigit))
			if lm == nil {
				lm = lexers.Fallback
			}
			lm = chroma.Coalesce(lm)
			var buf bytes.Buffer
			it, err := lm.Tokenise(nil, string(code))
			if err != nil {
				return "", err
			}
			if err := fmt.Format(&buf, styles.Get("github"), it); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		"genProblemsParams": func(pbs []*kilonova.ScoredProblem, showPublished bool) *ProblemListingParams {
			return &ProblemListingParams{pbs, true, showPublished, -1, -1}
		},
		"genListProblemsParams": func(pbs []*kilonova.ScoredProblem, showPublished bool, listID int) *ProblemListingParams {
			return &ProblemListingParams{pbs, true, showPublished, -1, listID}
		},
		"genContestProblemsParams": func(pbs []*kilonova.ScoredProblem, contest *kilonova.Contest) *ProblemListingParams {
			slog.ErrorContext(ctx, "Uninitialized `genContestProblemsParams`")
			return nil
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
			user, err := base.UserBrief(ctx, uid)
			if err != nil {
				return nil
			}
			return user
		},
		"problemEditors": func(problem *kilonova.Problem) []*kilonova.UserBrief {
			users, err := base.ProblemEditors(ctx, problem.ID)
			if err != nil {
				return nil
			}
			return users
		},
		"problemViewers": func(problem *kilonova.Problem) []*kilonova.UserBrief {
			users, err := base.ProblemViewers(ctx, problem.ID)
			if err != nil {
				return nil
			}
			return users
		},
		"problemTests": func(problem *kilonova.Problem) []*kilonova.Test {
			tests, err := base.Tests(ctx, problem.ID)
			if err != nil {
				return nil
			}
			return tests
		},
		"problemSubtasks": func(problem *kilonova.Problem) []*kilonova.SubTask {
			sts, err := base.SubTasks(ctx, problem.ID)
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
		"humanizeBytes": func(cnt int64) string {
			return humanize.IBytes(uint64(cnt))
		},
		"titleName": func(s string) string {
			return cases.Title(language.English).String(s)
		},
		"hashedName": fsys.HashName,
		"version":    func() string { return kilonova.Version },
		"debug":      func() bool { return config.Common.Debug },

		"formatCanonical": func(path string) string {
			return hostURL.JoinPath(path).String()
		},

		"tagsByType": func(g string) []*kilonova.Tag {
			tags, err := base.TagsByType(ctx, kilonova.TagType(g))
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get tags", slog.String("tag_type", g))
				return nil
			}
			return tags
		},
		//NOTE: problemTags does no checking whether the tags are visible. It must be used only after making sure that the problem is Fully Visible.
		"problemTags": func(pb *kilonova.Problem) []*kilonova.Tag {
			if pb == nil {
				return nil
			}
			tags, err := base.ProblemTags(ctx, pb.ID)
			if err != nil {
				slog.WarnContext(ctx, "Couldn't get problem tags", slog.Any("err", err))
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
				slog.WarnContext(ctx, "Flag is not int", slog.String("name", name))
			}
			return val
		},
		"boolFlag": func(name string) bool {
			val, ok := config.GetFlagVal[bool](name)
			if !ok {
				slog.WarnContext(ctx, "Flag is not bool", slog.String("name", name))
			}
			return val
		},
		"stringFlag": func(name string) string {
			val, ok := config.GetFlagVal[string](name)
			if !ok {
				slog.WarnContext(ctx, "Flag is not string", slog.String("name", name))
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

		"sessionDevices": func(sid string) []*sudoapi.SessionDevice {
			devices, err := base.SessionDevices(ctx, sid)
			if err != nil {
				slog.WarnContext(ctx, "Could not get session devices", slog.Any("err", err))
				return nil
			}
			return devices
		},
		"ipData": func(ip *netip.Addr) *maxmind.Data {
			if ip == nil {
				return nil
			}
			data, err := maxmind.IPData(*ip)
			if err == nil && data != nil {
				return data
			}
			return nil
		},

		// for problem edit page
		"maxMemMB": func() float64 { return float64(config.Common.TestMaxMemKB) / 1024.0 },

		"intList": func(ids []int) string {
			if len(ids) == 0 {
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
		"stringList": func(vals []string) string {
			return strings.Join(vals, ", ")
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
		"formatStmtVariant": func(fmt *kilonova.StatementVariant) string {
			slog.ErrorContext(ctx, "Uninitialized `formatStmtVariant`")
			return ""
		},
		"mdVariantCount": func(sv []*kilonova.StatementVariant) int {
			var cnt int
			for _, v := range sv {
				if v.Format == "md" {
					cnt++
				}
			}
			return cnt
		},
		"computeDonationSum": func(d *kilonova.Donation) float64 {
			endTime := time.Now()
			if d.CancelledAt != nil {
				endTime = *d.CancelledAt
			}
			numMonths := (endTime.Year()-d.DonatedAt.Year())*12 + int(endTime.Month()-d.DonatedAt.Month()) + 1
			if endTime.Day() < d.DonatedAt.Day() {
				numMonths--
			}
			switch d.Type {
			case kilonova.DonationTypeMonthly:
				return d.Amount * float64(numMonths)
			case kilonova.DonationTypeYearly:
				numMonths /= 12
				return d.Amount * float64(numMonths)
			default:
				return d.Amount
			}
		},
		"forceSubCode": func(sub *kilonova.FullSubmission) []byte {
			code, err := base.SubmissionCode(ctx, &sub.Submission, sub.Problem, nil, false)
			if err != nil {
				slog.WarnContext(ctx, "Could not get submission code", slog.Any("err", err))
				code = nil
			}
			return code
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

		"getCaptchaID": base.NewCaptchaID,
		"pLanguages": func() map[string]string {
			slog.ErrorContext(ctx, "Uninitialized `pLanguages`")
			return make(map[string]string)
		},
		"mustSolveCaptcha": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `mustSolveCaptcha`")
			return false
		},
		"getText": func(key string, vals ...any) string {
			slog.ErrorContext(ctx, "Uninitialized `getText`")
			return "FATAL ERR"
		},
		"olderSubmissions": func(user *kilonova.UserBrief, problem *kilonova.Problem, contest *kilonova.Contest, limit int) *OlderSubmissionsParams {
			slog.ErrorContext(ctx, "Uninitialized `olderSubmissions`")
			return nil
		},
		"reqPath": func() string {
			slog.ErrorContext(ctx, "Uninitialized `reqPath`")
			return "/"
		},
		"htmxRequest": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `htmxRequest`")
			return false
		},
		"language": func() string {
			slog.ErrorContext(ctx, "Uninitialized `language`")
			return "en"
		},
		"isDarkMode": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `isDarkMode`")
			return true
		},
		"authed": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `authed`")
			return false
		},
		"prepareDuration": func() time.Duration {
			slog.ErrorContext(ctx, "Uninitialized `prepareDuration`")
			return time.Duration(0)
		},
		"renderDuration": func() time.Duration {
			slog.ErrorContext(ctx, "Uninitialized `renderDuration`")
			return time.Duration(0)
		},
		"queryCount": func() int64 {
			slog.ErrorContext(ctx, "Uninitialized `queryCount`")
			return -1
		},
		"contentUser": func() *kilonova.UserBrief {
			slog.ErrorContext(ctx, "Uninitialized `contentUser`")
			return nil
		},
		"fullAuthedUser": func() *kilonova.UserFull {
			slog.ErrorContext(ctx, "Uninitialized `fullAuthedUser`")
			return nil
		},
		"authedUser": func() *kilonova.UserBrief {
			slog.ErrorContext(ctx, "Uninitialized `authedUser`")
			return nil
		},
		"isAdmin": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `isAdmin`")
			return false
		},
		"discordIdentity": func(user *kilonova.UserFull) *discordgo.User {
			slog.ErrorContext(ctx, "Uninitialized `discordIdentity`")
			return nil
		},
		"isContestEditor": func(c *kilonova.Contest) bool {
			slog.ErrorContext(ctx, "Uninitialized `isContestEditor`")
			return false
		},
		"contestLeaderboardVisible": func(c *kilonova.Contest) bool {
			slog.ErrorContext(ctx, "Uninitialized `contestLeaderboardVisible`")
			return false
		},
		"contestProblemsVisible": func(c *kilonova.Contest) bool {
			slog.ErrorContext(ctx, "Uninitialized `contestProblemsVisible`")
			return false
		},
		"contestQuestions": func(c *kilonova.Contest) []*kilonova.ContestQuestion {
			slog.ErrorContext(ctx, "Uninitialized `contestQuestions`")
			return nil
		},
		"currentProblem": func() *kilonova.Problem {
			slog.ErrorContext(ctx, "Uninitialized `currentProblem`")
			return nil
		},
		"canViewAllSubs": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `canViewAllSubs`")
			return false
		},
		"contestRegistration": func(c *kilonova.Contest) *kilonova.ContestRegistration {
			slog.ErrorContext(ctx, "Uninitialized `contestRegistration`")
			return nil
		},
		"problemFullyVisible": func() bool {
			slog.ErrorContext(ctx, "Uninitialized `problemFullyVisible`")
			return false
		},
		"numSolvedPblist": func(listID int) int {
			slog.ErrorContext(ctx, "Uninitialized `numSolvedPblist`")
			return -1
		},
		"subCode": func(sub *kilonova.FullSubmission) []byte {
			slog.ErrorContext(ctx, "Uninitialized `subCode`")
			return nil
		},
	}
	return &Web{funcs, base}
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
	// Note that all files created in static/misc should have a hash generated automatically by the bundler
	if orgHash != "" || strings.HasPrefix(trueName, "static/misc/") {
		w.Header().Set("Cache-Control", `public, max-age=31536000, immutable`)
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

func removeTrailingZeros(score string) string {
	if !strings.ContainsRune(score, '.') {
		return score
	}
	return strings.TrimSuffix(strings.TrimRight(score, "0"), ".")
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" || r.FormValue("force_htmx") == "true"
}
