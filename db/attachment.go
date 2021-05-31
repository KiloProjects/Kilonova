package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
)

const createAttachmentQuery = "INSERT INTO attachments (problem_id, visible, name, data) VALUES (?, ?, ?, ?) RETURNING id;"

func (a *DB) CreateAttachment(ctx context.Context, att *kilonova.Attachment) error {
	if att.ProblemID == 0 || att.Data == nil {
		return kilonova.ErrMissingRequired
	}
	if _, err := a.Attachments(ctx, false, kilonova.AttachmentFilter{ProblemID: &att.ProblemID, Name: &att.Name}); err != nil {
		return kilonova.ErrAttachmentExists
	}

	var id int
	err := a.conn.GetContext(ctx, &id, a.conn.Rebind(createAttachmentQuery), att.ProblemID, att.Visible, att.Name, att.Data)
	if err == nil {
		att.ID = id
	}
	return err
}

func (a *DB) Attachment(ctx context.Context, id int) (*kilonova.Attachment, error) {
	var attachment kilonova.Attachment
	err := a.conn.GetContext(ctx, &attachment, a.conn.Rebind("SELECT * FROM attachments WHERE id = ? LIMIT 1"), id)
	return &attachment, err
}

func (a *DB) Attachments(ctx context.Context, getData bool, filter kilonova.AttachmentFilter) ([]*kilonova.Attachment, error) {
	var attachments []*kilonova.Attachment
	where, args := attachmentFilterQuery(&filter)
	toSelect := "*"
	if !getData {
		toSelect = "id, created_at, problem_id, visible, name, data_size" // Make sure to keep this in sync
	}
	query := a.conn.Rebind("SELECT " + toSelect + " FROM attachments WHERE " + strings.Join(where, " AND ") + " ORDER BY name ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := a.conn.SelectContext(ctx, &attachments, query, args...)
	return attachments, err
}

const attachmentUpdateStatement = "UPDATE attachments SET %s WHERE id = ?"

func (a *DB) UpdateAttachment(ctx context.Context, id int, upd kilonova.AttachmentUpdate) error {
	toUpd, args := attachmentUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := a.conn.Rebind(fmt.Sprintf(attachmentUpdateStatement, strings.Join(toUpd, ", ")))
	_, err := a.conn.ExecContext(ctx, query, args...)
	return err
}

func (a *DB) DeleteAttachment(ctx context.Context, attid int) error {
	_, err := a.conn.ExecContext(ctx, a.conn.Rebind("DELETE FROM attachments WHERE id = ?"), attid)
	return err
}

func (a *DB) DeleteAttachments(ctx context.Context, pbid int) error {
	_, err := a.conn.ExecContext(ctx, a.conn.Rebind("DELETE FROM attachments WHERE problem_id = ?"), pbid)
	return err
}

func attachmentFilterQuery(filter *kilonova.AttachmentFilter) ([]string, []interface{}) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, v)
	}
	if v := filter.Name; v != nil {
		where, args = append(where, "name = ?"), append(args, v)
	}
	if v := filter.ProblemID; v != nil {
		where, args = append(where, "problem_id = ?"), append(args, v)
	}
	if v := filter.Visible; v != nil {
		where, args = append(where, "visible = ?"), append(args, v)
	}
	return where, args
}

func attachmentUpdateQuery(upd *kilonova.AttachmentUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Data; v != nil {
		toUpd, args = append(toUpd, "data = ?"), append(args, v)
	}
	if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}
	if v := upd.Visible; v != nil {
		toUpd, args = append(toUpd, "visible = ?"), append(args, v)
	}
	return toUpd, args
}
