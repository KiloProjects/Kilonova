package sudoapi

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) UserBlogPosts(ctx context.Context, userID int, lookingUser *kilonova.UserBrief) ([]*kilonova.BlogPost, error) {
	blogPosts, err := s.db.BlogPosts(ctx, kilonova.BlogPostFilter{AuthorID: &userID, Look: true, LookingUser: lookingUser})
	if err != nil {
		return nil, fmt.Errorf("couldn't find blog posts: %w", err)
	}
	if blogPosts == nil {
		blogPosts = []*kilonova.BlogPost{}
	}
	return blogPosts, nil
}

func (s *BaseAPI) BlogPosts(ctx context.Context, filter kilonova.BlogPostFilter) ([]*kilonova.BlogPost, error) {
	blogPosts, err := s.db.BlogPosts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("couldn't find posts: %w", err)
	}
	if blogPosts == nil {
		blogPosts = []*kilonova.BlogPost{}
	}
	return blogPosts, nil
}

func (s *BaseAPI) CountBlogPosts(ctx context.Context, filter kilonova.BlogPostFilter) (int, error) {
	cnt, err := s.db.CountBlogPosts(ctx, filter)
	if err != nil {
		return -1, fmt.Errorf("couldn't count posts: %w", err)
	}
	return cnt, nil
}

func (s *BaseAPI) BlogPost(ctx context.Context, id int) (*kilonova.BlogPost, error) {
	blogPost, err := s.db.BlogPost(ctx, kilonova.BlogPostFilter{ID: &id})
	if err != nil || blogPost == nil {
		return nil, fmt.Errorf("blog post not found: %w", err)
	}
	return blogPost, nil
}

func (s *BaseAPI) BlogPostBySlug(ctx context.Context, slug string) (*kilonova.BlogPost, error) {
	blogPost, err := s.db.BlogPost(ctx, kilonova.BlogPostFilter{Slug: &slug})
	if err != nil || blogPost == nil {
		return nil, fmt.Errorf("blog post not found: %w", err)
	}
	return blogPost, nil
}

func (s *BaseAPI) UpdateBlogPost(ctx context.Context, id int, upd kilonova.BlogPostUpdate) error {
	if upd.Title != nil && *upd.Title == "" {
		return Statusf(400, "Title can't be empty!")
	}
	if upd.Slug != nil {
		*upd.Slug = kilonova.MakeSlug(*upd.Slug)
		if *upd.Slug == "" {
			return Statusf(400, "Slug can't be empty!")
		}
	}
	if err := s.db.UpdateBlogPost(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't update blog post: %w", err)
	}
	if upd.Slug != nil {
		atts, err := s.BlogPostAttachments(ctx, id)
		if err != nil {
			zap.S().Warn(err)
		} else {
			for _, att := range atts {
				s.DelAttachmentRenders(att.ID)
			}
		}
	}
	return nil
}

func (s *BaseAPI) CreateBlogPost(ctx context.Context, title string, author *kilonova.UserBrief) (int, string, error) {
	postID, slug, err := s.db.CreateBlogPost(ctx, title, author.ID)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't create blog post: %w", err)
	}
	return postID, slug, nil
}

func (s *BaseAPI) DeleteBlogPost(ctx context.Context, post *kilonova.BlogPost) error {
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
		return fmt.Errorf("couldn't delete blog post: %w", err)
	}
	s.LogUserAction(ctx, "Removed blog post", slog.Any("post", post))
	return nil
}
