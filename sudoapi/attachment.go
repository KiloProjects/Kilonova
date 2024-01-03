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
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

// Expected attachment behavior:
//   - Private = true => Visible = false

func (s *BaseAPI) Attachment(ctx context.Context, id int) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ID: &id})
	if err != nil || attachment == nil {
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) ProblemAttachment(ctx context.Context, problemID, attachmentID int) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, ID: &attachmentID})
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) BlogPostAttachment(ctx context.Context, postID, attachmentID int) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, ID: &attachmentID})
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) ProblemAttByName(ctx context.Context, problemID int, name string) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, Name: &name})
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) BlogPostAttByName(ctx context.Context, postID int, name string) (*kilonova.Attachment, *StatusError) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, Name: &name})
	if err != nil || attachment == nil {
		return nil, WrapError(ErrNotFound, "Attachment not found")
	}
	return attachment, nil
}

func (s *BaseAPI) CreateProblemAttachment(ctx context.Context, att *kilonova.Attachment, problemID int, r io.Reader, authorID *int) *StatusError {
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
	if err := s.db.CreateProblemAttachment(ctx, att, problemID, data, authorID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create attachment")
	}
	return nil
}

func (s *BaseAPI) CreateBlogPostAttachment(ctx context.Context, att *kilonova.Attachment, postID int, r io.Reader, authorID *int) *StatusError {
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
	if err := s.db.CreateBlogPostAttachment(ctx, att, postID, data, authorID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create attachment")
	}
	return nil
}

func (s *BaseAPI) UpdateAttachment(ctx context.Context, aid int, upd *kilonova.AttachmentUpdate) *StatusError {
	if err := s.db.UpdateAttachment(ctx, aid, upd); err != nil {
		return WrapError(err, "Couldn't update attachment")
	}
	s.DelAttachmentRenders(aid)
	return nil
}

func (s *BaseAPI) UpdateAttachmentData(ctx context.Context, aid int, data []byte, author *kilonova.UserBrief) *StatusError {
	var authorID *int
	if author != nil {
		authorID = &author.ID
	}
	if err := s.db.UpdateAttachmentData(ctx, aid, data, authorID); err != nil {
		return WrapError(err, "Couldn't update attachment contents")
	}
	s.DelAttachmentRenders(aid)
	go func() {
		ctx = context.WithValue(context.WithoutCancel(ctx), util.UserKey, author)
		att, err := s.Attachment(ctx, aid)
		if err != nil {
			zap.S().Warn(err, aid)
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Attachment %q ", att.Name))

		pbs, _ := s.Problems(ctx, kilonova.ProblemFilter{AttachmentID: &aid})
		if len(pbs) == 0 {
			posts, _ := s.BlogPosts(ctx, kilonova.BlogPostFilter{AttachmentID: &aid})
			if len(posts) == 0 {
				b.WriteString("(orphaned, somehow) ")
			} else if len(posts) == 1 {
				b.WriteString(fmt.Sprintf("in blog post #%d: %q ", posts[0].ID, posts[0].Slug))
			} else {
				zap.S().Warn("Attachment %d is in multiple posts??", att.ID)
			}
		} else if len(pbs) == 1 {
			b.WriteString(fmt.Sprintf("in problem #%d: %q ", pbs[0].ID, pbs[0].Name))
		} else {
			zap.S().Warn("Attachment %d is in multiple problems??", att.ID)
		}

		b.WriteString("was updated")

		s.LogVerbose(ctx, b.String())
	}()
	return nil
}

func (s *BaseAPI) DeleteProblemAtts(ctx context.Context, problemID int, attIDs []int) (int, *StatusError) {
	num, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, IDs: attIDs})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return -1, WrapError(err, "Couldn't delete attachments")
	}
	for _, att := range attIDs {
		s.DelAttachmentRenders(att)
	}
	return int(num), nil
}

func (s *BaseAPI) DeleteBlogPostAtts(ctx context.Context, postID int, attIDs []int) (int, *StatusError) {
	num, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, IDs: attIDs})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return -1, WrapError(err, "Couldn't delete attachments")
	}
	for _, att := range attIDs {
		s.DelAttachmentRenders(att)
	}
	return int(num), nil
}

func (s *BaseAPI) ProblemAttachments(ctx context.Context, problemID int) ([]*kilonova.Attachment, *StatusError) {
	atts, err := s.db.Attachments(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get attachments")
	}
	return atts, nil
}

