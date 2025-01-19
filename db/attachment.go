package db

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

const createAttachmentQuery = "INSERT INTO attachments (visible, private, execable, name, data, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;"

func (a *DB) createAttachment(ctx context.Context, att *kilonova.Attachment, data []byte, authorID *int) (int, error) {
	if data == nil {
		return -1, kilonova.ErrMissingRequired
	}

	var id int
	err := a.conn.QueryRow(ctx, createAttachmentQuery, att.Visible, att.Private, att.Exec, att.Name, data, authorID).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (a *DB) CreateProblemAttachment(ctx context.Context, att *kilonova.Attachment, problemID int, data []byte, authorID *int) error {
	if problemID == 0 {
		return kilonova.ErrMissingRequired
	}
	if _, err := a.Attachments(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, Name: &att.Name}); err != nil {
		return kilonova.ErrAttachmentExists
	}

	id, err := a.createAttachment(ctx, att, data, authorID)
	if err != nil {
		return err
	}

	_, err = a.conn.Exec(ctx, "INSERT INTO problem_attachments_m2m (problem_id, attachment_id) VALUES ($1, $2)", problemID, id)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't associate problem with attachment", slog.Any("err", err))
		return err
	}
	att.ID = id
	return nil
}

func (a *DB) CreateBlogPostAttachment(ctx context.Context, att *kilonova.Attachment, postID int, data []byte, authorID *int) error {
	if postID == 0 {
		return kilonova.ErrMissingRequired
	}
	if _, err := a.Attachments(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, Name: &att.Name}); err != nil {
		return kilonova.ErrAttachmentExists
	}

	id, err := a.createAttachment(ctx, att, data, authorID)
	if err != nil {
		return err
	}

	_, err = a.conn.Exec(ctx, "INSERT INTO blog_post_attachments_m2m (blog_post_id, attachment_id) VALUES ($1, $2)", postID, id)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't associate blog post with attachment", slog.Any("err", err))
		return err
	}
	att.ID = id
	return nil
}

const selectedAttFields = "id, created_at, last_updated_at, last_updated_by, visible, private, execable, name, data_size" // Make sure to keep this in sync

func (a *DB) Attachment(ctx context.Context, filter *kilonova.AttachmentFilter) (*kilonova.Attachment, error) {
	if filter == nil {
		filter = &kilonova.AttachmentFilter{}
	}
	filter.Limit = 1
	return toSingular(ctx, filter, a.Attachments)
}

// TODO: Remove problem_attachments and blog_post_attachments views from DB
func (a *DB) Attachments(ctx context.Context, filter *kilonova.AttachmentFilter) ([]*kilonova.Attachment, error) {
	fb := newFilterBuilder()
	attachmentFilterQuery(filter, fb)

	limit, offset := 0, 0
	if filter != nil {
		limit, offset = filter.Limit, filter.Offset
	}

	query := "SELECT " + selectedAttFields + " FROM attachments WHERE " + fb.Where() + " ORDER BY name ASC " + FormatLimitOffset(limit, offset)
	rows, _ := a.conn.Query(ctx, query, fb.Args()...)
	attachments, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[dbAttachment])
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Attachment{}, nil
	}
	return mapper(attachments, internalToAttachment), err
}

func (a *DB) AttachmentData(ctx context.Context, filter *kilonova.AttachmentFilter) ([]byte, error) {
	var data []byte
	fb := newFilterBuilder()
	attachmentFilterQuery(filter, fb)
	err := a.conn.QueryRow(ctx, "SELECT data FROM attachments WHERE "+fb.Where()+" LIMIT 1", fb.Args()...).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return []byte{}, nil
	}
	return data, err
}

func (a *DB) UpdateAttachment(ctx context.Context, id int, upd *kilonova.AttachmentUpdate) error {
	ub := newUpdateBuilder()
	if v := upd.Name; v != nil {
		ub.AddUpdate("name = %s", v)
	}
	if v := upd.Visible; v != nil {
		ub.AddUpdate("visible = %s", v)
	}
	if v := upd.Private; v != nil {
		ub.AddUpdate("private = %s", v)
	}
	if v := upd.Exec; v != nil {
		ub.AddUpdate("execable = %s", v)
	}
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)
	_, err := a.conn.Exec(ctx, "UPDATE attachments SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (a *DB) UpdateAttachmentData(ctx context.Context, id int, data []byte, updatedBy *int) error {
	_, err := a.conn.Exec(ctx, "UPDATE attachments SET data = $1, last_updated_at = NOW(), last_updated_by = COALESCE($3, last_updated_by) WHERE id = $2", data, id, updatedBy)
	return err
}

func (a *DB) DeleteAttachments(ctx context.Context, filter *kilonova.AttachmentFilter) (int, error) {
	fb := newFilterBuilder()
	attachmentFilterQuery(filter, fb)
	result, err := a.conn.Exec(ctx, "DELETE FROM attachments WHERE "+fb.Where(), fb.Args()...)
	if err != nil {
		return -1, err
	}
	return int(result.RowsAffected()), nil
}

func attachmentFilterQuery(filter *kilonova.AttachmentFilter, fb *filterBuilder) {
	if filter == nil {
		return
	}
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil && len(v) == 0 {
		fb.AddConstraint("0 = 1")
	}
	if v := filter.IDs; len(v) > 0 {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if v := filter.ProblemID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM problem_attachments_m2m WHERE attachment_id = id AND problem_id = %s)", v)
	}
	if v := filter.BlogPostID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM blog_post_attachments_m2m WHERE attachment_id = id AND blog_post_id = %s)", v)
	}
	if v := filter.Name; v != nil {
		fb.AddConstraint("name = %s", v)
	}
	if v := filter.Visible; v != nil {
		fb.AddConstraint("visible = %s", v)
	}
	if v := filter.Private; v != nil {
		fb.AddConstraint("private = %s", v)
	}
	if v := filter.Exec; v != nil {
		fb.AddConstraint("execable = %s", v)
	}
}

type dbAttachment struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Visible   bool      `db:"visible"`
	Private   bool      `db:"private"`
	Exec      bool      `db:"execable"`

	LastUpdatedAt time.Time `db:"last_updated_at"`
	LastUpdatedBy *int      `db:"last_updated_by"`

	Name string `db:"name"`
	Size int    `db:"data_size"`
	//Data []byte `db:"data"`
}

func internalToAttachment(att *dbAttachment) *kilonova.Attachment {
	if att == nil {
		return nil
	}
	return &kilonova.Attachment{
		ID:        att.ID,
		CreatedAt: att.CreatedAt,
		Visible:   att.Visible,
		Private:   att.Private,
		Exec:      att.Exec,

		LastUpdatedAt: att.LastUpdatedAt,
		LastUpdatedBy: att.LastUpdatedBy,

		Name: att.Name,
		Size: att.Size,
	}
}
