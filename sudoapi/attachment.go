package sudoapi

import (
	"context"
	"io"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
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

func (s *BaseAPI) ProblemAttachment(ctx context.Context, problemID, attachmentID int) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.ProblemAttachment(ctx, problemID, attachmentID)
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

func (s *BaseAPI) UpdateAttachment(ctx context.Context, aid int, upd *kilonova.AttachmentUpdate) *StatusError {
	if err := s.db.UpdateAttachment(ctx, aid, upd); err != nil {
		return WrapError(err, "Couldn't update attachment")
	}
	return nil
}

func (s *BaseAPI) UpdateAttachmentData(ctx context.Context, aid int, data []byte) *StatusError {
	if err := s.db.UpdateAttachmentData(ctx, aid, data); err != nil {
		return WrapError(err, "Couldn't update attachment contents")
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
	var settings = &kilonova.ProblemEvalSettings{}
	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		zap.S().Warn(err)
		return nil, WrapError(err, "Couldn't get problem settings")
	}

	for _, att := range atts {
		if !att.Exec {
			continue
		}
		filename := path.Base(att.Name)
		filename = strings.TrimSuffix(filename, path.Ext(filename))
		if filename == "checker_legacy" && eval.GetLangByFilename(att.Name) != "" {
			settings.CheckerName = att.Name
			settings.LegacyChecker = true
			continue
		}
		if filename == "checker" && eval.GetLangByFilename(att.Name) != "" {
			settings.CheckerName = att.Name
			settings.LegacyChecker = false
			continue
		}

		if att.Name[0] == '_' {
			continue
		}

		if att.Name == ".output_only" {
			settings.OutputOnly = true
			continue
		}
		// If not checker and not skipped, continue searching

		if path.Ext(att.Name) == ".h" || path.Ext(att.Name) == ".hpp" {
			settings.OnlyCPP = true
			settings.HeaderFiles = append(settings.HeaderFiles, att.Name)
		}

		if eval.GetLangByFilename(att.Name) != "" {
			settings.OnlyCPP = true
			settings.GraderFiles = append(settings.GraderFiles, att.Name)
		}
	}

	return settings, nil
}
