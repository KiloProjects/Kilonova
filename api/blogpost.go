package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) userBlogPosts(w http.ResponseWriter, r *http.Request) {
	var args struct {
		UserID int `json:"id"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}
	posts, err := s.base.UserBlogPosts(r.Context(), args.UserID, util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}

	returnData(w, posts)
}

func (s *API) blogPostByID(ctx context.Context, _ struct{}) (*kilonova.BlogPost, error) {
	return util.BlogPostContext(ctx), nil
}

func (s *API) blogPostBySlug(w http.ResponseWriter, r *http.Request) {
	post, err := s.base.BlogPostBySlug(r.Context(), r.FormValue("slug"))
	if err != nil {
		statusError(w, err)
		return
	}

	if !s.base.IsBlogPostVisible(util.UserBrief(r), post) {
		errorData(w, "can't view this post", http.StatusForbidden)
		return
	}

	returnData(w, post)
}

func (s *API) validateBlogPostEditor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.base.IsBlogPostEditor(util.UserBrief(r), util.BlogPost(r)) {
			errorData(w, "You must be authorized to edit blog posts", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *API) createBlogPost(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Title    string  `json:"title"`
		Body     *string `json:"body"`
		BodyLang *string `json:"body_lang"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	// Do the check before post creation because it'd be awkward to create the post and then show the error
	if args.BodyLang != nil && !(*args.BodyLang == "en" || *args.BodyLang == "ro") {
		errorData(w, "Invalid initial language", 400)
		return
	}

	id, slug, err := s.base.CreateBlogPost(r.Context(), args.Title, util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}
	if args.Body != nil && args.BodyLang != nil {
		if err := s.base.CreateBlogPostAttachment(r.Context(), &kilonova.Attachment{
			Visible: false,
			Private: false,
			Exec:    false,
			Name:    fmt.Sprintf("statement-%s.md", *args.BodyLang),
		}, id, strings.NewReader(*args.Body), &util.UserBrief(r).ID,
		); err != nil {
			slog.WarnContext(r.Context(), "Couldn't initialize blog post attachment", slog.Any("err", err), slog.Any("post_id", id))
		}
	}

	returnData(w, struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
	}{ID: id, Slug: slug})
}

func (s *API) updateBlogPost(w http.ResponseWriter, r *http.Request) {
	var args kilonova.BlogPostUpdate
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.base.UpdateBlogPost(r.Context(), util.BlogPost(r).ID, args); err != nil {
		statusError(w, err)
		return
	}

	slug := util.BlogPost(r).Slug
	if args.Slug != nil {
		slug = kilonova.MakeSlug(*args.Slug)
	}

	returnData(w, struct {
		Slug    string `json:"slug"`
		Message string `json:"message"`
	}{slug, "Updated blog post"})
}

func (s *API) deleteBlogPost(ctx context.Context, _ struct{}) error {
	return s.base.DeleteBlogPost(context.WithoutCancel(ctx), util.BlogPostContext(ctx))
}
