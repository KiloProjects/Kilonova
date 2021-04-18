package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.Sessioner = &SessionService{}

type SessionService struct {
	db *sqlx.DB
}

func (s *SessionService) CreateSession(ctx context.Context, uid int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := s.db.ExecContext(ctx, s.db.Rebind(`INSERT INTO sessions (id, user_id) VALUES (?, ?)`), vid, uid)
	if err != nil {
		return "", err
	}
	return vid, nil
}

func (s *SessionService) GetSession(ctx context.Context, sess string) (int, error) {
	var session struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.db.GetContext(ctx, &session, s.db.Rebind(`SELECT * FROM sessions WHERE id = ?`), sess)
	if err != nil {
		return -1, errors.New("Unauthed")
	}
	if time.Now().Sub(session.CreatedAt) > time.Hour*24*30 {
		return -1, errors.New("Unauthed")
	}
	return session.UserID, nil
}

func (s *SessionService) RemoveSession(ctx context.Context, sess string) error {
	_, err := s.db.ExecContext(ctx, s.db.Rebind(`DELETE FROM sessions WHERE id = ?`), sess)
	return err
}

func NewSessionService(db *sqlx.DB) kilonova.Sessioner {
	return &SessionService{db}
}