func (s *BaseAPI) BlogPostAttachments(ctx context.Context, postID int) ([]*kilonova.Attachment, *StatusError) {
	atts, err := s.db.Attachments(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get attachments")
	}
	return atts, nil
}

func (s *BaseAPI) AttachmentData(ctx context.Context, id int) ([]byte, *StatusError) {
	data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ID: &id})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't read attachment data")
	}
	return data, nil
}

func (s *BaseAPI) ProblemAttDataByName(ctx context.Context, problemID int, name string) ([]byte, *StatusError) {
	data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, Name: &name})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't read attachment data")
	}
	return data, nil
}

var statementRegex = regexp.MustCompile("statement-([a-z]+).([a-z]+)")

func (s *BaseAPI) parseVariants(atts []*kilonova.Attachment, getPrivate bool) []*kilonova.StatementVariant {
	variants := []*kilonova.StatementVariant{}

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

	return variants
}

func (s *BaseAPI) ProblemDescVariants(ctx context.Context, problemID int, getPrivate bool) ([]*kilonova.StatementVariant, *StatusError) {
	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get problem statement variants")
	}

	return s.parseVariants(atts, getPrivate), nil
}

func (s *BaseAPI) BlogPostDescVariants(ctx context.Context, problemID int, getPrivate bool) ([]*kilonova.StatementVariant, *StatusError) {
	atts, err := s.BlogPostAttachments(ctx, problemID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, WrapError(err, "Couldn't get problem statement variants")
	}

	return s.parseVariants(atts, getPrivate), nil
}

func (s *BaseAPI) getCachedAttachment(attID int, renderType string) ([]byte, bool) {
	if s.HasAttachmentRender(attID, renderType) {
		r, err := s.GetAttachmentRender(attID, renderType)
		if err == nil {
			data, err := io.ReadAll(r)
			if err == nil {
				return data, true
			} else {
				zap.S().Warn("Error reading cache: ", err)
			}
		}
	}
	return nil, false
}

func (s *BaseAPI) RenderedProblemDesc(ctx context.Context, problem *kilonova.Problem, lang string, format string) ([]byte, *StatusError) {
	if len(lang) > 10 || len(format) > 10 {
		return nil, Statusf(400, "Not even trying to search for this description variant")
	}
	name := fmt.Sprintf("statement-%s.%s", lang, format)
	att, err := s.ProblemAttByName(ctx, problem.ID, name)
	if err != nil {
		return nil, err
	}

	switch format {
	case "md":
		d, ok := s.getCachedAttachment(att.ID, "mdhtml")
		if ok {
			return d, nil
		}
		data, err1 := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ProblemID: &problem.ID, Name: &name})
		if err1 != nil {
			return nil, WrapError(err1, "Couldn't get problem description")
		}

		buf, err := s.RenderMarkdown(data, &kilonova.RenderContext{Problem: problem})
		if err != nil {
			return data, WrapError(err, "Couldn't render markdown")
		}
		if err := s.manager.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
			zap.S().Warn("Couldn't save attachment to cache: ", err)
		}
		return buf, nil
	default:
		return s.AttachmentData(ctx, att.ID)
	}
}

func (s *BaseAPI) RenderedBlogPostDesc(ctx context.Context, post *kilonova.BlogPost, lang string, format string) ([]byte, *StatusError) {
	if len(lang) > 10 || len(format) > 10 {
		return nil, Statusf(400, "Not even trying to search for this description variant")
	}
	name := fmt.Sprintf("statement-%s.%s", lang, format)
	att, err := s.BlogPostAttByName(ctx, post.ID, name)
	if err != nil {
		return nil, err
	}

	switch format {
	case "md":
		d, ok := s.getCachedAttachment(att.ID, "mdhtml")
		if ok {
			return d, nil
		}
		data, err1 := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{BlogPostID: &post.ID, Name: &name})
		if err1 != nil {
			return nil, WrapError(err1, "Couldn't get blog post description")
		}

		buf, err := s.RenderMarkdown(data, &kilonova.RenderContext{BlogPost: post})
		if err != nil {
			return data, WrapError(err, "Couldn't render markdown")
		}
		if err := s.manager.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
			zap.S().Warn("Couldn't save attachment to cache: ", err)
		}
		return buf, nil
	default:
		return s.AttachmentData(ctx, att.ID)
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
