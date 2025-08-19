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
)

// Expected attachment behavior:
//   - Private = true => Visible = false

func (s *BaseAPI) Attachment(ctx context.Context, id int) (*kilonova.Attachment, error) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ID: &id})
	if err != nil || attachment == nil {
		if err != nil {
			slog.WarnContext(ctx, "Could not find attachment", slog.Any("err", err))
		}
		return nil, fmt.Errorf("attachment not found: %w", errors.Join(ErrNotFound, err))
	}
	return attachment, nil
}

func (s *BaseAPI) ProblemAttachment(ctx context.Context, problemID, attachmentID int) (*kilonova.Attachment, error) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, ID: &attachmentID})
	if err != nil || attachment == nil {
		return nil, fmt.Errorf("attachment not found: %w", errors.Join(ErrNotFound, err))
	}
	return attachment, nil
}

func (s *BaseAPI) BlogPostAttachment(ctx context.Context, postID, attachmentID int) (*kilonova.Attachment, error) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, ID: &attachmentID})
	if err != nil || attachment == nil {
		return nil, fmt.Errorf("attachment not found: %w", errors.Join(ErrNotFound, err))
	}
	return attachment, nil
}

func (s *BaseAPI) ProblemAttByName(ctx context.Context, problemID int, name string) (*kilonova.Attachment, error) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, Name: &name})
	if err != nil || attachment == nil {
		return nil, fmt.Errorf("attachment not found: %w", errors.Join(ErrNotFound, err))
	}
	return attachment, nil
}

func (s *BaseAPI) BlogPostAttByName(ctx context.Context, postID int, name string) (*kilonova.Attachment, error) {
	attachment, err := s.db.Attachment(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, Name: &name})
	if err != nil || attachment == nil {
		return nil, fmt.Errorf("attachment not found: %w", errors.Join(ErrNotFound, err))
	}
	return attachment, nil
}

func (s *BaseAPI) CreateProblemAttachment(ctx context.Context, att *kilonova.Attachment, problemID int, r io.Reader, authorID *int) error {
	// since we store the attachment in the database (hopefully, just for now),
	// we need to read the data into a byte array and send it over.
	// I cannot emphasize how inefficient this is.
	data, err := io.ReadAll(r)
	if err != nil {
		slog.WarnContext(ctx, "Could not read attachment data", slog.Any("err", err))
		return fmt.Errorf("couldn't read attachment data: %w", err)
	}

	if att.Private {
		att.Visible = false
	}
	if err := s.db.CreateProblemAttachment(ctx, att, problemID, data, authorID); err != nil {
		slog.WarnContext(ctx, "Could not create attachment", slog.Any("err", err))
		return fmt.Errorf("couldn't create attachment: %w", err)
	}
	return nil
}

