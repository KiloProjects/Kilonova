package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

func (s *DB) CreateExternalResource(ctx context.Context, resource *kilonova.ExternalResource) error {
	return s.conn.QueryRow(ctx, `INSERT INTO external_resources (
	name, description, url, visible, accepted, proposed_by, type, problem_id, position, language
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING id, created_at;`, resource.Name, resource.Description, resource.URL, resource.Visible, resource.Accepted, resource.ProposedBy, resource.Type, resource.ProblemID, resource.Position, resource.Language).Scan(&resource.ID, &resource.CreatedAt)
}

func (s *DB) ExternalResources(ctx context.Context, filter kilonova.ExternalResourceFilter) ([]*kilonova.ExternalResource, error) {
	fb := newFilterBuilder()
	externalResourceFilterQuery(&filter, fb)

	rows, _ := s.conn.Query(ctx, fmt.Sprintf("SELECT * FROM external_resources WHERE %s %s %s", fb.Where(), getResourceOrdering(filter.Ordering, filter.Descending), FormatLimitOffset(filter.Limit, filter.Offset)), fb.Args()...)
	resources, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[kilonova.ExternalResource])
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.ExternalResource{}, nil
	}
	return resources, err
}

func (s *DB) UpdateExternalResources(ctx context.Context, filter kilonova.ExternalResourceFilter, upd kilonova.ExternalResourceUpdate) error {
	ub := newUpdateBuilder()
	externalResourceUpdateQuery(&upd, ub)
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	externalResourceFilterQuery(&filter, fb)
	_, err := s.conn.Exec(ctx, "UPDATE external_resources SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteExternalResources(ctx context.Context, filter kilonova.ExternalResourceFilter) error {
	fb := newFilterBuilder()
	externalResourceFilterQuery(&filter, fb)
	_, err := s.conn.Exec(ctx, "DELETE FROM external_resources WHERE "+fb.Where(), fb.Args()...)
	return err
}

func externalResourceFilterQuery(filter *kilonova.ExternalResourceFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.ProblemID; v != nil {
		fb.AddConstraint("problem_id = %s", v)
	}
	if v := filter.Type; v != kilonova.ResourceTypeNone {
		fb.AddConstraint("type = %s", v)
	}
	if v := filter.Language; v != nil {
		fb.AddConstraint("language = %s", v)
	}
	if v := filter.Visible; v != nil {
		fb.AddConstraint("visible = %s", v)
	}
	if v := filter.Accepted; v != nil {
		fb.AddConstraint("accepted = %s", v)
	}
	if v := filter.ProposedBy; v != nil {
		fb.AddConstraint("proposed_by = %s", v)
	}

	if filter.Look {
		id := 0
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}

		fb.AddConstraint("EXISTS (SELECT 1 FROM persistently_visible_pbs(%s) vpbs WHERE vpbs.problem_id = external_resources.problem_id)", id)
	}
}

func externalResourceUpdateQuery(upd *kilonova.ExternalResourceUpdate, fb *updateBuilder) {
	if v := upd.Name; v != nil {
		fb.AddUpdate("name = %s", v)
	}
	if v := upd.Description; v != nil {
		fb.AddUpdate("description = %s", v)
	}
	if v := upd.URL; v != nil {
		fb.AddUpdate("url = %s", v)
	}
	if v := upd.Visible; v != nil {
		fb.AddUpdate("language = %s", v)
	}
	if v := upd.Visible; v != nil {
		fb.AddUpdate("visible = %s", v)
	}
	if v := upd.Accepted; v != nil {
		fb.AddUpdate("accepted = %s", v)
	}
	if v := upd.Type; v != kilonova.ResourceTypeNone {
		fb.AddUpdate("type = %s", v)
	}
	if v := upd.Position; v != nil {
		fb.AddUpdate("position = %s", v)
	}
}

func getResourceOrdering(ordering string, descending bool) string {
	ord := " ASC"
	if descending {
		ord = " DESC"
	}
	switch ordering {
	case "created_at":
		return "ORDER BY created_at" + ord
	default:
		return "ORDER BY position" + ord + ", created_at" + ord + ", id ASC"
	}
}
