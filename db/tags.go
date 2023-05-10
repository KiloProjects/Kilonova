package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

func (s *DB) Tags(ctx context.Context) ([]*kilonova.Tag, error) {
	var tags []*kilonova.Tag
	err := s.conn.SelectContext(ctx, &tags, "SELECT * FROM tags ORDER BY name ASC")
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Tag{}, nil
	}
	if err != nil {
		return []*kilonova.Tag{}, err
	}
	return tags, err
}

func (s *DB) TagsByType(ctx context.Context, tagType kilonova.TagType) ([]*kilonova.Tag, error) {
	var tags []*kilonova.Tag
	err := s.conn.SelectContext(ctx, &tags, "SELECT * FROM tags WHERE type = $1 ORDER BY name ASC", tagType)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Tag{}, nil
	}
	if err != nil {
		return []*kilonova.Tag{}, err
	}
	return tags, err
}

func (s *DB) Tag(ctx context.Context, id int) (*kilonova.Tag, error) {
	var tag *kilonova.Tag
	err := s.conn.GetContext(ctx, &tag, "SELECT * FROM tags WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, err
}

func (s *DB) TagByName(ctx context.Context, name string) (*kilonova.Tag, error) {
	var tag *kilonova.Tag
	err := s.conn.GetContext(ctx, &tag, "SELECT * FROM tags WHERE name = $1 LIMIT 1", name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tag, err
}

func (s *DB) UpdateTagName(ctx context.Context, id int, newName string) error {
	_, err := s.pgconn.Exec(ctx, "UPDATE tags SET name = $2 WHERE id = $1", id, newName)
	return err
}

func (s *DB) UpdateTagType(ctx context.Context, id int, newType kilonova.TagType) error {
	_, err := s.pgconn.Exec(ctx, "UPDATE tags SET type = $2 WHERE id = $1", id, newType)
	return err
}

func (s *DB) DeleteTag(ctx context.Context, id int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM tags WHERE id = $1", id)
	return err
}

func (s *DB) CreateTag(ctx context.Context, name string, tagType kilonova.TagType) (int, error) {
	var id int
	err := s.pgconn.QueryRow(ctx, "INSERT INTO tags (name, type) VALUES ($1, $2) RETURNING id", name, tagType).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, err
}

// original - the OG that will remain after the merge
// toReplace - the one that will be replaced
func (s *DB) MergeTags(ctx context.Context, original int, toReplace int) error {
	return pgx.BeginFunc(ctx, s.pgconn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO problem_tags (tag_id, problem_id, position) SELECT $1, problem_id, position FROM problem_tags WHERE tag_id = $2 ON CONFLICT UPDATE", original, toReplace); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, "DELETE FROM problem_tags WHERE tag_id = $1", toReplace); err != nil { // Will also cascade to problem tags
			return err
		}

		return nil
	})
}

func (s *DB) RemoveTag(ctx context.Context, id int) error {
	_, err := s.pgconn.Exec(ctx, "DELETE FROM tags WHERE id = $1", id)
	return err
}

func (s *DB) ProblemTags(ctx context.Context, problemID int) ([]*kilonova.Tag, error) {
	var tags []*kilonova.Tag
	err := s.conn.SelectContext(ctx, &tags, "SELECT tags.* FROM tags, problem_tags WHERE tags.id = problem_tags.tag_id AND problem_tags.problem_id = $1 ORDER BY problem_tags.position ASC", problemID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Tag{}, nil
	}
	if err != nil {
		return []*kilonova.Tag{}, err
	}
	return tags, err
}

func (s *DB) UpdateProblemTags(ctx context.Context, problemID int, tagIDs []int) error {
	return s.updateManyToMany(ctx, "problem_tags", "problem_id", "tag_id", problemID, tagIDs, true)
}
