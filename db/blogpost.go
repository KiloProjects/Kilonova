package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
	sq "github.com/Masterminds/squirrel"
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
	qb := sq.Select("*").From("blog_posts").OrderBy(getBlogPostOrdering(filter.Ordering, filter.Ascending))
	qb = blogPostParams(filter, qb)
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, sql, args...)
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
	qb := sq.Select("COUNT(*)").From("blog_posts")
	qb = blogPostParams(filter, qb)
	sql, args, err := qb.ToSql()
	if err != nil {
		return -1, err
	}

	var cnt int
	err = s.conn.QueryRow(ctx, sql, args...).Scan(&cnt)
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
	slug := kilonova.MakeSlug(fmt.Sprintf("%s %d %s", title, authorID, kilonova.RandomString(12)))
	rows, _ := s.conn.Query(ctx, "INSERT INTO blog_posts (title, author_id, slug) VALUES ($1, $2, $3) RETURNING id", title, authorID, slug)
	id, err := pgx.CollectOneRow(rows, pgx.RowTo[int])
	if err != nil {
		return -1, "", err
	}
	return id, slug, nil
}

func (s *DB) UpdateBlogPost(ctx context.Context, id int, upd kilonova.BlogPostUpdate) error {
	qb := sq.Update("blog_posts").Where(sq.Eq{"id": id})
	if v := upd.Slug; v != nil {
		qb = qb.Set("slug", v)
	}
	if v := upd.Title; v != nil {
		qb = qb.Set("title", v)
	}
	if v := upd.Visible; v != nil {
		qb = qb.Set("visible", v)
		// if is set to visible
		if *v {
			// Published at - first time it was set visible
			qb = qb.Set("published_at", sq.Expr("COALESCE(published_at, NOW())"))
		}
	}
	query, args, err := qb.ToSql()
	if err != nil {
		if err.Error() == "update statements must have at least one Set clause" {
			return kilonova.ErrNoUpdates
		}
		return err
	}
	_, err = s.conn.Exec(ctx, query, args...)
	return err
}

func (s *DB) DeleteBlogPost(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM blog_posts WHERE id = $1", id)
	return err
}

func blogPostParams(filter kilonova.BlogPostFilter, sb sq.SelectBuilder) sq.SelectBuilder {
	where := sq.And{}
	if v := filter.ID; v != nil {
		where = append(where, sq.Eq{"id": v})
	}
	if v := filter.IDs; v != nil {
		where = append(where, sq.Expr("id = ANY(?)", v))
	}
	if v := filter.AuthorID; v != nil {
		where = append(where, sq.Eq{"author_id": v})
	}
	if v := filter.Slug; v != nil {
		where = append(where, sq.Eq{"slug": v})
	}
	if v := filter.AttachmentID; v != nil {
		where = append(where, sq.Expr("EXISTS (SELECT 1 FROM blog_post_attachments_m2m WHERE attachment_id = ? AND blog_post_id = id)", v))
	}

	if filter.Look {
		var id = 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		where = append(where, sq.Expr("EXISTS (SELECT 1 FROM visible_posts(?) WHERE post_id = blog_posts.id)", id))
	}

	if v := filter.Limit; v > 0 {
		sb = sb.Limit(v)
	}
	if v := filter.Offset; v > 0 {
		sb = sb.Offset(v)
	}

	return sb.Where(where)
}

func getBlogPostOrdering(ordering string, ascending bool) string {
	ord := " DESC"
	if ascending {
		ord = "ASC"
	}
	switch ordering {
	case "slug":
		return "slug" + ord + ", id DESC"
	case "published_at":
		return "published_at" + ord + ", id DESC"
	case "author_id":
		return "author_id" + ord + ", id DESC"
	default:
		return "id" + ord
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
