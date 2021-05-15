package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.AttachmentService = &AttachmentService{}

type AttachmentService struct {
	db *sqlx.DB
}

const createAttachmentQuery = "INSERT INTO attachments (problem_id, visible, name, data) VALUES (?, ?, ?, ?) RETURNING id;"

func (a *AttachmentService) CreateAttachment(ctx context.Context, att *kilonova.Attachment) error {
	if att.ProblemID == 0 || att.Data == nil {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := a.db.GetContext(ctx, &id, a.db.Rebind(createAttachmentQuery), att.ProblemID, att.Visible, att.Name, att.Data)
	if err == nil {
		att.ID = id
	}
	return err
}

func (a *AttachmentService) Attachment(ctx context.Context, id int) (*kilonova.Attachment, error) {
	var attachment kilonova.Attachment
	err := a.db.GetContext(ctx, &attachment, a.db.Rebind("SELECT * FROM attachments WHERE id = ? LIMIT 1"), id)
	return &attachment, err
}

func (a *AttachmentService) Attachments(ctx context.Context, getData bool, filter kilonova.AttachmentFilter) ([]*kilonova.Attachment, error) {
	var attachments []*kilonova.Attachment
	where, args := a.filterQueryMaker(&filter)
	toSelect := "*"
	if !getData {
		toSelect = "id, created_at, problem_id, visible, name" // Make sure to keep this in sync
	}
	query := a.db.Rebind("SELECT " + toSelect + " FROM attachments WHERE " + strings.Join(where, " AND ") + " ORDER BY name ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
	err := a.db.SelectContext(ctx, &attachments, query, args...)
	return attachments, err
}

const attachmentUpdateQuery = "UPDATE attachments SET %s WHERE id = ?"

func (a *AttachmentService) UpdateAttachment(ctx context.Context, id int, upd kilonova.AttachmentUpdate) error {
	toUpd, args := a.updateQueryMaker(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := a.db.Rebind(fmt.Sprintf(attachmentUpdateQuery, strings.Join(toUpd, ", ")))
	_, err := a.db.ExecContext(ctx, query, args...)
	return err
}

func (a *AttachmentService) DeleteAttachment(ctx context.Context, attid int) error {
	_, err := a.db.ExecContext(ctx, "DELETE FROM attachments WHERE id = ?", attid)
	return err
}

func (a *AttachmentService) DeleteAttachments(ctx context.Context, pbid int) error {
	_, err := a.db.ExecContext(ctx, "DELETE FROM attachments WHERE problem_id = ?", pbid)
	return err
}

func (a *AttachmentService) filterQueryMaker(filter *kilonova.AttachmentFilter) ([]string, []interface{}) {
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

func (a *AttachmentService) updateQueryMaker(upd *kilonova.AttachmentUpdate) ([]string, []interface{}) {
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

func NewAttachmentService(db *sqlx.DB) kilonova.AttachmentService {
	return &AttachmentService{db}
}
