package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

const sessionDuration = time.Hour * 24 * 30

func (s *DB) CreateVerification(ctx context.Context, id int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`INSERT INTO verifications (id, user_id) VALUES (?, ?)`), vid, id)
	return vid, err
}

func (s *DB) GetVerification(ctx context.Context, id string) (int, error) {
	var verif struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.conn.GetContext(ctx, &verif, s.conn.Rebind(`SELECT * FROM verifications WHERE id = ?`), id)
	if err != nil {
		return -1, err
	}
	if time.Since(verif.CreatedAt) > time.Hour {
		return -1, err
	}
	return verif.UserID, err
}

func (s *DB) RemoveVerification(ctx context.Context, verif string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`DELETE FROM verifications WHERE id = ?`), verif)
	return err
}

type dbSession struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UserID    int       `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
}

func (s *DB) CreateSession(ctx context.Context, uid int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := s.conn.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`, vid, uid, time.Now().Add(sessionDuration))
	if err != nil {
		return "", err
	}
	return vid, nil
}

func (s *DB) GetSession(ctx context.Context, sess string) (int, error) {
	session, err := s.getSession(ctx, sess)
	if err != nil {
		return -1, err
	}
	return session.UserID, nil
}

func (s *DB) getSession(ctx context.Context, sess string) (*dbSession, error) {
	var session dbSession
	err := s.conn.GetContext(ctx, &session, s.conn.Rebind(`SELECT * FROM sessions WHERE id = ? AND NOW() < expires_at`), sess)
	if err != nil {
		return nil, errors.New("Unauthed")
	}
	return &session, nil
}

func (s *DB) RemoveSession(ctx context.Context, sess string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`DELETE FROM sessions WHERE id = ?`), sess)
	return err
}

func (s *DB) ExtendSession(ctx context.Context, sid string) (time.Time, error) {
	sess, err := s.getSession(ctx, sid)
	if err != nil || sess == nil {
		return time.Time{}, err
	}
	var newExpiration time.Time
	err = s.conn.GetContext(ctx, &newExpiration, s.conn.Rebind(`UPDATE sessions SET expires_at = $2 WHERE id = $1 RETURNING expires_at`), sess.ID, time.Now().Add(sessionDuration))
	return newExpiration, err
}
