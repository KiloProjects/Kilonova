package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type WebCtx string

const (
	PblistCntCacheKey = WebCtx("pblist_cache")
)

func (rt *Web) index() http.HandlerFunc {
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

		var pblists []*kilonova.ProblemList
		if config.Frontend.RootProblemList > 0 {
			pblists, err = rt.base.PblistChildrenLists(r.Context(), config.Frontend.RootProblemList)
			if err != nil {
				zap.S().Warn(err)
				pblists = []*kilonova.ProblemList{}
			}
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}

		if util.UserBrief(r) != nil {
			pblistCache, err := rt.base.NumSolvedFromPblists(r.Context(), listIDs, util.UserBrief(r).ID)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, pblistCache))
			} else if !errors.Is(err, context.Canceled) {
				zap.S().Warn(err)
			}
		}

		rt.runTempl(w, r, templ, &IndexParams{GenContext(r), futureContests, runningContests, pblists})
	}
}

func (rt *Web) problems() http.HandlerFunc {
	templ := rt.parse(nil, "pbs.html", "modals/pbs.html")
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	type filterQuery struct {
		Query   *string `json:"q"`
		Editor  *int    `json:"editor_user"`
		Visible *bool   `json:"published"`

		Tags *string `json:"tags"`

		Page int `json:"page"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var q filterQuery
		r.ParseForm()
		if err := decoder.Decode(&q, r.Form); err != nil {
			zap.S().Warn(err)
		}
		if q.Page < 1 {
			q.Page = 1
		}
		gr := []*kilonova.TagGroup{}
		tags := []*kilonova.Tag{}

		if q.Tags != nil {
			// Parse tag string
			// format example:
			// - !1,2_3,4_5_6,9 = (NOT 1) AND (2 OR 3) AND (4) AND (4 OR 5 OR 6) AND (9)

			var allTags []int

			groups := strings.Split(*q.Tags, ",")
			for _, group := range groups {
				group := group
				if len(group) == 0 {
					continue
				}

				// Check negate flag
				negate := false
				if group[0] == '!' {
					group = group[1:]
					negate = true
				}

				// Parse tags and, if some found, add group
				tags := strings.Split(group, "_")
				var tagIDs []int
				for _, tag := range tags {
					id, err := strconv.Atoi(tag)
					if err == nil {
						allTags = append(allTags, id)
						tagIDs = append(tagIDs, id)
					}
				}
				if len(tagIDs) > 0 {
					gr = append(gr, &kilonova.TagGroup{
						Negate: negate,
						TagIDs: tagIDs,
					})
				}
			}

			slices.Sort(allTags)
			allTags = slices.Compact(allTags)

			var err *kilonova.StatusError
			tags, err = rt.base.TagsByID(r.Context(), allTags)
			if err != nil {
				zap.S().Warn(err)
			}

			var tagMap = make(map[int]*kilonova.Tag)
			for _, tag := range tags {
				tagMap[tag.ID] = tag
			}

			for i := range gr {
				gr[i].TagIDs = slices.DeleteFunc(gr[i].TagIDs, func(id int) bool {
					_, ok := tagMap[id]
					return !ok
				})
			}

			gr = slices.DeleteFunc(gr, func(tg *kilonova.TagGroup) bool {
				return len(tg.TagIDs) == 0
			})

			_ = tags

		}

		pbs, cnt, err := rt.base.SearchProblems(r.Context(), kilonova.ProblemFilter{
			LookingUser: util.UserBrief(r), Look: true,
			FuzzyName: q.Query, EditorUserID: q.Editor, Visible: q.Visible,
			Tags: gr,

			Limit: 50, Offset: (q.Page - 1) * 50,
		}, util.UserBrief(r))
		if err != nil {
			zap.S().Warn(err)
			// TODO: Maybe not fail to load and insted just load on the browser?
			rt.statusPage(w, r, 500, "N-am putut încărca problemele")
			return
		}
		rt.runTempl(w, r, templ, &ProblemSearchParams{GenContext(r), pbs, gr, tags, cnt})
	}
}

func (rt *Web) tags() http.HandlerFunc {
	templ := rt.parse(nil, "tags/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

type TagPageParams struct {
	Ctx *ReqContext

	Tag *kilonova.Tag

	RelevantTags []*kilonova.Tag
	Problems     []*kilonova.ScoredProblem
}

func (rt *Web) tag() http.HandlerFunc {
	templ := rt.parse(nil, "tags/tag.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pbs, err := rt.base.ScoredProblems(r.Context(), kilonova.ProblemFilter{
			Look: true,

			LookingUser: util.UserBrief(r),
			Tags:        []*kilonova.TagGroup{{TagIDs: []int{util.Tag(r).ID}}},
		}, util.UserBrief(r))
		if err != nil {
			zap.S().Warn("Couldn't fetch tag problems: ", err)
			pbs = nil
		}

		relevantTags, err := rt.base.RelevantTags(r.Context(), util.Tag(r).ID, 15)
		if err != nil {
			zap.S().Warn("Couldn't fetch relevant tags: ", err)
			relevantTags = nil
		}
		rt.runTempl(w, r, templ, &TagPageParams{GenContext(r), util.Tag(r), relevantTags, pbs})
	}
}

func (rt *Web) justRender(files ...string) http.HandlerFunc {
	templ := rt.parse(nil, files...)
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &SimpleParams{GenContext(r)})
	}
}

func (rt *Web) pbListIndex() http.HandlerFunc {
	templ := rt.parse(nil, "lists/index.html", "modals/pblist.html", "modals/pbs.html")
	return func(w http.ResponseWriter, r *http.Request) {
		pblists, err := rt.base.ProblemLists(context.Background(), true)
		if err != nil {
			rt.statusPage(w, r, 500, "Eroare la obținerea listelor")
			return
		}

		listIDs := []int{}
		for _, list := range pblists {
			listIDs = append(listIDs, list.ID)
			for _, slist := range list.SubLists {
				listIDs = append(listIDs, slist.ID)
			}
		}

		if util.UserBrief(r) != nil {
			pblistCache, err := rt.base.NumSolvedFromPblists(r.Context(), listIDs, util.UserBrief(r).ID)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, pblistCache))
			} else {
				zap.S().Warn(err)
			}
		}

		rt.runTempl(w, r, templ, &ProblemListParams{GenContext(r), nil, pblists})
	}
}

func (rt *Web) pbListView() http.HandlerFunc {
	templ := rt.parse(nil, "lists/view.html", "modals/pblist.html", "modals/pbs.html", "proposer/createpblist.html")
	return func(w http.ResponseWriter, r *http.Request) {
		listIDs := []int{util.ProblemList(r).ID}
		for _, slist := range util.ProblemList(r).SubLists {
			listIDs = append(listIDs, slist.ID)
		}

		if util.UserBrief(r) != nil {
			pblistCache, err := rt.base.NumSolvedFromPblists(r.Context(), listIDs, util.UserBrief(r).ID)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), PblistCntCacheKey, pblistCache))
			} else {
				zap.S().Warn(err)
			}
		}

		rt.runTempl(w, r, templ, &ProblemListParams{GenContext(r), util.ProblemList(r), nil})
	}
}

func (rt *Web) auditLog() http.HandlerFunc {
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

func (rt *Web) submission() http.HandlerFunc {
	templ := rt.parse(nil, "submission.html")
	return func(w http.ResponseWriter, r *http.Request) {
		fullSub, err := rt.base.Submission(r.Context(), util.Submission(r).ID, util.UserBrief(r))
		if err != nil {
			rt.statusPage(w, r, 500, "N-am putut obține submisia")
			return
		}
		rt.runTempl(w, r, templ, &SubParams{GenContext(r), util.Submission(r), fullSub})
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
// TODO: Restrict on the backend, as well.
func (rt *Web) canViewAllSubs(user *kilonova.UserBrief) bool {
	if AllSubsPage.Value() {
		return true
	}
	return rt.base.IsProposer(user)
}

func (rt *Web) paste() http.HandlerFunc {
	templ := rt.parse(nil, "paste.html")
	return func(w http.ResponseWriter, r *http.Request) {
		fullSub, err := rt.base.FullSubmission(r.Context(), util.Paste(r).Submission.ID)
		if err != nil {
			rt.statusPage(w, r, 500, "N-am putut obține submisia aferentă")
			return
		}
		rt.runTempl(w, r, templ, &PasteParams{GenContext(r), util.Paste(r), fullSub})
	}
}

func (rt *Web) appropriateDescriptionVariant(r *http.Request, variants []*kilonova.StatementVariant) (string, string) {
	prefLang := r.FormValue("pref_lang")
	if prefLang == "" {
		prefLang = util.Language(r)
	}
	prefFormat := r.FormValue("pref_format")
	if prefFormat == "" {
		prefFormat = "md"
	}

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

func (rt *Web) problem() http.HandlerFunc {
	templ := rt.parse(nil, "problem/summary.html", "problem/topbar.html", "modals/contest_sidebar.html", "modals/pb_submit_form.html")
	return func(w http.ResponseWriter, r *http.Request) {
		problem := util.Problem(r)

		var statement = []byte("This problem doesn't have a statement.")

		variants, err := rt.base.ProblemDescVariants(r.Context(), problem.ID, rt.base.IsProblemEditor(util.UserBrief(r), problem))
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Warn("Couldn't get problem desc variants", err)
		}

		foundLang, foundFmt := rt.appropriateDescriptionVariant(r, variants)

		url := fmt.Sprintf("/problems/%d/attachments/statement-%s.%s", problem.ID, foundLang, foundFmt)
		switch foundFmt {
		case "md":
			statement, err = rt.base.RenderedProblemDesc(r.Context(), problem, foundLang, foundFmt)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					zap.S().Warn("Error getting problem markdown: ", err)
				}
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				url, kilonova.GetText(util.Language(r), "desc_link"), url,
			))
		case "":
		default:
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				url, kilonova.GetText(util.Language(r), "desc_link"),
			))
		}

		atts, err := rt.base.ProblemAttachments(r.Context(), problem.ID)
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

		tags, err := rt.base.ProblemTags(r.Context(), util.Problem(r).ID)
		if err != nil {
			zap.S().Warn("Couldn't get tags: ", err)
			tags = []*kilonova.Tag{}
		}

		var initialSubs *sudoapi.Submissions

		if util.UserBrief(r) != nil {
			filter := kilonova.SubmissionFilter{
				ProblemID: &util.Problem(r).ID,
				UserID:    &util.UserBrief(r).ID,

				Limit: 5,
			}
			if util.Contest(r) != nil {
				filter.ContestID = &util.Contest(r).ID
			}
			// subs, err := rt.base.Submissions(r.Context(), filter, true, util.UserBrief(r))
			// No need to filter, since they can see submissions because they can see problem
			subs, err := rt.base.Submissions(r.Context(), filter, false, nil)
			if err == nil {
				initialSubs = subs
			} else {
				zap.S().Warn("Couldn't fetch submissions: ", err)
			}
		}

		rt.runTempl(w, r, templ, &ProblemParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "pb_statement", -1),

			Problem:     util.Problem(r),
			Attachments: atts,
			Tags:        tags,

			Submissions: initialSubs,

			Statement: template.HTML(statement),
			Languages: langs,
			Variants:  variants,

			SelectedLang:   foundLang,
			SelectedFormat: foundFmt,
		})
	}
}

func (rt *Web) problemSubmissions() http.HandlerFunc {
	templ := rt.parse(nil, "problem/pb_submissions.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ProblemTopbarParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "pb_submissions", -1),

			Problem: util.Problem(r),
		})
	}
}

func (rt *Web) problemSubmit() http.HandlerFunc {
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
			Topbar: rt.problemTopbar(r, "pb_submit", -1),

			Languages: langs,
			Problem:   util.Problem(r),
		})
	}
}

var maxPostsPerPage = 10

func (rt *Web) blogPosts() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.FormValue("page"))
		if err != nil {
			page = 1
		}

		filter := kilonova.BlogPostFilter{
			Limit:  maxPostsPerPage,
			Offset: (page - 1) * maxPostsPerPage,

			Look:        true,
			LookingUser: util.UserBrief(r),
		}

		posts, err1 := rt.base.BlogPosts(r.Context(), filter)
		if err1 != nil {
			zap.S().Warn(err1)
			rt.statusPage(w, r, 500, "N-am putut încărca postările")
			return
		}

		numPosts, err1 := rt.base.CountBlogPosts(r.Context(), filter)
		if err1 != nil {
			zap.S().Warn("N-am putut încărca numărul de postări", err1)
			numPosts = 0
		}

		numPages := numPosts / maxPostsPerPage
		if numPosts%maxPostsPerPage > 0 {
			numPages++
		}

		authorIDs := []int{}
		for _, post := range posts {
			authorIDs = append(authorIDs, post.AuthorID)
		}

		authors, err1 := rt.base.UsersBrief(r.Context(), kilonova.UserFilter{IDs: authorIDs})
		if err1 != nil {
			zap.S().Warn(err1)
		}

		authorMap := make(map[int]*kilonova.UserBrief)
		for _, author := range authors {
			authorMap[author.ID] = author
		}

		for _, post := range posts {
			if _, ok := authorMap[post.AuthorID]; !ok {
				authorMap[post.AuthorID] = nil
			}
		}

		rt.runTempl(w, r, templ, &BlogPostIndexParams{
			Ctx: GenContext(r),

			Posts:   posts,
			Authors: authorMap,

			Page:     page,
			NumPages: numPages,
		})
	}
}

func (rt *Web) blogPost() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/view.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		post := util.BlogPost(r)
		var statement = []byte("Post is empty.")

		variants, err := rt.base.BlogPostDescVariants(r.Context(), post.ID, rt.base.IsBlogPostEditor(util.UserBrief(r), post))
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Warn("Couldn't get problem desc variants", err)
		}

		foundLang, foundFmt := rt.appropriateDescriptionVariant(r, variants)

		url := fmt.Sprintf("/posts/%s/attachments/statement-%s.%s", post.Slug, foundLang, foundFmt)
		switch foundFmt {
		case "md":
			statement, err = rt.base.RenderedBlogPostDesc(r.Context(), post, foundLang, foundFmt)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					zap.S().Warn("Error getting problem markdown: ", err)
				}
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				url, kilonova.GetText(util.Language(r), "desc_link_post"), url,
			))
		case "":
		default:
			statement = []byte(fmt.Sprintf(
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				url, kilonova.GetText(util.Language(r), "desc_link_post"),
			))
		}

		atts, err := rt.base.BlogPostAttachments(r.Context(), post.ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || rt.base.IsBlogPostEditor(util.UserBrief(r), post) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		att, err := rt.base.BlogPostAttByName(r.Context(), post.ID, fmt.Sprintf("statement-%s.%s", foundLang, foundFmt))
		if err != nil {
			att = nil
		}

		rt.runTempl(w, r, templ, &BlogPostParams{
			Ctx: GenContext(r),

			Topbar: rt.postTopbar(r, "view"),

			Attachments:  atts,
			Statement:    template.HTML(statement),
			StatementAtt: att,
			Variants:     variants,

			SelectedLang:   foundLang,
			SelectedFormat: foundFmt,
		})
	}
}

func (rt *Web) getFinalLang(prefLang string, variants []*kilonova.StatementVariant) string {
	var finalLang string

	for _, vr := range variants {
		if vr.Format == "md" && vr.Language == prefLang {
			finalLang = vr.Language
		}
	}

	if finalLang == "" {
		for _, vr := range variants {
			if vr.Format == "md" {
				finalLang = vr.Language
			}
		}
	}

	return finalLang
}

func (rt *Web) editBlogPostIndex() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/editIndex.html", "modals/md_att_editor.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {

		variants, err := rt.base.BlogPostDescVariants(r.Context(), util.BlogPost(r).ID, true)
		if err != nil {
			zap.S().Warn(err)
			http.Error(w, "Couldn't get statement variants", 500)
			return
		}

		finalLang := rt.getFinalLang(r.FormValue("pref_lang"), variants)

		var statementData string
		var att *kilonova.Attachment
		if finalLang == "" {
			finalLang = config.Common.DefaultLang
		} else {
			att, err = rt.base.BlogPostAttByName(r.Context(), util.BlogPost(r).ID, fmt.Sprintf("statement-%s.md", finalLang))
			if err != nil {
				zap.S().Warn(err)
				http.Error(w, "Couldn't get post content attachment", 500)
				return
			}
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				zap.S().Warn(err)
				http.Error(w, "Couldn't get post content", 500)
				return
			}
			statementData = string(val)
		}

		rt.runTempl(w, r, templ, &BlogPostParams{
			Ctx: GenContext(r),

			Topbar: rt.postTopbar(r, "editIndex"),

			StatementEditor: &StatementEditorParams{
				Lang: finalLang,
				Data: statementData,
				Att:  att,

				APIPrefix: fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			},
		})
	}
}

func (rt *Web) editBlogPostAtts() http.HandlerFunc {
	templ := rt.parse(nil, "blogpost/editAttachments.html", "modals/att_manager.html", "blogpost/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.base.BlogPostAttachments(r.Context(), util.BlogPost(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		rt.runTempl(w, r, templ, &BlogPostParams{
			Ctx: GenContext(r),

			Topbar: rt.postTopbar(r, "editAttachments"),

			AttachmentEditor: &AttachmentEditorParams{
				Attachments: atts,
				BlogPost:    util.BlogPost(r),
				APIPrefix:   fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			},
		})
	}
}

func (rt *Web) contests() http.HandlerFunc {
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

func (rt *Web) contest() http.HandlerFunc {
	templ := rt.parse(nil, "contest/view.html", "problem/topbar.html", "modals/pbs.html", "modals/contest_sidebar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "contest_general", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestEdit() http.HandlerFunc {
	templ := rt.parse(nil, "contest/edit.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "contest_edit", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestCommunication() http.HandlerFunc {
	templ := rt.parse(nil, "contest/communication.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "contest_communication", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestRegistrations() http.HandlerFunc {
	templ := rt.parse(nil, "contest/registrations.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "contest_registrations", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) contestLeaderboard() http.HandlerFunc {
	templ := rt.parse(nil, "contest/leaderboard.html", "problem/topbar.html")
	return func(w http.ResponseWriter, r *http.Request) {
		// This is assumed to be called from a context in which
		// IsContestVisible is already true
		if !(util.Contest(r).PublicLeaderboard || rt.base.IsContestEditor(util.UserBrief(r), util.Contest(r))) {
			rt.statusPage(w, r, 400, "You are not allowed to view the leaderboard")
			return
		}
		rt.runTempl(w, r, templ, &ContestParams{
			Ctx:    GenContext(r),
			Topbar: rt.problemTopbar(r, "contest_leaderboard", -1),

			Contest: util.Contest(r),
		})
	}
}

func (rt *Web) selfProfile() http.HandlerFunc {
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

func (rt *Web) profile() http.HandlerFunc {
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

func (rt *Web) resendEmail() http.HandlerFunc {
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

func (rt *Web) verifyEmail() http.HandlerFunc {
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

func (rt *Web) resetPassword() http.HandlerFunc {
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

func (rt *Web) checkLockout() func(next http.Handler) http.Handler {
	// templ := rt.parse(nil, "util/lockout.html")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ForceLogin.Value() && util.UserBrief(r) == nil {
				http.Redirect(w, r, "/login?back="+url.PathEscape(r.URL.Path), http.StatusTemporaryRedirect)
				return
			}
			next.ServeHTTP(w, r)
			// rt.runTempl(w, r, templ, &LockoutParams{})
		})
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

	redirect := "/"
	back := r.FormValue("back")
	backURL, err := url.Parse(back)
	if err == nil && backURL.Path != "" {
		redirect = backURL.Path
	}

	http.Redirect(w, r, redirect+"?logout=1", http.StatusTemporaryRedirect)
}

func (rt *Web) chromaCSS() http.HandlerFunc {
	formatter := chtml.New(chtml.WithClasses(true), chtml.TabWidth(4)) // Identical to mdrenderer.go
	var lightBuf, darkBuf bytes.Buffer
	formatter.WriteCSS(&lightBuf, styles.Get("github"))
	formatter.WriteCSS(&darkBuf, styles.Get("github-dark"))
	css := fmt.Sprintf(".light {%s} .dark {%s}", lightBuf.String(), darkBuf.String())
	rez := api.Transform(css, api.TransformOptions{
		Loader: api.LoaderCSS,
		// MinifyWhitespace: true,
		Engines: []api.Engine{
			{Name: api.EngineChrome, Version: "100"},
			{Name: api.EngineFirefox, Version: "100"},
			{Name: api.EngineSafari, Version: "11"},
		},
	})
	// fmt.Println(string(rez.Code))
	if len(rez.Errors) > 0 {
		zap.S().Fatalf("Found %d errors in chroma.css: %#v", len(rez.Errors), rez.Errors)
		return nil
	}

	createTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {

		http.ServeContent(w, r, "chroma.css", createTime, bytes.NewReader(rez.Code))
	}
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
	var pblistCache map[int]int
	switch v := r.Context().Value(PblistCntCacheKey).(type) {
	case map[int]int:
		pblistCache = v
	}

	// Add request-specific functions
	templ.Funcs(template.FuncMap{
		"getText": func(line string, args ...any) template.HTML {
			return template.HTML(kilonova.GetText(lang, line, args...))
		},
		"reqPath": func() string {
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
		"contestLeaderboardVisible": func(c *kilonova.Contest) bool {
			if c.PublicLeaderboard {
				// This is assumed to be called from a context in which
				// IsContestVisible is already true
				// return rt.base.IsContestVisible(authedUser, c)
				return true
			}
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
		"numSolvedPblist": func(listID int) int {
			if pblistCache != nil {
				if val, ok := pblistCache[listID]; ok {
					return val
				}
			}
			zap.S().Warn("Cache miss: ", listID)
			cnt, err := rt.base.NumSolvedFromPblist(context.Background(), listID, authedUser.ID)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					zap.S().Warn(err)
				}
				return -1
			}
			return cnt
		},
	})

	if err := templ.Execute(w, data); err != nil {
		fmt.Fprintf(w, "Error executing template, report to admin: %s", err)
		if !strings.Contains(err.Error(), "broken pipe") {
			zap.S().WithOptions(zap.AddCallerSkip(1)).Warnf("Error executing template: %q %q %#v", err, r.URL.Path, util.UserBrief(r))
		}
	}
}
