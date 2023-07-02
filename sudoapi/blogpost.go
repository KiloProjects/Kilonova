package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) UserBlogPosts(ctx context.Context, userID int) ([]*kilonova.BlogPost, *StatusError) {
	blogPost, err := s.db.BlogPosts(ctx, kilonova.BlogPostFilter{AuthorID: &userID})
	if err != nil {
		return nil, WrapError(err, "Couldn't find blog posts")
	}
	return blogPost, nil
}

func (s *BaseAPI) BlogPost(ctx context.Context, id int) (*kilonova.BlogPost, *StatusError) {
	blogPost, err := s.db.BlogPost(ctx, kilonova.BlogPostFilter{ID: &id})
	if err != nil || blogPost == nil {
		return nil, WrapError(err, "Blog post not found")
	}
	return blogPost, nil
}

func (s *BaseAPI) BlogPostBySlug(ctx context.Context, slug string) (*kilonova.BlogPost, *StatusError) {
	blogPost, err := s.db.BlogPost(ctx, kilonova.BlogPostFilter{Slug: &slug})
	if err != nil || blogPost == nil {
		return nil, WrapError(err, "Blog post not found")
	}
	return blogPost, nil
}

func (s *BaseAPI) UpdateBlogPost(ctx context.Context, id int, upd kilonova.BlogPostUpdate) *StatusError {
	if upd.Title != nil && *upd.Title == "" {
		return Statusf(400, "Title can't be empty!")
	}
	if upd.Slug != nil && *upd.Slug == "" {
		return Statusf(400, "Slug can't be empty!")
	}
	if err := s.db.UpdateBlogPost(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update blog post")
	}
	if upd.Slug != nil {
		atts, err := s.BlogPostAttachments(ctx, id)
		if err != nil {
			zap.S().Warn(err)
		} else {
			for _, att := range atts {
				s.manager.DelAttachmentRender(att.ID)
			}
		}
	}
	return nil
}

func (s *BaseAPI) CreateBlogPost(ctx context.Context, title string, author *kilonova.UserBrief) (int, string, *StatusError) {
	postID, slug, err := s.db.CreateBlogPost(ctx, title, author.ID)
	if err != nil {
		return -1, "", WrapError(err, "")
	}
	return postID, slug, nil
}

func (s *BaseAPI) DeleteBlogPost(ctx context.Context, post *kilonova.BlogPost) *StatusError {
	// Delete attachments first, so they are fully removed from the database
	atts, err := s.BlogPostAttachments(ctx, post.ID)
	if err != nil {
		zap.S().Warn(err)
	} else {
		attIDs := []int{}
		for _, att := range atts {
			attIDs = append(attIDs, att.ID)
		}
		if _, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{IDs: attIDs, BlogPostID: &post.ID}); err != nil {
			zap.S().Warn(err)
		}
	}

	if err := s.db.DeleteBlogPost(ctx, post.ID); err != nil {
		return WrapError(err, "Couldn't delete blog post")
	}
	s.LogUserAction(ctx, "Removed blog post #%d: %s", post.ID, post.Title)
	return nil
}
