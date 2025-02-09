package sudoapi

import (
	"context"
	"fmt"
	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"log/slog"
)

var (
	ExternalResourcesEnabled = config.GenFlag("feature.external_resources.enabled", true, "External resources availability on this instance")
)

func (s *BaseAPI) ExternalResources(ctx context.Context, filter kilonova.ExternalResourceFilter) ([]*kilonova.ExternalResource, error) {
	if !ExternalResourcesEnabled.Value() {
		return []*kilonova.ExternalResource{}, nil
	}
	resources, err := s.db.ExternalResources(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("couldn't get external resources: %w", err)
	}
	return resources, nil
}

func (s *BaseAPI) UpdateExternalResource(ctx context.Context, id int, upd kilonova.ExternalResourceUpdate) error {
	if !ExternalResourcesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if err := s.db.UpdateExternalResources(ctx, kilonova.ExternalResourceFilter{ID: &id}, upd); err != nil {
		return fmt.Errorf("couldn't update external resource: %w", err)
	}
	return nil
}

func (s *BaseAPI) IsExternalResourceEditor(user *kilonova.UserBrief, resource *kilonova.ExternalResource) bool {
	if user == nil {
		return false
	}
	pb, err := s.Problem(context.Background(), resource.ProblemID)
	if err != nil {
		slog.WarnContext(context.Background(), "Couldn't get problem from external resource", slog.Any("err", err))
		return resource.ProposedBy != nil && *resource.ProposedBy == user.ID
	}
	return s.IsProblemEditor(user, pb) || (resource.ProposedBy != nil && *resource.ProposedBy == user.ID)
}

func (s *BaseAPI) CreateExternalResource(ctx context.Context, name, description, url string, resType kilonova.ResourceType, author *kilonova.UserBrief, problem *kilonova.Problem, preApproved bool) (*kilonova.ExternalResource, error) {
	if !ExternalResourcesEnabled.Value() {
		return nil, kilonova.ErrFeatureDisabled
	}
	res := &kilonova.ExternalResource{
		Name:        name,
		Description: description,
		URL:         url,
		Type:        resType,
		ProposedBy:  &author.ID,
		ProblemID:   problem.ID,
		Visible:     preApproved,
		Accepted:    preApproved,
		Position:    -1,
	}
	if err := s.db.CreateExternalResource(ctx, res); err != nil {
		return nil, fmt.Errorf("couldn't create external resource: %w", err)
	}

	if !preApproved {
		s.LogToDiscord(ctx, "New external resource proposed, requires approval.", slog.Any("resource", res))
	}

	return res, nil
}

func (s *BaseAPI) DeleteExternalResource(ctx context.Context, id int) error {
	if !ExternalResourcesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if err := s.db.DeleteExternalResources(ctx, kilonova.ExternalResourceFilter{ID: &id}); err != nil {
		return fmt.Errorf("couldn't delete external resource: %w", err)
	}
	return nil
}
