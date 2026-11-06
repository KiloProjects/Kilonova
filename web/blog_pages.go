package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/blog"
	"github.com/KiloProjects/kilonova/web/ui/editors"
	g "maragu.dev/gomponents"
)

var maxPostsPerPage uint64 = 10

func (rt *Web) postTopbar(r *http.Request, page string) blog.TopbarParams {
	return blog.TopbarParams{
		IsPostEditor: rt.base.IsBlogPostEditor(user.UserBrief(r), util.BlogPost(r)),
		Post:         util.BlogPost(r),
		Page:         page,
	}
}

// attUpdaters loads the users who last updated atts, for the attachment manager's history table.
func (rt *Web) attUpdaters(ctx context.Context, atts []*kilonova.Attachment) map[int]*kilonova.UserBrief {
	var ids []int
	for _, att := range atts {
		if att.LastUpdatedBy != nil {
			ids = append(ids, *att.LastUpdatedBy)
		}
	}
	users := make(map[int]*kilonova.UserBrief)
	if len(ids) == 0 {
		return users
	}
	list, err := rt.base.UsersBrief(ctx, kilonova.UserFilter{IDs: ids})
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get attachment updaters", slog.Any("err", err))
	}
	for _, u := range list {
		users[u.ID] = u
	}
	return users
}

func (rt *Web) createPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "title.create_post"),
			Content: blog.Create(r.Context()),
		})
	}
}

func (rt *Web) blogPosts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.ParseUint(r.FormValue("page"), 10, 64)
		if err != nil {
			page = 1
		}

		filter := kilonova.BlogPostFilter{
			Limit:  maxPostsPerPage,
			Offset: (page - 1) * maxPostsPerPage,

			Look:        true,
			LookingUser: user.UserBrief(r),
		}

		posts, err := rt.base.BlogPosts(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get blog posts", slog.Any("err", err))
			rt.statusPage(w, r, 500, "N-am putut încărca postările")
			return
		}

		numPosts, err := rt.base.CountBlogPosts(r.Context(), filter)
		if err != nil {
			slog.WarnContext(r.Context(), "Could not get number of posts", slog.Any("err", err))
			numPosts = 0
		}

		numPages := numPosts / int(maxPostsPerPage)
		if numPosts%int(maxPostsPerPage) > 0 {
			numPages++
		}

		authorIDs := []int{}
		for _, post := range posts {
			authorIDs = append(authorIDs, post.AuthorID)
		}

		authors, err := rt.base.UsersBrief(r.Context(), kilonova.UserFilter{IDs: authorIDs})
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get authors", slog.Any("err", err))
		}

		authorMap := make(map[int]*kilonova.UserBrief)
		for _, author := range authors {
			authorMap[author.ID] = author
		}

		var pagination g.Node
		if numPages > 1 {
			pagination = ui.Paginator(ui.PaginatorConfig{
				Page:       int(page),
				NumPages:   numPages,
				ShowArrows: true,
			})
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "title.blogPosts"),
			Content: blog.Index(r.Context(), blog.IndexParams{
				Posts:      posts,
				Authors:    authorMap,
				Pagination: pagination,
			}),
		})
	}
}

func (rt *Web) blogPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		post := util.BlogPost(r)
		var statement = []byte("Post is empty.")

		variants, err := rt.base.BlogPostDescVariants(r.Context(), post.ID, rt.base.IsBlogPostEditor(user.UserBrief(r), post))
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get problem desc variants", slog.Any("err", err))
		}

		descVariant := rt.appropriateDescriptionVariant(r, variants)

		assetLink := fmt.Sprintf("/assets/blogPost/%s/attachment/%s?t=%d", post.Slug, rt.base.FormatDescName(descVariant), descVariant.LastUpdatedAt.UnixMilli())
		switch descVariant.Format {
		case "md":
			statement, err = rt.base.RenderedBlogPostDesc(r.Context(), post, descVariant)
			if err != nil {
				slog.WarnContext(
					r.Context(),
					"Error getting problem markdown",
					slog.Any("err", err),
					slog.Any("variant", descVariant),
					slog.Any("problem", post),
				)
				statement = []byte("Error loading markdown.")
			}
		case "pdf":
			statement = fmt.Appendf(nil,
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>
					<embed class="mx-2 my-2" type="application/pdf" src="%s"
					style="width:95%%; height: 90vh; background: white; object-fit: contain;"></embed>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link_post"), assetLink,
			)
		case "":
		default:
			statement = fmt.Appendf(nil,
				`<a class="btn btn-blue" target="_blank" href="%s">%s</a>`,
				assetLink, kilonova.GetText(util.Language(r), "desc_link_post"),
			)
		}

		atts, err := rt.base.BlogPostAttachments(r.Context(), post.ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}

		if atts != nil {
			newAtts := make([]*kilonova.Attachment, 0, len(atts))
			for _, att := range atts {
				if att.Visible || rt.base.IsBlogPostEditor(user.UserBrief(r), post) {
					newAtts = append(newAtts, att)
				}
			}

			atts = newAtts
		}

		att, err := rt.base.BlogPostAttByName(r.Context(), post.ID, rt.base.FormatDescName(descVariant))
		if err != nil {
			att = nil
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: post.Title + " | Kilonova",
			Content: blog.View(r.Context(), blog.ViewParams{
				Topbar: rt.postTopbar(r, "view"),

				Attachments:  atts,
				Statement:    string(statement),
				StatementAtt: att,
				Variants:     variants,

				SelectedVariant: descVariant,
			}),
		})
	}
}

func (rt *Web) editBlogPostIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		variants, err := rt.base.BlogPostDescVariants(r.Context(), util.BlogPost(r).ID, true)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get blog post desc variants", slog.Any("err", err))
			http.Error(w, "Couldn't get statement variants", 500)
			return
		}

		finalVariant := rt.getFinalVariant(r.FormValue("pref_lang"), r.FormValue("pref_type"), variants)

		var statementData string
		var att *kilonova.Attachment
		att, err = rt.base.BlogPostAttByName(r.Context(), util.BlogPost(r).ID, rt.base.FormatDescName(finalVariant))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Couldn't get blog post content attachment", slog.Any("err", err))
			http.Error(w, "Couldn't get post content attachment", 500)
			return
		}
		if att != nil {
			val, err := rt.base.AttachmentData(r.Context(), att.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Couldn't get blog post content attachment data", slog.Any("err", err))
				http.Error(w, "Couldn't get post content", 500)
				return
			}
			statementData = string(val)
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "postEditIndex"),
			Content: blog.EditIndex(r.Context(), rt.postTopbar(r, "editIndex"), editors.StatementEditorParams{
				Variants: variants,
				Variant:  finalVariant,
				Data:     statementData,
				Att:      att,

				APIPrefix: fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			}),
		})
	}
}

func (rt *Web) editBlogPostAtts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atts, err := rt.base.BlogPostAttachments(r.Context(), util.BlogPost(r).ID)
		if err != nil || len(atts) == 0 {
			atts = nil
		}
		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "postEditAttachments"),
			Content: blog.EditAttachments(r.Context(), rt.postTopbar(r, "editAttachments"), editors.AttachmentEditorParams{
				Attachments: atts,
				BlogPost:    util.BlogPost(r),
				Updaters:    rt.attUpdaters(r.Context(), atts),
				APIPrefix:   fmt.Sprintf("/blogPosts/%d", util.BlogPost(r).ID),
			}),
		})
	}
}
