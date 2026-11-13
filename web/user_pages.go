package web

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/submissions"
	"github.com/KiloProjects/kilonova/web/ui/userviews"
	g "maragu.dev/gomponents"
)

// renderMarkdownHTML renders md for embedding with g.Raw, falling back to the same
// placeholder the renderMarkdown template func uses.
func (rt *Web) renderMarkdownHTML(r *http.Request, md string) string {
	val, err := rt.base.RenderMarkdown([]byte(md), nil)
	if err != nil {
		slog.WarnContext(r.Context(), "Error rendering markdown", slog.Any("err", err))
		return "[Error rendering markdown]"
	}
	return string(val)
}

// contentUserByName loads the user named in the {user} path value. When it's missing it
// redirects a former username to its current holder (suffix is appended to /profile/<name>)
// or writes a 404, and returns nil.
func (rt *Web) contentUserByName(w http.ResponseWriter, r *http.Request, suffix string) *kilonova.UserFull {
	username := strings.TrimSpace(r.PathValue("user"))
	userFull, err := rt.base.UserFullByName(r.Context(), username)
	if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
		slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
		rt.statusPage(w, r, 500, "")
		return nil
	}
	if userFull == nil {
		user2, err := rt.base.HistoricalUsernameHolder(r.Context(), username)
		if err != nil || user2 == nil {
			rt.statusPage(w, r, 404, "")
		} else {
			http.Redirect(w, r, "/profile/"+user2.Name+suffix, http.StatusMovedPermanently)
		}
		return nil
	}
	return userFull
}

func (rt *Web) profilePage(w http.ResponseWriter, r *http.Request, userFull *kilonova.UserFull) {
	if !(flags.ViewOtherProfiles.Get(r.Context()) || user.UserBrief(r).IsAdmin() || (user.UserBrief(r) != nil && user.UserBrief(r).ID == userFull.ID)) {
		rt.statusPage(w, r, 400, "You are not allowed to view other profiles!")
		return
	}

	solvedPbs, solvedCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
		LookingUser: user.UserBrief(r), Look: true, LookFullyVisible: true,
		SolvedBy: &userFull.ID,

		Limit: 50,
	}, userFull.Brief(), user.UserBrief(r))
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get solved problems", slog.Any("err", err))
		solvedPbs = []*sudoapi.FullProblem{}
	}

	attemptedPbs, attemptedCnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
		LookingUser: user.UserBrief(r), Look: true, LookFullyVisible: true,
		AttemptedBy: &userFull.ID,

		Limit: 50,
	}, userFull.Brief(), user.UserBrief(r))
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get attempted problems", slog.Any("err", err))
		attemptedPbs = []*sudoapi.FullProblem{}
	}

	changeHistory, err := rt.base.UsernameChangeHistory(r.Context(), userFull.ID)
	if err != nil {
		changeHistory = []*kilonova.UsernameChange{}
	}

	rt.runLayout(w, r, &LayoutParams{
		Title: kilonova.GetText(util.Language(r), "profile") + " " + userFull.Name,
		Content: userviews.Profile(r.Context(), userviews.ProfileParams{
			User:              userFull,
			BioHTML:           rt.renderMarkdownHTML(r, userFull.Bio),
			SolvedProblems:    solvedPbs,
			SolvedCount:       solvedCnt,
			AttemptedProblems: attemptedPbs,
			AttemptedCount:    attemptedCnt,
			ChangeHistory:     changeHistory,
		}),
	})
}

func (rt *Web) selfProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.profilePage(w, r, user.UserFull(r))
	}
}

func (rt *Web) profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFull := rt.contentUserByName(w, r, ""); userFull != nil {
			rt.profilePage(w, r, userFull)
		}
	}
}

func (rt *Web) userSubmissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userFull := rt.contentUserByName(w, r, "/submissions")
		if userFull == nil {
			return
		}

		query := parseSubsViewerQuery(r)
		query.UserID = &userFull.ID
		filter := buildSubsFilter(query)
		subs, err := rt.base.Submissions(r.Context(), filter, true, user.UserBrief(r))
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch submissions")
			return
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.user_past_submissions", userFull.Name),
			Head:  ui.NoRobots(),
			Content: g.Group{
				userviews.Topbar(r.Context(), userFull, "submissions"),
				submissions.SubsViewer(r.Context(), submissions.SubsViewerParams{
					Submissions: subs,
					Query:       query,
					NumPages:    (subs.Count + 49) / 50,
					Overwrites:  submissions.SubsViewerOverwrites{UserID: true},
					Languages:   rt.base.EnabledLanguages(),
					IsAdmin:     user.UserBrief(r).IsAdmin(),
				}),
			},
		})
	}
}

func (rt *Web) userSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := user.UserFull(r)
		lang := util.Language(r)
		avatarHelp := kilonova.GetText(lang, "changeAvatarNoDiscord")
		if dUser, err := rt.base.GetDiscordIdentity(r.Context(), u.ID); err == nil && dUser != nil {
			avatarHelp = kilonova.GetText(lang, "changeAvatarWithDiscord", dUser.String())
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(lang, "settings"),
			Content: userviews.Settings(r.Context(), userviews.SettingsParams{
				User:           u,
				AvatarHelpHTML: rt.renderMarkdownHTML(r, avatarHelp),
			}),
		})
	}
}

func (rt *Web) linkStatusPage(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull) {
	dUser, err := rt.base.GetDiscordIdentity(r.Context(), user.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "Could not get Discord identity", slog.Any("user", user), slog.Any("err", err))
		dUser = nil
	}
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "discordLink") + ": " + user.Name,
		Content: userviews.DiscordLink(r.Context(), user, dUser),
	})
}

func (rt *Web) selfLinkStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.linkStatusPage(w, r, user.UserFull(r))
	}
}

func (rt *Web) linkStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userFull, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("user")))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
			rt.statusPage(w, r, 500, "")
			return
		}
		if userFull == nil {
			rt.statusPage(w, r, 404, "")
			return
		}
		// Only admins and that specific user can view their link status
		if !(user.UserBrief(r).IsAdmin() || user.UserBrief(r).ID == userFull.ID) {
			rt.statusPage(w, r, 403, "")
			return
		}

		rt.linkStatusPage(w, r, userFull)
	}
}
