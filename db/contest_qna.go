package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

// Contest Questions/Answers and Announcements

type dbContestQuestion struct {
	ID        int       `db:"id"`
	AuthorID  int       `db:"author_id"`
	ContestID int       `db:"contest_id"`
	Question  string    `db:"question"`
	CreatedAt time.Time `db:"created_at"`

	RespondedAt *time.Time `db:"responded_at"`
	Response    *string    `db:"response"`
}

type dbContestAnnouncement struct {
	ID           int       `db:"id"`
	ContestID    int       `db:"contest_id"`
	Announcement string    `db:"announcement"`
	CreatedAt    time.Time `db:"created_at"`
}

func (s *DB) CreateContestQuestion(ctx context.Context, contestID, authorID int, text string) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, `INSERT INTO contest_questions (author_id, contest_id, text) VALUES ($1, $2, $3) RETURNING id`, authorID, contestID, text)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (s *DB) ContestQuestions(ctx context.Context, contestID int) ([]*kilonova.ContestQuestion, error) {
	var qs []*dbContestQuestion
	err := s.conn.SelectContext(ctx, &qs, "SELECT * FROM contest_questions WHERE contest_id = $1", contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ContestQuestion{}, nil
	} else if err != nil {
		return nil, err
	}
	return mapper(qs, s.internalToContestQuestion), nil
}

func (s *DB) ContestQuestion(ctx context.Context, id int) (*kilonova.ContestQuestion, error) {
	var q dbContestQuestion
	err := s.conn.GetContext(ctx, &q, "SELECT * FROM contest_questions WHERE id = $1 LIMIT 1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s.internalToContestQuestion(&q), err
}

func (s *DB) ContestQuestionsByUser(ctx context.Context, contestID, userID int) ([]*kilonova.ContestQuestion, error) {
	var qs []*dbContestQuestion
	err := s.conn.SelectContext(ctx, &qs, "SELECT * FROM contest_questions WHERE contest_id = $1 AND author_id = $2", contestID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ContestQuestion{}, nil
	} else if err != nil {
		return nil, err
	}
	return mapper(qs, s.internalToContestQuestion), nil
}

func (s *DB) AnswerContestQuestion(ctx context.Context, questionID int, response string) error {
	_, err := s.conn.ExecContext(ctx, "UPDATE contest_questions SET responded_at = NOW(), response = $1 WHERE id = $2", response, questionID)
	return err
}

func (s *DB) CreateContestAnnouncement(ctx context.Context, contestID int, text string) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, `INSERT INTO contest_announcements (contest_id, announcement) VALUES ($1, $2) RETURNING id`, contestID, text)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (s *DB) ContestAnnouncements(ctx context.Context, contestID int) ([]*kilonova.ContestAnnouncement, error) {
	var answers []*dbContestAnnouncement
	err := s.conn.SelectContext(ctx, &answers, `SELECT * FROM contest_questions WHERE contest_id = $1`, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.ContestAnnouncement{}, nil
	} else if err != nil {
		return []*kilonova.ContestAnnouncement{}, err
	}
	return mapper(answers, s.internalToContestAnnouncement), nil
}

func (s *DB) ContestAnnouncement(ctx context.Context, id int) (*kilonova.ContestAnnouncement, error) {
	var answer dbContestAnnouncement
	err := s.conn.GetContext(ctx, &answer, `SELECT * FROM contest_questions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s.internalToContestAnnouncement(&answer), err
}

func (s *DB) UpdateContestAnnouncement(ctx context.Context, announcementID int, text string) error {
	_, err := s.conn.ExecContext(ctx, "UPDATE contest_announcements SET announcement = $1 WHERE id = $2", text, announcementID)
	return err
}

func (s *DB) DeleteContestAnnouncement(ctx context.Context, announcementID int) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM contest_announcements WHERE id = $1", announcementID)
	return err
}

func (s *DB) internalToContestQuestion(q *dbContestQuestion) *kilonova.ContestQuestion {
	return &kilonova.ContestQuestion{
		ID:         q.ID,
		AuthorID:   q.AuthorID,
		AskedAt:    q.CreatedAt,
		ContestID:  q.ContestID,
		Text:       q.Question,
		ResponedAt: q.RespondedAt,
		Response:   q.Response,
	}
}

func (s *DB) internalToContestAnnouncement(ann *dbContestAnnouncement) *kilonova.ContestAnnouncement {
	return &kilonova.ContestAnnouncement{
		ID:        ann.ID,
		CreatedAt: ann.CreatedAt,
		ContestID: ann.ContestID,
		Text:      ann.Announcement,
	}
}
