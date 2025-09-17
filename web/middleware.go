package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"golang.org/x/text/language"
)

// Language stuff
var serverLangs = []language.Tag{
	language.English,
	language.Romanian,
}
var langMatcher = language.NewMatcher(serverLangs)

func trimNonDigits(s string) string {
	return strings.TrimRightFunc(s, func(r rune) bool { return !unicode.IsDigit(r) })
}

// ValidateProblemID makes sure the problem ID is a valid uint
func (rt *Web) ValidateProblemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		problemID, err := strconv.Atoi(trimNonDigits(r.PathValue("pbid")))
		if err != nil {
			rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		var pb *kilonova.Problem
		if util.Contest(r) == nil {
			problem, err := rt.base.Problem(r.Context(), problemID)
			if err != nil || problem == nil {
				rt.statusPage(w, r, 404, "Problema nu a fost găsită")
				return
			}
			pb = problem
		} else {
			problem, err := rt.base.ContestProblem(r.Context(), util.Contest(r), util.UserBrief(r), problemID)
			if err != nil || problem == nil {
				rt.statusPage(w, r, 404, "Problema nu a fost găsită sau nu aparține concursului")
				return
			}
			pb = problem
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemKey, pb)))
	})
}

// ValidateBlogPostSlug makes sure the blog post slug is a valid one
func (rt *Web) ValidateBlogPostSlug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postSlug := strings.TrimRightFunc(r.PathValue("postslug"), func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_')
		})
		post, err := rt.base.BlogPostBySlug(r.Context(), postSlug)
		if err != nil {
			rt.statusPage(w, r, 404, "Articolul nu a fost găsit")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.BlogPostKey, post)))
	})
}

// ValidateListID makes sure the list ID is a valid uint
func (rt *Web) ValidateListID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listID, err := strconv.Atoi(trimNonDigits(r.PathValue("id")))
		if err != nil {
			rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		list, err := rt.base.ProblemList(r.Context(), listID)
		if err != nil {
			rt.statusPage(w, r, 404, "Lista nu a fost găsită")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ProblemListKey, list)))
	})
}

// ValidateExternalResourceID makes sure the resource ID is a valid uint
func (rt *Web) ValidateExternalResourceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resourceID, err := strconv.Atoi(trimNonDigits(r.PathValue("resID")))
		if err != nil {
			rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		resources, err := rt.base.ExternalResources(r.Context(), kilonova.ExternalResourceFilter{
			ID: &resourceID,

			Look:        true,
			LookingUser: util.UserBrief(r),
		})
		if err != nil || len(resources) != 1 {
			rt.statusPage(w, r, 404, "Eticheta nu există sau nu ai acces la aceasta")
			return
		}

		next.ServeHTTP(w, r.WithContext(
			context.WithValue(r.Context(), util.ExternalResourceKey, resources[0]),
		))
	})
}

// ValidateTagID makes sure the tag ID is a valid uint
func (rt *Web) ValidateTagID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tagID, err := strconv.Atoi(trimNonDigits(r.PathValue("tagid")))
		if err != nil {
			// Try to also match names
			tag, err := rt.base.TagByName(r.Context(), r.PathValue("tagid"))
			if err != nil {
				rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TagKey, tag)))
			return
		}
		tag, err := rt.base.TagByID(r.Context(), tagID)
		if err != nil {
			rt.statusPage(w, r, 404, "Eticheta nu a fost găsită")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.TagKey, tag)))
	})
}

// canViewTags makes sure the user can view tags
func (rt *Web) canViewTags(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(flags.AnyoneTags.Value() || util.UserBrief(r).IsProposer()) {
			rt.authedStatusPage(w, r, 400, "Nu ai voie să accesezi etichetele!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateProblemVisible checks if the problem from context is visible from the logged in user
func (rt *Web) ValidateProblemVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsProblemVisible(util.UserBrief(r), util.Problem(r)) {
			rt.authedStatusPage(w, r, 403, "Nu ai voie să accesezi problema!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateProblemFullyVisible checks if the problem from context is FULLY visible from the logged in user
func (rt *Web) ValidateProblemFullyVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsProblemFullyVisible(util.UserBrief(r), util.Problem(r)) {
			rt.authedStatusPage(w, r, 403, "Nu ai voie să accesezi această pagină!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateBlogPostVisible checks if the post from context is visible from the logged in user
func (rt *Web) ValidateBlogPostVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsBlogPostVisible(util.UserBrief(r), util.BlogPost(r)) {
			rt.authedStatusPage(w, r, 403, "Nu ai voie să accesezi articolul!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateContestID makes sure the contest ID is a valid uint
func (rt *Web) ValidateContestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contestID, err := strconv.Atoi(trimNonDigits(r.PathValue("contestID")))
		if err != nil {
			rt.statusPage(w, r, http.StatusBadRequest, "ID invalid")
			return
		}
		contest, err := rt.base.Contest(r.Context(), contestID)
		if err != nil {
			rt.statusPage(w, r, 404, "Concursul nu a fost găsit")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ContestKey, contest)))
	})
}

// ValidateContestVisible checks if the problem from context is visible from the logged in user
func (rt *Web) ValidateContestVisible(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsContestVisible(util.UserBrief(r), util.Contest(r)) {
			rt.authedStatusPage(w, r, 403, "Nu ai voie să accesezi concursul!")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ValidateSubmissionID puts the ID and the Submission in the router context
func (rt *Web) ValidateSubmissionID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subID, err := strconv.Atoi(trimNonDigits(r.PathValue("subID")))
		if err != nil {
			rt.statusPage(w, r, 400, "ID submisie invalid")
			return
		}
		sub, err := rt.base.Submission(r.Context(), subID, util.UserBrief(r))
		if err != nil {
			if kilonova.ErrorCode(err) != 404 {
				slog.WarnContext(r.Context(), "Could not get submission", slog.Any("err", err), slog.Int("subID", subID))
			}
			rt.statusPage(w, r, 400, "Submisia nu există sau nu poate fi vizualizată")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.SubKey, sub)))
	})
}

// ValidatePasteID puts the ID and the Paste in the router context
func (rt *Web) ValidatePasteID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paste, err := rt.base.SubmissionPaste(r.Context(), r.PathValue("pasteID"))
		if err != nil {
			rt.statusPage(w, r, 400, "Paste-ul nu există")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.PasteKey, paste)))
	})
}

