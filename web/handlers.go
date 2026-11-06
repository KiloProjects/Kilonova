package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"math/rand/v2"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/web/ui/adminviews"
	"github.com/KiloProjects/kilonova/web/ui/authviews"
	"github.com/KiloProjects/kilonova/web/ui/modals"
	"github.com/KiloProjects/kilonova/web/ui/submissions"
	"github.com/KiloProjects/kilonova/web/ui/utilviews"
	"github.com/shopspring/decimal"
	"github.com/skip2/go-qrcode"

	knlanguage "github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/layout"
	"github.com/KiloProjects/kilonova/web/ui/navbar"
	g "maragu.dev/gomponents"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/gorilla/schema"
)

type webContextType string

const (
	PblistCntCacheKey = webContextType("pblist_cache")
)

func (rt *Web) buildPblistCache(r *http.Request, listIDs []int) *http.Request {
	if user.UserBrief(r) == nil {
		return r
	}
	cache, err := rt.base.NumSolvedFromPblists(r.Context(), listIDs, user.UserBrief(r))
	if err == nil {
		return r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, cache))
	}
	if errors.Is(err, context.Canceled) {
		// Build mock cache to silence cache misses
		mockCache := make(map[int]int)
		for _, id := range listIDs {
			mockCache[id] = 0
		}
		return r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, mockCache))
	}
	slog.WarnContext(r.Context(), "Couldn't build problem list cache", slog.Any("err", err))
	return r
}

func (rt *Web) discordLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user.UserFull(r).DiscordID != nil {
			http.Redirect(w, r, "/profile/linked", http.StatusTemporaryRedirect)
			return
		}
		st, err := rt.base.DiscordAuthURL(r.Context(), user.UserBrief(r).ID)
		if err != nil {
			rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
			return
		}

		http.Redirect(w, r, st, http.StatusTemporaryRedirect)
	}
}

func (rt *Web) usersList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.FormValue("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			page = 1
		}
		name := r.FormValue("name")

		var fuzzyName *string
		if len(name) > 0 {
			fuzzyName = &name
		}

		users, err := rt.base.UsersBrief(r.Context(), kilonova.UserFilter{
			FuzzyName: fuzzyName,
			Limit:     50,
			Offset:    uint64(50 * (page - 1)),
		})
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch users")
			return
		}

		cnt, err := rt.base.CountUsers(r.Context(), kilonova.UserFilter{FuzzyName: fuzzyName})
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't count users")
			return
		}

		numPages := cnt / 50
		if cnt%50 > 0 {
			numPages++
		}

		var pagination g.Node
		if numPages > 1 {
			pagination = ui.Paginator(ui.PaginatorConfig{
				Page:       page,
				NumPages:   numPages,
				ShowArrows: true,
			})
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "users"),
			Content: adminviews.UsersList(r.Context(), adminviews.UsersListParams{
				Users:      users,
				Name:       name,
				Pagination: pagination,
			}),
		})
	}
}

func (rt *Web) submission(w http.ResponseWriter, r *http.Request) {
	sub := util.Submission(r)

	var olderSubs *modals.OlderSubmissionsParams = nil
	var err error
	if user.UserBrief(r) != nil {
		olderSubs, err = rt.getOlderSubmissions(r.Context(), user.UserBrief(r), user.UserBrief(r).ID, sub.Problem, nil)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get submissions", slog.Any("err", err))
		}
	}
	if olderSubs != nil {
		olderSubs.AutoReload = false
	}

	subFiles, err := rt.base.SubmissionFiles(r.Context(), &sub.Submission, sub.Problem, user.UserBrief(r), true)
	if err != nil {
		if kilonova.ErrorCode(err) != 400 {
			slog.WarnContext(r.Context(), "Could not get submission files", slog.Any("err", err))
		}
		subFiles = []*kilonova.SubmissionFile{}
	}

	rt.runLayout(w, r, &LayoutParams{
		Title: fmt.Sprintf("%s %d", kilonova.GetText(util.Language(r), "sub"), sub.ID),
		Head:  ui.NoRobots(),
		Content: submissions.Submission(r.Context(), submissions.SubmissionPageParams{
			Submission:    sub,
			ForceShowCode: r.URL.Query().Get("forceCode") == "1",
			LanguageFormatter: func(lang string) string {
				if lng := rt.base.AnyLanguage(lang); lng != nil {
					return lng.PrintableName()
				}
				return "Unknown"
			},
			OlderSubmissions: olderSubs,
			SubmissionFiles:  subFiles,
		}),
	})
}

