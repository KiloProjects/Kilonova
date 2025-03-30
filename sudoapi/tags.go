package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (s *BaseAPI) Tags(ctx context.Context) ([]*kilonova.Tag, error) {
	tags, err := s.db.Tags(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, fmt.Errorf("couldn't get tags: %w", err)
	}
	return tags, nil
}

func (s *BaseAPI) TagsByID(ctx context.Context, tagIDs []int) ([]*kilonova.Tag, error) {
	tags, err := s.db.TagsByID(ctx, tagIDs)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, fmt.Errorf("couldn't get tags: %w", err)
	}
	return tags, nil
}

func (s *BaseAPI) TagsByType(ctx context.Context, tagType kilonova.TagType) ([]*kilonova.Tag, error) {
	tags, err := s.db.TagsByType(ctx, tagType)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, fmt.Errorf("couldn't get tags: %w", err)
	}
	return tags, nil
}

func (s *BaseAPI) RelevantTags(ctx context.Context, tagID int, max int, lookingUser *kilonova.UserBrief) ([]*kilonova.Tag, error) {
	var id int
	if lookingUser != nil {
		id = lookingUser.ID
	}
	tags, err := s.db.RelevantTags(ctx, tagID, max, id)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, fmt.Errorf("couldn't get relevant tags: %w", err)
	}
	return tags, nil
}

func (s *BaseAPI) TagByID(ctx context.Context, id int) (*kilonova.Tag, error) {
	tag, err := s.db.Tag(ctx, id)
	if err != nil || tag == nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}
	return tag, nil
}

func (s *BaseAPI) TagByName(ctx context.Context, name string) (*kilonova.Tag, error) {
	tag, err := s.db.TagByName(ctx, name)
	if err != nil || tag == nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}
	return tag, nil
}

func (s *BaseAPI) TagByLooseName(ctx context.Context, name string) (*kilonova.Tag, error) {
	tag, err := s.db.TagByLooseName(ctx, name)
	if err != nil || tag == nil {
		return nil, fmt.Errorf("tag not found: %w", err)
	}
	return tag, nil
}

func (s *BaseAPI) UpdateTagName(ctx context.Context, tag *kilonova.Tag, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return kilonova.ErrMissingRequired
	}
	if err := s.db.UpdateTagName(ctx, tag.ID, newName); err != nil {
		return fmt.Errorf("couldn't update tag: %w", err)
	}
	s.LogUserAction(ctx, "Changed tag name", slog.Any("tag", tag), slog.String("new_name", newName))
	return nil
}

func (s *BaseAPI) UpdateTagType(ctx context.Context, tag *kilonova.Tag, newType kilonova.TagType) error {
	if err := s.db.UpdateTagType(ctx, tag.ID, newType); err != nil {
		return fmt.Errorf("couldn't update tag: %w", err)
	}
	s.LogUserAction(ctx, "Changed tag type", slog.Any("tag", tag), slog.Any("old_type", tag.Type), slog.Any("new_type", newType))
	return nil
}

func (s *BaseAPI) DeleteTag(ctx context.Context, tag *kilonova.Tag) error {
	if err := s.db.DeleteTag(ctx, tag.ID); err != nil {
		return fmt.Errorf("couldn't delete tag: %w", err)
	}
	s.LogUserAction(ctx, "Deleted tag", slog.Any("tag", tag))
	return nil
}

func (s *BaseAPI) CreateTag(ctx context.Context, name string, tagType kilonova.TagType) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" || tagType == kilonova.TagTypeNone {
		return -1, kilonova.ErrMissingRequired
	}
	id, err := s.db.CreateTag(ctx, name, tagType)
	if err != nil {
		return -1, fmt.Errorf("couldn't create tag: %w", err)
	}
	s.LogUserAction(ctx, "New tag created",
		slog.String("name", name),
		slog.String("type", cases.Title(language.English).String(string(tagType))), // Title case
	)
	return id, nil
}

// original - the OG that will remain after the merge
// toReplace - the one that will be replaced
func (s *BaseAPI) MergeTags(ctx context.Context, original int, toReplace []int) error {
	if err := s.db.MergeTags(ctx, original, toReplace); err != nil {
		return fmt.Errorf("couldn't merge tags: %w", err)
	}
	return nil
}

func (s *BaseAPI) ProblemTags(ctx context.Context, problemID int) ([]*kilonova.Tag, error) {
	tags, err := s.db.ProblemTags(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get problem tags: %w", err)
	}
	return tags, nil
}

func (s *BaseAPI) UpdateProblemTags(ctx context.Context, problemID int, tagIDs []int) error {
	slices.Sort(tagIDs)
	tagIDs = slices.Compact(tagIDs)
	if err := s.db.UpdateProblemTags(ctx, problemID, tagIDs); err != nil {
		return fmt.Errorf("couldn't update problem tags: %w", err)
	}
	return nil
}
