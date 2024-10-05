package sudoapi

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova"
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
		ctx = context.WithValue(context.WithoutCancel(ctx), util.AuthedUserKey, author)
		att, err := s.Attachment(ctx, aid)
		if err != nil {
			zap.S().Warn(err, aid)
			return
		}
		attrs := []slog.Attr{}

		pbs, _ := s.Problems(ctx, kilonova.ProblemFilter{AttachmentID: &aid})
		if len(pbs) == 0 {
			posts, _ := s.BlogPosts(ctx, kilonova.BlogPostFilter{AttachmentID: &aid})
			if len(posts) == 0 {
				attrs = append(attrs, slog.String("error", "Orphaned"))
			} else if len(posts) == 1 {
				attrs = append(attrs, slog.Any("post", posts[0]))
			} else {
				attrs = append(attrs, slog.String("error", "In multiple posts"))
			}
		} else if len(pbs) == 1 {
			attrs = append(attrs, slog.Any("problem", pbs[0]))
		} else {
			attrs = append(attrs, slog.String("error", "In multiple problems"))
		}

		attrs = append(attrs, slog.String("attachment_name", att.Name))

		s.LogVerbose(ctx, "Attachment updated", attrs...)
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

var statementRegex = regexp.MustCompile(`statement-([a-z]+)(?:-([a-z]+))?\.([a-z]+)`)

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
			Type:     matches[2],
			Format:   matches[3],
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
	r, err := s.GetAttachmentRender(attID, renderType)
	if err == nil {
		data, err := io.ReadAll(r)
		if err == nil {
			return data, true
		} else {
			zap.S().Warn("Error reading cache: ", err)
		}
	}
	return nil, false
}

func (s *BaseAPI) FormatDescName(lang, format, t string) string {
	if t == "" {
		return fmt.Sprintf("statement-%s.%s", lang, format)
	} else {
		return fmt.Sprintf("statement-%s-%s.%s", lang, t, format)
	}
}

func (s *BaseAPI) RenderedProblemDesc(ctx context.Context, problem *kilonova.Problem, lang, format, t string) ([]byte, *StatusError) {
	if len(lang) > 10 || len(format) > 10 {
		return nil, Statusf(400, "Not even trying to search for this description variant")
	}
	name := s.FormatDescName(lang, format, t)
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
		if err := s.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
			zap.S().Warn("Couldn't save attachment to cache: ", err)
		}
		return buf, nil
	default:
		return s.AttachmentData(ctx, att.ID)
	}
}

func (s *BaseAPI) RenderedBlogPostDesc(ctx context.Context, post *kilonova.BlogPost, lang, format, t string) ([]byte, *StatusError) {
	if len(lang) > 10 || len(format) > 10 {
		return nil, Statusf(400, "Not even trying to search for this description variant")
	}
	name := s.FormatDescName(lang, format, t)
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
		if err := s.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
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

	var whitelistC, whitelistCPP bool
	var biggestCPP string

	for _, att := range atts {
		if !att.Exec {
			continue
		}
		filename := path.Base(att.Name)
		filename = strings.TrimSuffix(filename, path.Ext(filename))
		if filename == "checker_legacy" && s.LanguageFromFilename(att.Name) != "" {
			settings.CheckerName = att.Name
			settings.LegacyChecker = true
			continue
		}
		if filename == "checker" && s.LanguageFromFilename(att.Name) != "" {
			settings.CheckerName = att.Name
			settings.LegacyChecker = false
			continue
		}

		if att.Name[0] == '_' {
			continue
		}

		// If not checker and not skipped, continue searching

		if path.Ext(att.Name) == ".h" {
			whitelistC = true
			settings.HeaderFiles = append(settings.HeaderFiles, att.Name)
		}

		if path.Ext(att.Name) == ".hpp" {
			whitelistCPP = true
			settings.HeaderFiles = append(settings.HeaderFiles, att.Name)
		}

		if lang := s.LanguageFromFilename(att.Name); lang != "" {
			if strings.HasPrefix(lang, "cpp") {
				whitelistCPP = true
				if lang > biggestCPP {
					biggestCPP = lang
				}
			} else if lang == "c" {
				whitelistC = true
			} else {
				settings.LanguageWhitelist = append(settings.LanguageWhitelist, lang)
			}
			if att.Size > 0 && att.Name != ".output_only" {
				// .output_only is special since it's just a flag file, but the others should be included
				settings.GraderFiles = append(settings.GraderFiles, att.Name)
			}
		}
	}

	if whitelistCPP {
		// limit cpp version to the ones >= the grader has
		for name := range s.EnabledLanguages() {
			if strings.HasPrefix(name, "cpp") && name >= biggestCPP {
				settings.LanguageWhitelist = append(settings.LanguageWhitelist, name)
			}
		}
	} else if whitelistC {
		// Allow C and don't limit cpp version
		settings.LanguageWhitelist = append(settings.LanguageWhitelist, "c")
		for name := range s.EnabledLanguages() {
			if strings.HasPrefix(name, "cpp") {
				settings.LanguageWhitelist = append(settings.LanguageWhitelist, name)
			}
		}
	}

	return settings, nil
}

// ProblemLanguages wraps around ProblemSettings to provide a better interface to expose to the API
// And deduplicate separate code that handles allowed submission languages
func (s *BaseAPI) ProblemLanguages(ctx context.Context, problemID int) ([]*Language, *StatusError) {
	settings, err := s.ProblemSettings(ctx, problemID)
	if err != nil {
		return nil, err
	}
	langs := make([]*Language, 0, len(s.EnabledLanguages()))
	if len(settings.LanguageWhitelist) == 0 {
		for name := range s.EnabledLanguages() {
			langs = append(langs, s.Language(name))
		}
	} else {
		for _, val := range settings.LanguageWhitelist {
			v := s.Language(val)
			if v == nil {
				slog.Warn("Language found in whitelist but not enabled", slog.String("wh_value", val))
			}
			langs = append(langs, v)
		}
	}

	slices.SortFunc(langs, func(a, b *Language) int { return cmp.Compare(a.InternalName, b.InternalName) })
	return langs, nil
}
