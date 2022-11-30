package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/KiloProjects/kilonova"
)

type dbPaste struct {
	ID     string `db:"paste_id"`
	SubID  int    `db:"submission_id"`
	UserID int    `db:"author_id"`
}

func (s *DB) CreatePaste(ctx context.Context, p *kilonova.SubmissionPaste) error {
	if p.Submission == nil || p.Author == nil {
		return kilonova.ErrMissingRequired
	}
	p.ID = kilonova.RandomString(6)
	_, err := s.conn.ExecContext(ctx, "INSERT INTO submission_pastes (paste_id, submission_id, author_id) VALUES ($1, $2, $3)", p.ID, p.Submission.ID, p.Author.ID)
	return err
}

func (s *DB) SubmissionPaste(ctx context.Context, id string) (*kilonova.SubmissionPaste, error) {
	var paste dbPaste
	err := s.conn.GetContext(ctx, &paste, "SELECT * FROM submission_pastes WHERE paste_id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToPaste(ctx, &paste)
}

func (s *DB) DeleteSubPaste(ctx context.Context, id string) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM submission_pastes WHERE paste_id = $1", id)
	return err
}

func (s *DB) internalToPaste(ctx context.Context, p *dbPaste) (*kilonova.SubmissionPaste, error) {
	if p == nil {
		return nil, nil
	}
	user, err := s.User(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	sub, err := s.Submission(ctx, p.SubID)
	if err != nil {
		return nil, err
	}
	return &kilonova.SubmissionPaste{
		ID:         p.ID,
		Submission: sub,
		Author:     user.ToBrief(),
	}, nil
}