func (s *BaseAPI) CreateBlogPostAttachment(ctx context.Context, att *kilonova.Attachment, postID int, r io.Reader, authorID *int) error {
	// since we store the attachment in the database (hopefully, just for now),
	// we need to read the data into a byte array and send it over.
	// I cannot emphasize how inefficient this is.
	data, err := io.ReadAll(r)
	if err != nil {
		slog.WarnContext(ctx, "Could not read attachment data", slog.Any("err", err))
		return fmt.Errorf("couldn't read attachment data: %w", err)
	}

	if att.Private {
		att.Visible = false
	}
	if err := s.db.CreateBlogPostAttachment(ctx, att, postID, data, authorID); err != nil {
		slog.WarnContext(ctx, "Could not create attachment", slog.Any("err", err))
		return fmt.Errorf("couldn't create attachment: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateAttachment(ctx context.Context, aid int, upd *kilonova.AttachmentUpdate) error {
	if err := s.db.UpdateAttachment(ctx, aid, upd); err != nil {
		return fmt.Errorf("couldn't update attachment: %w", err)
	}
	s.DelAttachmentRenders(aid)
	return nil
}

func (s *BaseAPI) UpdateAttachmentData(ctx context.Context, aid int, data []byte, author *kilonova.UserBrief) error {
	var authorID *int
	if author != nil {
		authorID = &author.ID
	}
	if err := s.db.UpdateAttachmentData(ctx, aid, data, authorID); err != nil {
		return fmt.Errorf("couldn't update attachment contents: %w", err)
	}
	s.DelAttachmentRenders(aid)
	go func() {
		ctx = context.WithValue(context.WithoutCancel(ctx), util.AuthedUserKey, author)
		att, err := s.Attachment(ctx, aid)
		if err != nil {
			slog.WarnContext(ctx, "Could not get attachment", slog.Any("err", err), slog.Int("attachmentID", aid))
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

func (s *BaseAPI) DeleteProblemAtts(ctx context.Context, problemID int, attIDs []int) (int, error) {
	num, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, IDs: attIDs})
	if err != nil {
		slog.WarnContext(ctx, "Could not delete problem attachments", slog.Any("err", err))
		return -1, fmt.Errorf("couldn't delete attachments: %w", err)
	}
	for _, att := range attIDs {
		s.DelAttachmentRenders(att)
	}
	return int(num), nil
}

func (s *BaseAPI) DeleteBlogPostAtts(ctx context.Context, postID int, attIDs []int) (int, error) {
	num, err := s.db.DeleteAttachments(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID, IDs: attIDs})
	if err != nil {
		slog.WarnContext(ctx, "Could not delete blog post attachments", slog.Any("err", err))
		return -1, fmt.Errorf("couldn't delete attachments: %w", err)
	}
	for _, att := range attIDs {
		s.DelAttachmentRenders(att)
	}
	return int(num), nil
}

func (s *BaseAPI) ProblemAttachments(ctx context.Context, problemID int) ([]*kilonova.Attachment, error) {
	atts, err := s.db.Attachments(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID})
	if err != nil {
		slog.WarnContext(ctx, "Could not get problem attachments", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get attachments: %w", err)
	}
	return atts, nil
}

func (s *BaseAPI) BlogPostAttachments(ctx context.Context, postID int) ([]*kilonova.Attachment, error) {
	atts, err := s.db.Attachments(ctx, &kilonova.AttachmentFilter{BlogPostID: &postID})
	if err != nil {
		slog.WarnContext(ctx, "Could not get blog post attachments", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get attachments: %w", err)
	}
	return atts, nil
}

func (s *BaseAPI) AttachmentData(ctx context.Context, id int) ([]byte, error) {
	data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ID: &id})
	if err != nil {
		slog.WarnContext(ctx, "Could not get attachment data", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't read attachment data: %w", err)
	}
	return data, nil
}

func (s *BaseAPI) ProblemAttDataByName(ctx context.Context, problemID int, name string) ([]byte, error) {
	data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ProblemID: &problemID, Name: &name})
	if err != nil {
		slog.WarnContext(ctx, "Could not get problem attachment data", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't read attachment data: %w", err)
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

			AttachmentName: att.Name,
			LastUpdatedAt:  att.LastUpdatedAt,
		})
	}

	return variants
}

func (s *BaseAPI) ProblemDescVariants(ctx context.Context, problemID int, getPrivate bool) ([]*kilonova.StatementVariant, error) {
	atts, err := s.ProblemAttachments(ctx, problemID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get problem statement variants", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get problem statement variants: %w", err)
	}

	return s.parseVariants(atts, getPrivate), nil
}

func (s *BaseAPI) BlogPostDescVariants(ctx context.Context, problemID int, getPrivate bool) ([]*kilonova.StatementVariant, error) {
	atts, err := s.BlogPostAttachments(ctx, problemID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get blog post statement variants", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get blog post statement variants: %w", err)
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
			slog.WarnContext(context.Background(), "Error reading attachment cache", slog.Any("err", err))
		}
	}
	return nil, false
}

func (s *BaseAPI) FormatDescName(variant *kilonova.StatementVariant) string {
	if variant.Type == "" {
		return fmt.Sprintf("statement-%s.%s", variant.Language, variant.Format)
	} else {
		return fmt.Sprintf("statement-%s-%s.%s", variant.Language, variant.Type, variant.Format)
	}
}