func (rt *Web) downloadSubmission(w http.ResponseWriter, r *http.Request) {
	code, err := rt.base.SubmissionCode(r.Context(), &util.Submission(r).Submission, util.Submission(r).Problem, nil, false)
	if err != nil {
		rt.statusPage(w, r, 500, "Failed to retrieve submission code")
		return
	}
	if len(code) == 0 {
		rt.statusPage(w, r, 400, "Code is either unavailable or doesn't exist.")
		return
	}
	extension := knlanguage.FirstExtension(rt.base.AnyLanguage(util.Submission(r).Language))
	if extension == ".outputOnly" {
		extension = ".txt"
	}
	filename := fmt.Sprintf("%d-%s%s", util.Submission(r).ID, kilonova.MakeSlug(util.Submission(r).Problem.Name), extension)
	w.Header().Add("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	http.ServeContent(w, r, filename, util.Submission(r).CreatedAt, bytes.NewReader(code))
}

func (rt *Web) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	// Check submission permissions
	if !util.Submission(r).CanDelete(user.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't delete this submission!")
		return
	}

	if err := rt.base.DeleteSubmission(r.Context(), util.Submission(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not delete submission")
		return
	}

	w.Header().Set("HX-Redirect", "/submissions")
	http.Redirect(w, r, "/submissions", http.StatusNoContent)
}

func (rt *Web) reevaluateSubmission(w http.ResponseWriter, r *http.Request) {
	// Check submission permissions
	if !util.Submission(r).CanDelete(user.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't reevaluate this submission!")
		return
	}

	if err := rt.base.ResetSubmission(r.Context(), util.Submission(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not reevaluate submission")
		return
	}

	htmxSuccessToast(w, r, "Reset submission")
	w.Header().Set("HX-Refresh", "true")
	http.Redirect(w, r, "/submissions/"+strconv.Itoa(util.Submission(r).ID), http.StatusFound)
}

func (rt *Web) createPaste(w http.ResponseWriter, r *http.Request) {
	if !util.Submission(r).IsEditor(user.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't create a paste for this submission!")
		return
	}

	id, err := rt.base.CreatePaste(r.Context(), &util.Submission(r).Submission, user.UserBrief(r))
	if err != nil {
		rt.statusPage(w, r, 500, "Could not create paste")
		return
	}
	http.Redirect(w, r, "/pastes/"+id, http.StatusFound)
}

func (rt *Web) submissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.canViewAllSubs(user.UserBrief(r)) {
			rt.statusPage(w, r, 401, "Cannot view all submissions")
			return
		}
		query := parseSubsViewerQuery(r)
		filter := buildSubsFilter(query)
		subs, err := rt.base.Submissions(r.Context(), filter, true, user.UserBrief(r))
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch submissions")
			return
		}
		numPages := (subs.Count + 49) / 50
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "submissions"),
			Head:  ui.CanonicalURL("/submissions"),
			Content: submissions.SubsPage(r.Context(), submissions.SubsViewerParams{
				Submissions: subs,
				Query:       query,
				NumPages:    numPages,
				Languages:   rt.base.EnabledLanguages(),
				IsAdmin:     user.UserBrief(r) != nil && user.UserBrief(r).IsAdmin(),
			}),
		})
	}
}

// canViewAllSubs is just for the text in the navbar and the submissions page
// TODO: Restrict on the backend, as well.
func (rt *Web) canViewAllSubs(user *kilonova.UserBrief) bool {
	if flags.AllSubsPage.Value() {
		return true
	}
	return user.IsProposer()
}

