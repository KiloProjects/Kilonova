package sudoapi

import (
	"context"
	"io"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

// Expected attachment behavior:
//   - Private = true => Visible = false

func (s *BaseAPI) Attachment(ctx context.Context, id int) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, id)
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) AttachmentByName(ctx context.Context, problemID int, name string) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.AttachmentByName(ctx, problemID, name)
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) CreateAttachment(ctx context.Context, att *kilonova.Attachment, problemID int, r io.Reader) *StatusError {
	// since we store the attachment in the database (hopefully, just for now),
	// we need to read the data into a byte array and send it over.
	// I cannot emphasize how inefficient this is.
	data, err := io.ReadAll(r)
	if err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't read attachment data")
	}

	if att.Private {
		att.Visible = false
	}
	if err := s.db.CreateAttachment(ctx, att, problemID, data); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create attachment")
	}
	return nil
}

func (s *BaseAPI) DeleteAttachment(ctx context.Context, attachmentID int) *StatusError {
	if err := s.db.DeleteAttachment(ctx, attachmentID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete attachment")
	}
	return nil
}

func (s *BaseAPI) DeleteAttachments(ctx context.Context, problemID int, attIDs []int) (int, *StatusError) {
	num, err := s.db.DeleteAttachments(ctx, problemID, attIDs)
	if err != nil {
		zap.S().Warn(err)
		return -1, WrapError(err, "Couldn't delete attachments")
	}
	return int(num), nil
}

func (s *BaseAPI) ProblemAttachments(ctx context.Context, problemId int) ([]*kilonova.Attachment, *StatusError) {
	atts, err := s.db.ProblemAttachments(ctx, problemId, nil)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get attachments")
	}
	return atts, nil
}

func (s *BaseAPI) AttachmentData(ctx context.Context, id int) ([]byte, *StatusError) {
	data, err := s.db.AttachmentData(ctx, id)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't read attachment data")
	}
	return data, nil
}

func (s *BaseAPI) AttachmentDataByName(ctx context.Context, problemID int, name string) ([]byte, *StatusError) {
	data, err := s.db.AttachmentDataByName(ctx, problemID, name)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't read attachment data")
	}
	return data, nil
}

func (s *BaseAPI) ProblemSettings(ctx context.Context, problemID int) (*kilonova.ProblemEvalSettings, *StatusError) {
	config, err := s.db.GetProblemSettings(ctx, problemID)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problem settings")
	}
	return config, nil
}