func (s *BaseAPI) RenderedProblemDesc(ctx context.Context, problem *kilonova.Problem, variant *kilonova.StatementVariant) ([]byte, error) {
	name := s.FormatDescName(variant)
	att, err := s.ProblemAttByName(ctx, problem.ID, name)
	if err != nil {
		return nil, err
	}

	switch variant.Format {
	case "md":
		d, ok := s.getCachedAttachment(att.ID, "mdhtml")
		if ok {
			return d, nil
		}
		data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{ProblemID: &problem.ID, Name: &name})
		if err != nil {
			return nil, fmt.Errorf("couldn't get problem description: %w", err)
		}

		buf, err := s.RenderMarkdown(data, &kilonova.MarkdownRenderContext{Problem: problem})
		if err != nil {
			return data, fmt.Errorf("couldn't render markdown: %w", err)
		}
		if err := s.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
			slog.WarnContext(ctx, "Couldn't save attachment to cache", slog.Any("err", err))
		}
		return buf, nil
	default:
		return s.AttachmentData(ctx, att.ID)
	}
}

func (s *BaseAPI) RenderedBlogPostDesc(ctx context.Context, post *kilonova.BlogPost, variant *kilonova.StatementVariant) ([]byte, error) {
	name := s.FormatDescName(variant)
	att, err := s.BlogPostAttByName(ctx, post.ID, name)
	if err != nil {
		return nil, err
	}

	switch variant.Format {
	case "md":
		d, ok := s.getCachedAttachment(att.ID, "mdhtml")
		if ok {
			return d, nil
		}
		data, err := s.db.AttachmentData(ctx, &kilonova.AttachmentFilter{BlogPostID: &post.ID, Name: &name})
		if err != nil {
			return nil, fmt.Errorf("couldn't get blog post description: %w", err)
		}

		buf, err := s.RenderMarkdown(data, &kilonova.MarkdownRenderContext{BlogPost: post})
		if err != nil {
			return data, fmt.Errorf("couldn't render markdown: %w", err)
		}
		if err := s.SaveAttachmentRender(att.ID, "mdhtml", buf); err != nil {
			slog.WarnContext(ctx, "Couldn't save attachment to cache", slog.Any("err", err))
		}
		return buf, nil
	default:
		return s.AttachmentData(ctx, att.ID)
	}
}

func (s *BaseAPI) ProblemSettings(ctx context.Context, problem *kilonova.Problem) (*kilonova.ProblemEvalSettings, error) {
	var settings = &kilonova.ProblemEvalSettings{}
	atts, err := s.ProblemAttachments(ctx, problem.ID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get problem settings", slog.Any("err", err))
		return nil, fmt.Errorf("couldn't get problem settings: %w", err)
	}

	var whitelistC, whitelistCPP bool
	var biggestCPP string
	checkerStem := "checker"
	if problem.TaskType == kilonova.TaskTypeCommunication {
		checkerStem = "manager"
	}

	for _, att := range atts {
		if !att.Exec {
			continue
		}
		filename := path.Base(att.Name)
		filename = strings.TrimSuffix(filename, path.Ext(filename))
		if filename == checkerStem+"legacy" && s.LanguageFromFilename(ctx, att.Name) != "" {
			settings.CheckerName = att.Name
			settings.LegacyChecker = true
			continue
		}
		if filename == checkerStem && s.LanguageFromFilename(ctx, att.Name) != "" {
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

		if lang := s.LanguageFromFilename(ctx, att.Name); lang != "" {
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
func (s *BaseAPI) ProblemLanguages(ctx context.Context, problem *kilonova.Problem) ([]*Language, error) {
	settings, err := s.ProblemSettings(ctx, problem)
	if err != nil {
		return nil, err
	}
	langs := make([]*Language, 0, len(s.EnabledLanguages()))
	if len(settings.LanguageWhitelist) == 0 {
		for name := range s.EnabledLanguages() {
			langs = append(langs, s.Language(ctx, name))
		}
	} else {
		for _, val := range settings.LanguageWhitelist {
			v := s.Language(ctx, val)
			if v == nil {
				slog.WarnContext(ctx, "Language found in whitelist but not enabled", slog.String("wh_value", val))
			}
			langs = append(langs, v)
		}
	}

	slices.SortFunc(langs, func(a, b *Language) int { return cmp.Compare(a.InternalName, b.InternalName) })
	return langs, nil
}