// ValidateAttachmentID makes sure the attachment ID is a valid uint
func (rt *Web) ValidateAttachmentID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attid, err := strconv.Atoi(trimNonDigits(r.PathValue("aid")))
		if err != nil {
			http.Error(w, "ID invalid", 400)
			return
		}
		att, err := rt.base.Attachment(r.Context(), attid)
		if err != nil {
			http.Error(w, "Atașamentul nu a fost găsit", 404)
			return
		}
		if att.Private && !rt.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
			rt.authedStatusPage(w, r, 403, "Nu ai voie să accesezi acest atașament")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.AttachmentKey, att)))
	})
}

func (rt *Web) mustBeAuthed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsAuthed() {
			http.Redirect(w, r, "/login?back="+url.PathEscape(r.URL.Path), http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// maybe the user is just not authed, let's give them another chance before actually returning an error
func (rt *Web) authedStatusPage(w http.ResponseWriter, r *http.Request, code int, message string) {
	if !util.UserBrief(r).IsAuthed() {
		http.Redirect(w, r, "/login?back="+url.PathEscape(r.URL.Path), http.StatusTemporaryRedirect)
		return
	}
	rt.statusPage(w, r, code, message)
}

func (rt *Web) mustBeProposer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsProposer() {
			rt.statusPage(w, r, 401, "Trebuie să fii propunător")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeAdmin(next http.Handler) http.Handler {
	return rt.mustBeAuthed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.UserBrief(r).IsAdmin() {
			rt.statusPage(w, r, 401, "Trebuie să fii admin")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (rt *Web) mustBeVisitor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if util.UserBrief(r).IsAuthed() {
			http.Redirect(w, r, "/", http.StatusFound)
			//rt.statusPage(w, r, 401, "Trebuie să fii delogat")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeProblemEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) {
			rt.statusPage(w, r, 401, "Trebuie să fii un editor al problemei")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBePostEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.base.IsBlogPostEditor(util.UserBrief(r), util.BlogPost(r)) {
			rt.statusPage(w, r, 401, "Trebuie să fii editor al articolului")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) mustBeContestEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !util.Contest(r).IsEditor(util.UserBrief(r)) {
			rt.statusPage(w, r, 401, "Trebuie să fii un administrator al concursului")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Web) checkFlag(flag config.Flag[bool]) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !flag.Value() {
				rt.statusPage(w, r, http.StatusPreconditionFailed, "Page has been disabled on this instance.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rt *Web) initSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), MiddlewareStartKey, time.Now())
		ctx = sudoapi.InitQueryCounter(ctx)
		user, err := rt.base.SessionUser(ctx, rt.base.GetSessCookie(r), r)
		if err != nil || user == nil {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		trace.SpanFromContext(ctx).SetAttributes(attribute.Int("user.id", user.ID), attribute.String("user.name", user.Name))
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, util.AuthedUserKey, user)))
	})
}

func (rt *Web) initLanguage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userLang := ""
		if util.UserFull(r) != nil {
			userLang = util.UserFull(r).PreferredLanguage
		}
		// get language
		lang, _ := r.Cookie("kn-lang")
		cookieLang := ""
		if lang != nil {
			cookieLang = lang.Value
		}

		accept := r.Header.Get("Accept-Language")
		tag, _ := language.MatchStrings(langMatcher, cookieLang, userLang, accept)
		language, _ := tag.Base()

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.LangKey, language.String())))
	})
}

func (rt *Web) initTheme(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userTheme kilonova.PreferredTheme = kilonova.PreferredThemeNone
		if util.UserFull(r) != nil {
			userTheme = util.UserFull(r).PreferredTheme
		}

		// get cookie that overrides
		cTheme, _ := r.Cookie("kn-theme")
		if cTheme != nil {
			if cTheme.Value == "light" || cTheme.Value == "dark" {
				userTheme = kilonova.PreferredTheme(cTheme.Value)
			}
		}

		if userTheme == kilonova.PreferredThemeNone { // Default to dark mode
			userTheme = kilonova.PreferredThemeDark
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.ThemeKey, userTheme)))
	})
}
