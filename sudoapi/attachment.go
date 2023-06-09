package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
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

func (s *BaseAPI) CreateAttachment(ctx context.Context, att *kilonova.Attachment, problemID int, r io.Reader, authorID *int) *StatusError {
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
	if err := s.db.CreateAttachment(ctx, att, problemID, data, authorID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create attachment")
	}
	return nil
}

func (s *BaseAPI) UpdateAttachment(ctx context.Context, aid int, upd *kilonova.AttachmentUpdate) *StatusError {
	if err := s.db.UpdateAttachment(ctx, aid, upd); err != nil {
		return WrapError(err, "Couldn't update attachment")
	}
	s.manager.DelAttachmentRender(aid)
	return nil
}

func (s *BaseAPI) UpdateAttachmentData(ctx context.Context, aid int, data []byte, authorID *int) *StatusError {
	if err := s.db.UpdateAttachmentData(ctx, aid, data, authorID); err != nil {
		return WrapError(err, "Couldn't update attachment contents")
	}
	s.manager.DelAttachmentRender(aid)
	return nil
}

func (s *BaseAPI) DeleteAttachment(ctx context.Context, attachmentID int) *StatusError {
	if err := s.db.DeleteAttachment(ctx, attachmentID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete attachment")
	}
	s.manager.DelAttachmentRender(attachmentID)
	return nil
}

func (s *BaseAPI) DeleteAttachments(ctx context.Context, problemID int, attIDs []int) (int, *StatusError) {
	num, err := s.db.DeleteAttachments(ctx, problemID, attIDs)
	if err != nil {
		zap.S().Warn(err)
		return -1, WrapError(err, "Couldn't delete attachments")
	}
	for _, att := range attIDs {
		s.manager.DelAttachmentRender(att)
	}
	return int(num), nil
}

func (s *BaseAPI) ProblemAttachments(ctx context.Context, problemID int) ([]*kilonova.Attachment, *StatusError) {
	atts, err := s.db.ProblemAttachments(ctx, problemID, nil)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
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

var statementRegex = regexp.MustCompile("statement-([a-z]+).([a-z]+)")

func (s *BaseAPI) ProblemDescVariants(ctx context.Context, problemID int, getPrivate bool) ([]*kilonova.StatementVariant, *StatusError) {
	variants := []*kilonova.StatementVariant{}
	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get problem statement variants")
	}

	for _, att := range atts {
		matches := statementRegex.FindStringSubmatch(att.Name)
		if len(matches) == 0 {
			continue
		}
		if att.Private && !getPrivate {
			continue
		}
		variants = append(variants, &kilonova.StatementVariant{
			Language: matches[1],
			Format:   matches[2],
			Private:  att.Private,
		})
	}

	return variants, nil
}

// ProblemRawDesc returns the raw data of the problem description,
// along with a bool meaning if the description is private or not
func (s *BaseAPI) ProblemRawDesc(ctx context.Context, problemID int, lang string, format string) ([]byte, *StatusError) {
	if len(lang) > 10 || len(format) > 10 {
		return nil, Statusf(400, "Not even trying to search for this description variant")
	}
	name := fmt.Sprintf("statement-%s.%s", lang, format)
	data, _, err := s.db.ProblemRawDesc(ctx, problemID, name)
	if err != nil {
		return nil, WrapError(err, "Couldn't get problem description")
	}
	return data, nil
}

func (s *BaseAPI) RenderedProblemDesc(ctx context.Context, problem *kilonova.Problem, lang string, format string) ([]byte, *StatusError) {
	switch format {
	case "md":
		name := fmt.Sprintf("statement-%s.%s", lang, format)
		att, err := s.AttachmentByName(ctx, problem.ID, name)
		if err != nil {
			return nil, err
		}

		if s.manager.HasAttachmentRender(att.ID) {
			r, err := s.manager.GetAttachmentRender(att.ID)
			if err == nil {
				data, err := io.ReadAll(r)
				if err == nil {
					return data, nil
				} else {
					zap.S().Warn("Error reading cache: ", err)
				}
			}
		}
		data, _, err1 := s.db.ProblemRawDesc(ctx, problem.ID, name)
		if err1 != nil {
			return nil, WrapError(err1, "Couldn't get problem description")
		}

		buf, err := s.RenderMarkdown(data, &kilonova.RenderContext{Problem: problem})
		if err != nil {
			return data, WrapError(err, "Couldn't render markdown")
		}
		if err := s.manager.SaveAttachmentRender(att.ID, buf); err != nil {
			zap.S().Warn("Couldn't save attachment to cache: ", err)
		}
		return buf, nil
	default:
		return s.ProblemRawDesc(ctx, problem.ID, lang, format)
	}
}

func (s *BaseAPI) ProblemSettings(ctx context.Context, problemID int) (*kilonova.ProblemEvalSettings, *StatusError) {
	var settings = &kilonova.ProblemEvalSettings{}
	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
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
