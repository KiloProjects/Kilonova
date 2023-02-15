package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) GetTags(ctx context.Context) ([]*kilonova.Tag, error) {
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
	_, err := s.conn.ExecContext(ctx, "UPDATE tags SET name = $2 WHERE id = $1", id, newName)
	return err
}

func (s *DB) CreateTag(ctx context.Context, name string) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, "INSERT INTO tags (name) VALUES ($1) RETURNING id", name)
	if err != nil {
		return -1, err
	}
	return id, err
}

func (s *DB) RemoveTag(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM tags WHERE id = $1", id)
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