func (rt *Web) paste(w http.ResponseWriter, r *http.Request) {
	sub, err := rt.base.FullSubmission(r.Context(), util.Paste(r).Submission.ID)
	if err != nil {
		rt.statusPage(w, r, 500, "N-am putut obține submisia aferentă")
		return
	}

	var olderSubs *modals.OlderSubmissionsParams
	if user.UserBrief(r) != nil {
		olderSubs, err = rt.getOlderSubmissions(r.Context(), user.UserBrief(r), user.UserBrief(r).ID, sub.Problem, nil)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get submissions", slog.Any("err", err))
		}
	}
	if olderSubs != nil {
		olderSubs.AutoReload = false
	}

	// Get submission files without checking the looking user
	subFiles, err := rt.base.SubmissionFiles(r.Context(), &sub.Submission, sub.Problem, nil, false)
	if err != nil {
		if kilonova.ErrorCode(err) != 400 {
			slog.WarnContext(r.Context(), "Could not get submission files", slog.Any("err", err))
		}
		subFiles = []*kilonova.SubmissionFile{}
	}

	rt.runLayout(w, r, &LayoutParams{
		Title: fmt.Sprintf("%s #%d", kilonova.GetText(util.Language(r), "sub"), sub.ID),
		Content: submissions.Paste(r.Context(),
			submissions.PastePageParams{
				Paste:         util.Paste(r),
				Submission:    sub,
				ForceShowCode: true,
				LanguageFormatter: func(lang string) string {
					if lng := rt.base.AnyLanguage(lang); lng != nil {
						return lng.PrintableName()
					}
					return "Unknown"
				},
				OlderSubmissions: olderSubs,
				SubmissionFiles:  subFiles,
			}),
	})
}

func (rt *Web) deletePaste(w http.ResponseWriter, r *http.Request) {
	// Check paste permissions
	if !util.Paste(r).IsEditor(user.UserBrief(r)) {
		rt.statusPage(w, r, 403, "You can't delete this paste!")
		return
	}

	if err := rt.base.DeletePaste(r.Context(), util.Paste(r).ID); err != nil {
		rt.statusPage(w, r, 500, "Could not delete submission")
		return
	}

	w.Header().Set("HX-Redirect", "/submissions")
	http.Redirect(w, r, "/submissions", http.StatusNoContent)
}

func (rt *Web) downloadPaste() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fullSub, err := rt.base.FullSubmission(r.Context(), util.Paste(r).Submission.ID)
		if err != nil {
			rt.statusPage(w, r, 500, "N-am putut obține submisia aferentă")
			return
		}
		code, err := rt.base.SubmissionCode(r.Context(), &fullSub.Submission, fullSub.Problem, nil, false)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get submission code", slog.Any("err", err))
			code = nil
		}

		extension := knlanguage.FirstExtension(rt.base.AnyLanguage(fullSub.Language))
		if extension == ".outputOnly" {
			extension = ".txt"
		}
		filename := fmt.Sprintf("%s-%s%s", util.Paste(r).ID, kilonova.MakeSlug(fullSub.Problem.Name), extension)
		w.Header().Add("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		http.ServeContent(w, r, filename, fullSub.CreatedAt, bytes.NewReader(code))
	}
}

