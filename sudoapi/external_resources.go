package sudoapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/KiloProjects/kilonova/domain/user"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

// ProblemExternalResources never exposes hidden or unapproved links to readers.
// Like editorials and tags, resources are unavailable while a problem is only
// visible through an ongoing contest.
func (s *BaseAPI) ProblemExternalResources(ctx context.Context, problem *kilonova.Problem, viewer *kilonova.UserBrief, language string) ([]*kilonova.ExternalResource, error) {
	if !s.IsProblemFullyVisible(viewer, problem) {
		return nil, kilonova.Statusf(403, "Problem resources are not available")
	}
	filter := kilonova.ExternalResourceFilter{ProblemID: &problem.ID}
	if !s.IsProblemEditor(viewer, problem) {
		filter.Visible, filter.Accepted = new(true), new(true)
	}
	resources, err := s.ExternalResources(ctx, filter)
	if err != nil {
		return nil, err
	}
	if language == "" {
		return resources, nil
	}
	filtered := make([]*kilonova.ExternalResource, 0, len(resources))
	for _, resource := range resources {
		if resource.Language == "" || resource.Language == language {
			filtered = append(filtered, resource)
		}
	}
	return filtered, nil
}

func validateExternalResource(name, rawURL, language string, resourceType kilonova.ResourceType) error {
	if strings.TrimSpace(name) == "" || len(name) > 200 || len(rawURL) > 4096 {
		return kilonova.Statusf(400, "A name and an HTTP(S) URL are required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return kilonova.Statusf(400, "Only absolute HTTP(S) resource URLs without credentials are allowed")
	}
	if language != "" && language != "en" && language != "ro" {
		return kilonova.Statusf(400, "Resource language must be en, ro, or unspecified")
	}
	switch resourceType {
	case kilonova.ResourceTypeEditorial, kilonova.ResourceTypeVideo, kilonova.ResourceTypeSolution, kilonova.ResourceTypeArticle, kilonova.ResourceTypeOther:
		return nil
	default:
		return kilonova.Statusf(400, "Invalid resource type")
	}
}
func (s *BaseAPI) ExternalResources(ctx context.Context, filter kilonova.ExternalResourceFilter) ([]*kilonova.ExternalResource, error) {
	if !flags.ExternalResourcesEnabled.Value() {
		return []*kilonova.ExternalResource{}, nil
	}
	resources, err := s.db.ExternalResources(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("couldn't get external resources: %w", err)
	}
	return resources, nil
}

func (s *BaseAPI) UpdateExternalResource(ctx context.Context, problem *kilonova.Problem, id int, upd kilonova.ExternalResourceUpdate) error {
	if !flags.ExternalResourcesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if !s.IsProblemEditor(user.UserBriefContext(ctx), problem) {
		return kilonova.Statusf(403, "Only problem editors may update resources")
	}
	resources, err := s.db.ExternalResources(ctx, kilonova.ExternalResourceFilter{ID: &id, ProblemID: &problem.ID})
	if err != nil {
		return err
	}
	if len(resources) != 1 {
		return kilonova.ErrNotFound
	}
	resource := resources[0]
	if upd.Name != nil {
		resource.Name = *upd.Name
	}
	if upd.URL != nil {
		resource.URL = *upd.URL
	}
	if upd.Language != nil {
		resource.Language = *upd.Language
	}
	if upd.Type != kilonova.ResourceTypeNone {
		resource.Type = upd.Type
	}
	if err := validateExternalResource(resource.Name, resource.URL, resource.Language, resource.Type); err != nil {
		return err
	}
	return s.db.UpdateExternalResources(ctx, kilonova.ExternalResourceFilter{ID: &id, ProblemID: &problem.ID}, upd)
}

func (s *BaseAPI) CreateExternalResource(ctx context.Context, resource *kilonova.ExternalResource, problem *kilonova.Problem) error {
	if !flags.ExternalResourcesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	author := user.UserBriefContext(ctx)
	if !s.IsProblemEditor(author, problem) {
		return kilonova.Statusf(403, "Only problem editors may add resources")
	}
	if err := validateExternalResource(resource.Name, resource.URL, resource.Language, resource.Type); err != nil {
		return err
	}
	resource.ProposedBy = &author.ID
	resource.ProblemID = problem.ID
	if err := s.db.CreateExternalResource(ctx, resource); err != nil {
		return fmt.Errorf("couldn't create external resource: %w", err)
	}
	if !resource.Accepted {
		s.LogToDiscord(ctx, "New external resource requires approval.", slog.Any("resource", resource))
	}
	return nil
}

func (s *BaseAPI) DeleteExternalResource(ctx context.Context, problem *kilonova.Problem, id int) error {
	if !flags.ExternalResourcesEnabled.Value() {
		return kilonova.ErrFeatureDisabled
	}
	if !s.IsProblemEditor(user.UserBriefContext(ctx), problem) {
		return kilonova.Statusf(403, "Only problem editors may delete resources")
	}
	if err := s.db.DeleteExternalResources(ctx, kilonova.ExternalResourceFilter{ID: &id, ProblemID: &problem.ID}); err != nil {
		return fmt.Errorf("couldn't delete external resource: %w", err)
	}
	return nil
}
