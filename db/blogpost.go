package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

type dbBlogPost struct {
	ID          int        `db:"id"`
	CreatedAt   time.Time  `db:"created_at"`
	PublishedAt *time.Time `db:"published_at"`
	AuthorID    int        `db:"author_id"`

	Title string `db:"title"`

	Slug    string `db:"slug"` // unique, used in URL
	Visible bool   `db:"visible"`
}

func (s *DB) BlogPost(ctx context.Context, filter kilonova.BlogPostFilter) (*kilonova.BlogPost, error) {
	filter.Limit = 1
	return toSingular(ctx, filter, s.BlogPosts)
}

func (s *DB) BlogPosts(ctx context.Context, filter kilonova.BlogPostFilter) ([]*kilonova.BlogPost, error) {
	fb := newFilterBuilder()
	blogPostParams(filter, fb)

	q := fmt.Sprintf("SELECT * FROM blog_posts WHERE %s %s %s", fb.Where(), getBlogPostOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset))
	rows, _ := s.pgconn.Query(ctx, q, fb.Args()...)
	posts, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[dbBlogPost])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mapper(posts, s.internalToBlogPost), nil
}

func (s *DB) CountBlogPosts(ctx context.Context, filter kilonova.BlogPostFilter) (int, error) {
	fb := newFilterBuilder()
	blogPostParams(filter, fb)

	var cnt int
	err := s.pgconn.QueryRow(ctx, "SELECT COUNT(*) FROM blog_posts WHERE "+fb.Where(), fb.Args()...).Scan(&cnt)
	if errors.Is(err, pgx.ErrNoRows) { // Should never happen
		return 0, nil
	}
	if err != nil {
		return -1, err
	}
	return cnt, nil
}

func (s *DB) CreateBlogPost(ctx context.Context, title string, authorID int) (int, string, error) {
	if title == "" || authorID == 0 {
		return -1, "", kilonova.ErrMissingRequired
	}
	slug := kilonova.MakeSlug(fmt.Sprintf("%s %d %s", title, authorID, kilonova.RandomString(6)))
	rows, _ := s.pgconn.Query(ctx, "INSERT INTO blog_posts (title, author_id, slug) VALUES ($1, $2, $3) RETURNING id", title, authorID, slug)
	id, err := pgx.CollectOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return -1, "", err
	}
	return id, slug, nil
}

func (s *DB) UpdateBlogPost(ctx context.Context, id int, upd kilonova.BlogPostUpdate) error {
	ub := newUpdateBuilder()
	if v := upd.Slug; v != nil {
		ub.AddUpdate("slug = %s", v)
	}
	if v := upd.Title; v != nil {
		ub.AddUpdate("title = %s", v)
	}
	if v := upd.Visible; v != nil {
		ub.AddUpdate("visible = %s", v)
		// if is set to visible
		if *v {
			// Published at - first time it was set visible
			ub.AddUpdate("published_at = COALESCE(published_at, NOW())")
		}
	}
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)
	_, err := s.pgconn.Exec(ctx, "UPDATE blog_posts SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteBlogPost(ctx context.Context, id int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM blog_posts WHERE id = $1", id)
	return err
}

func blogPostParams(filter kilonova.BlogPostFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.AuthorID; v != nil {
		fb.AddConstraint("author_id = %s", v)
	}
	if v := filter.Slug; v != nil {
		fb.AddConstraint("slug = %s", v)
	}
	if v := filter.AttachmentID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM blog_post_attachments_m2m WHERE attachment_id = %s AND blog_post_id = id)", v)
	}

	if filter.Look {
		var id int = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		fb.AddConstraint("EXISTS (SELECT 1 FROM visible_posts(%s) WHERE post_id = blog_posts.id)", id)
	}
}

func getBlogPostOrdering(ordering string, ascending bool) string {
	ord := " DESC"
	if ascending {
		ord = "ASC"
	}
	switch ordering {
	case "slug":
		return "ORDER BY slug" + ord + ", id DESC"
	case "published_at":
		return "ORDER BY published_at" + ord + ", id DESC"
	case "author_id":
		return "ORDER BY author_id" + ord + ", id DESC"
	default:
		return "ORDER BY id" + ord
	}
}

func (s *DB) internalToBlogPost(bp *dbBlogPost) *kilonova.BlogPost {
	return &kilonova.BlogPost{
		ID:        bp.ID,
		CreatedAt: bp.CreatedAt,
		AuthorID:  bp.AuthorID,

		Title: bp.Title,

		Slug:    bp.Slug,
		Visible: bp.Visible,

		PublishedAt: bp.PublishedAt,
	}
}