func (rt *Web) randomProblem() http.HandlerFunc {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")

	type problemArgs struct {
		ListID int   `json:"list_id"`
		TagIDs []int `json:"tag_id"`

		// If nil, it's disregarded
		// If true, searches for unsolved problems
		// If false, searches for solved problems
		Unsolved *bool `json:"unsolved"`
		// User ID to base searches on.
		UnsolvedBy *int `json:"unsolved_by"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		var args problemArgs
		if err := decoder.Decode(&args, r.Form); err != nil {
			slog.WarnContext(r.Context(), "Could not decode HTTP form", slog.Any("err", err))
		}
		filter := kilonova.ProblemFilter{
			Look: true, LookingUser: user.UserBrief(r),
		}
		if args.ListID > 0 {
			filter.DeepListID = &args.ListID
		}
		if len(args.TagIDs) > 0 {
			filter.Tags = []*kilonova.TagGroup{
				{TagIDs: args.TagIDs},
			}
		}
		if args.Unsolved != nil {
			userID := -1
			if user.UserBrief(r) != nil {
				userID = user.UserBrief(r).ID
			}
			if args.UnsolvedBy != nil && *args.UnsolvedBy > 0 {
				userID = *args.UnsolvedBy
			}
			if userID > 0 {
				if *args.Unsolved {
					filter.UnsolvedBy = &userID
				} else {
					filter.SolvedBy = &userID
				}
			}
		}

		pbs, err := rt.base.Problems(r.Context(), filter)
		if err != nil {
			w.Header().Add("X-Problem-ID", "-1")
			rt.statusPage(w, r, 500, "Could not get random problem: "+err.Error())
			return
		}
		if len(pbs) == 0 {
			w.Header().Add("X-Problem-ID", "-1")
			rt.statusPage(w, r, 400, "Could not find a random problem matching the given criteria")
			return
		}

		pbid := strconv.Itoa(pbs[rand.N(len(pbs))].ID)
		w.Header().Add("X-Problem-ID", pbid)
		http.Redirect(w, r, "/problems/"+pbid, http.StatusFound)
	}
}

func (rt *Web) appropriateDescriptionVariant(r *http.Request, variants []*kilonova.StatementVariant) *kilonova.StatementVariant {
	prefLang, prefFormat, prefType := util.Language(r), "md", ""
	variant := strings.SplitN(r.FormValue("var"), "-", 3)
	if lang := r.FormValue("pref_lang"); len(lang) > 0 { // Backwards compatibility
		prefLang = lang
	}
	if len(variant) > 0 && len(variant[0]) > 0 {
		prefLang = variant[0]
	}
	if format := r.FormValue("pref_format"); len(format) > 0 { // Backwards compatibility
		prefFormat = format
	}
	if len(variant) > 1 && len(variant[1]) > 0 {
		prefFormat = variant[1]
	}
	if len(variant) > 2 && len(variant[2]) > 0 {
		prefType = variant[2]
	}

	if len(variants) == 0 {
		return &kilonova.StatementVariant{}
	}
	// Search for the ideal scenario
	for _, v := range variants {
		if v.Language == prefLang && v.Format == prefFormat && v.Type == prefType {
			return v
		}
	}
	// Then search if anything matches the format
	for _, v := range variants {
		if v.Format == prefFormat {
			return v
		}
	}
	// Then search if anything matches the language
	for _, v := range variants {
		if v.Language == prefLang {
			return v
		}
	}
	// If nothing was found, then just return the first available variant
	return variants[0]
}

// TODO: Properly figure out priorities
func (rt *Web) getFinalVariant(prefLang string, prefType string, variants []*kilonova.StatementVariant) *kilonova.StatementVariant {
	var finalVariant *kilonova.StatementVariant

	for _, vr := range variants {
		if vr.Format == "md" && vr.Language == prefLang {
			if len(prefType) > 0 && vr.Type == prefType {
				return vr
			}
			if finalVariant == nil || finalVariant.Type > vr.Type {
				finalVariant = vr
			}
		}
	}

	if finalVariant != nil {
		return finalVariant
	}

	for _, vr := range variants {
		if vr.Format == "md" {
			if len(prefType) > 0 && vr.Type == prefType {
				return vr
			}
			if finalVariant == nil || finalVariant.Type > vr.Type {
				finalVariant = vr
			}
		}
	}

	if finalVariant == nil {
		return &kilonova.StatementVariant{
			Language: prefLang,
			Format:   "md",
			Type:     "",
		}
	}
	return finalVariant
}

func (rt *Web) contestInviteQRCode(w http.ResponseWriter, r *http.Request) {
	inv, err := rt.base.ContestInvitation(r.Context(), r.PathValue("inviteID"))
	if err != nil || inv == nil {
		slog.WarnContext(r.Context(), "Couldn't get contest invitation", slog.Any("err", err))
		rt.statusPage(w, r, 404, "Invite not found")
		return
	}

	inviteURL := kilonova.HostURL().JoinPath("contests/invite", inv.ID).String()
	data, err := qrcode.Encode(inviteURL, qrcode.High, 256)
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't encode QR code", slog.Any("err", err))
		http.Error(w, "Couldn't generate QR code", 500)
		return
	}

	http.ServeContent(w, r, "qr.png", inv.CreatedAt, bytes.NewReader(data))
}

// func (rt *Web) updateExternalResource() http.HandlerFunc {
//	templ := rt.parse(nil, "externalResources/view.html")
//	return func(w http.ResponseWriter, r *http.Request) {
//		var author *kilonova.UserBrief
//		if res := util.ExternalResource(r); res.ProposedBy != nil {
//			if proposer, err := rt.base.UserBrief(r.Context(), *res.ProposedBy); err != nil {
//				slog.WarnContext(r.Context(), "Couldn't get external resource author", slog.Any("err", err))
//			} else {
//				author = proposer
//			}
//		}
//		rt.runTempl(w, r, templ, &ResourcesPageParams{
//			util.ExternalResource(r),
//			util.Problem(r),
//			author,
//		})
//	}
// }
// func (rt *Web) deleteExternalResource() http.HandlerFunc {
//	templ := rt.parse(nil, "externalResources/view.html")
//	return func(w http.ResponseWriter, r *http.Request) {
//		var author *kilonova.UserBrief
//		if res := util.ExternalResource(r); res.ProposedBy != nil {
//			if proposer, err := rt.base.UserBrief(r.Context(), *res.ProposedBy); err != nil {
//				slog.WarnContext(r.Context(), "Couldn't get external resource author", slog.Any("err", err))
//			} else {
//				author = proposer
//			}
//		}
//		rt.runTempl(w, r, templ, &ResourcesPageParams{
//			util.ExternalResource(r),
//			util.Problem(r),
//			author,
//		})
//	}
// }

func (rt *Web) serveGravatar(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull, size int) {
	// Read from cache
	rd, lastmod, valid, err := rt.base.GetGravatar(r.Context(), user.Email, size, time.Now().Add(-12*time.Hour))
	if !valid || err != nil {
		slog.WarnContext(r.Context(), "BaseAPI GetGravatar is not valid or returned error", slog.Bool("valid", valid), slog.Any("err", err))
		http.Error(w, "", 500)
		return
	}
	defer rd.Close()
	w.Header().Add("ETag", fmt.Sprintf("\"kn-%s-%d\"", user.Name, lastmod.Unix()))
	// Cache for 1 day
	w.Header().Add("Cache-Control", "public, max-age=86400, immutable")

	http.ServeContent(w, r, "gravatar.png", lastmod, rd)
}

func (rt *Web) avatarUserSize(w http.ResponseWriter, r *http.Request) (*kilonova.UserFull, int, bool) {
	userFull, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("user")))
	if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
		slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
		http.Error(w, "could not get user", 500)
		return nil, -1, false
	}
	if userFull == nil {
		http.Error(w, "user not found", 500)
		return nil, -1, false
	}

	size, err := strconv.Atoi(r.FormValue("s"))
	if err != nil || size == 0 {
		size = 128
	}
	return userFull, size, true
}

func (rt *Web) gravatar(w http.ResponseWriter, r *http.Request) {
	userFull, size, ok := rt.avatarUserSize(w, r)
	if !ok {
		return
	}

	rt.serveGravatar(w, r, userFull, size)
}

func (rt *Web) serveDiscordAvatar(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull, size int) {
	// Read from cache
	rd, lastmod, valid, err := rt.base.GetDiscordAvatar(r.Context(), user, size, time.Now().Add(-12*time.Hour))
	if !valid || err != nil {
		// slog.WarnContext(r.Context(), "BaseAPI GetDiscordAvatar is not valid or returned error", slog.Bool("valid", valid), slog.Any("err", err))
		http.Error(w, "discord not properly linked", 400)
		return
	}
	defer rd.Close()
	w.Header().Add("ETag", fmt.Sprintf("\"kn-discord-%s-%d\"", user.Name, lastmod.Unix()))
	// Cache for 1 day
	w.Header().Add("Cache-Control", "public, max-age=86400, immutable")

	http.ServeContent(w, r, "discordAvatar.png", lastmod, rd)
}

func (rt *Web) discordAvatar(w http.ResponseWriter, r *http.Request) {
	userFull, size, ok := rt.avatarUserSize(w, r)
	if !ok {
		return
	}

	rt.serveDiscordAvatar(w, r, userFull, size)
}

func (rt *Web) profilePicture(w http.ResponseWriter, r *http.Request) {
	userFull, size, ok := rt.avatarUserSize(w, r)
	if !ok {
		return
	}

	if userFull.AvatarType == "discord" && userFull.DiscordID != nil {
		rt.serveDiscordAvatar(w, r, userFull, size)
	} else {
		rt.serveGravatar(w, r, userFull, size)
	}
}

func (rt *Web) redirectDiscord(w http.ResponseWriter, r *http.Request) {
	id := flags.DiscordInviteID.Value()
	if id == "" {
		rt.statusPage(w, r, 500, "Discord was not configured on this instance")
		return
	}

	http.Redirect(w, r, "https://discord.gg/"+id, http.StatusFound)
}

func (rt *Web) checkLockout() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flags.ForceLogin.Value() && !user.UserBrief(r).IsAuthed() {
				http.Redirect(w, r, "/login?back="+url.PathEscape(r.URL.Path), http.StatusTemporaryRedirect)
				return
			}

			if user.UserFull(r) != nil && user.UserFull(r).NameChangeForced {
				rt.runLayout(w, r, &LayoutParams{
					Title:   kilonova.GetText(util.Language(r), "usernameChangeForced"),
					Content: utilviews.UserLockout(r.Context()),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rt *Web) confirmLogout(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "auth.logout"),
		Head:    ui.NoRobots(),
		Content: authviews.ConfirmLogout(r.Context(), r.FormValue("back")),
	})
}

func (rt *Web) logout(w http.ResponseWriter, r *http.Request) {
	emptyCookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, emptyCookie)

	c, err := r.Cookie("kn-sessionid")
	if err != nil {
		return
	}
	rt.base.RemoveSession(r.Context(), c.Value)

	http.Redirect(w, r, kilonova.HostURL().JoinPath(r.FormValue("back")).String()+"?logout=1", http.StatusSeeOther)
}

type LayoutParams struct {
	Title       string
	Description string
	Head        g.Node
	Content     g.Node
}

func (rt *Web) runLayout(w io.Writer, r *http.Request, params *LayoutParams) {
	rt.renderLayout(w, r, params, true)
}

//nolint:unused
func (rt *Web) runEmptyPage(w io.Writer, r *http.Request, params *LayoutParams) {
	rt.renderLayout(w, r, params, false)
}

// renderLayout wraps params in the page layout; chrome adds the navbar and footer.
func (rt *Web) renderLayout(w io.Writer, r *http.Request, params *LayoutParams, chrome bool) {
	ctx := r.Context()
	props := layout.Props{
		EnabledLanguages: rt.base.EnabledLanguages(),
		Title:            params.Title,
		Description:      params.Description,
		Head:             params.Head,
		Content:          params.Content,
		Assets:           assets,
		HideFooter:       !chrome,
	}
	if chrome {
		props.Navbar = navbar.View(ctx, rt.canViewAllSubs(user.UserBrief(r)), reqPath(r))
	}
	if err := layout.View(ctx, props).Render(w); err != nil {
		slog.WarnContext(ctx, "Error rendering layout", slog.Any("err", err), slog.String("path", r.URL.Path), slog.Any("user", user.UserBrief(r)))
		fmt.Fprintf(w, "Error rendering layout, report to admin: %s", html.EscapeString(err.Error()))
	}
}

func reqPath(r *http.Request) string {
	if r.URL.Path == "/login" || r.URL.Path == "/signup" {
		// when navigating between /login and /signup, retain back or just leave empty
		val := "/"
		if back := r.FormValue("back"); back != "" {
			link, err := url.Parse(back)
			if err == nil {
				val = link.Path
			}
		}
		return val
	}
	return r.URL.Path
}

func (rt *Web) componentModal(w http.ResponseWriter, r *http.Request, component g.Node) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(w); err != nil {
		slog.WarnContext(r.Context(), "Error rendering modal", slog.Any("err", err))
	}
}

func (rt *Web) getOlderSubmissions(ctx context.Context, lookingUser *kilonova.UserBrief, userID int, problem *kilonova.Problem, contest *kilonova.Contest) (*modals.OlderSubmissionsParams, error) {
	var filter = kilonova.SubmissionFilter{
		UserID:    &userID,
		ProblemID: &problem.ID,
		Limit:     5,
	}
	if contest != nil {
		filter.ContestID = &contest.ID
	}
	subs, err := rt.base.Submissions(ctx, filter, true, lookingUser)
	if err != nil {
		return nil, err
	}
	allFinished := true
	for _, sub := range subs.Submissions {
		if sub.Status != kilonova.StatusFinished {
			allFinished = false
		}
	}

	return &modals.OlderSubmissionsParams{
		UserID:  userID,
		Problem: problem,
		Contest: contest,
		Limit:   5,

		Submissions: subs,
		NumHidden:   subs.Count - len(subs.Submissions),
		AllFinished: allFinished,
		AutoReload:  true,
	}, nil
}

func parseSubsViewerQuery(r *http.Request) submissions.SubsViewerQuery {
	q := r.URL.Query()
	query := submissions.SubsViewerQuery{
		Status:    q.Get("status"),
		Lang:      q.Get("lang"),
		Ordering:  q.Get("ordering"),
		Ascending: q.Get("ascending") == "true",
		Page:      1,
	}
	if page, err := strconv.Atoi(q.Get("page")); err == nil && page > 0 {
		query.Page = page
	}
	if uid, err := strconv.Atoi(q.Get("user_id")); err == nil && uid > 0 {
		query.UserID = &uid
	}
	if username := q.Get("username"); username != "" {
		query.Username = &username
	}
	if pid, err := strconv.Atoi(q.Get("problem_id")); err == nil && pid > 0 {
		query.ProblemID = &pid
	}
	if plid, err := strconv.Atoi(q.Get("problem_list_id")); err == nil && plid > 0 {
		query.ProblemListID = &plid
	}
	if cid, err := strconv.Atoi(q.Get("contest_id")); err == nil && cid > 0 {
		query.ContestID = &cid
	}
	if scoreStr := q.Get("score"); scoreStr != "" {
		if score, err := decimal.NewFromString(scoreStr); err == nil {
			query.Score = &score
		}
	}
	switch ce := q.Get("compile_error"); ce {
	case "true":
		query.CompileError = new(true)
	case "false":
		query.CompileError = new(false)
	}
	if q.Get("accepted_only") == "true" {
		query.Status = "finished"
		query.Score = new(decimal.NewFromInt(100))
	}
	if query.Ordering == "" {
		query.Ordering = "id"
	}
	return query
}

func buildSubsFilter(query submissions.SubsViewerQuery) kilonova.SubmissionFilter {
	filter := kilonova.SubmissionFilter{
		UserID:        query.UserID,
		Username:      query.Username,
		ProblemID:     query.ProblemID,
		ProblemListID: query.ProblemListID,
		ContestID:     query.ContestID,
		Status:        kilonova.Status(query.Status),
		Score:         query.Score,
		CompileError:  query.CompileError,
		Ordering:      query.Ordering,
		Ascending:     query.Ascending,
		Limit:         50,
		Offset:        50 * (query.Page - 1),
	}
	if query.Lang != "" {
		filter.Lang = &query.Lang
	}
	return filter
}
