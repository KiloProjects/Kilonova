package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

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
	if time.Since(verif.CreatedAt) > time.Hour*24*30 {
		return -1, err
	}
	return verif.UserID, err
}

func (s *DB) RemoveVerification(ctx context.Context, verif string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`DELETE FROM verifications WHERE id = ?`), verif)
	return err
}

func (s *DB) CreateSession(ctx context.Context, uid int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`INSERT INTO sessions (id, user_id) VALUES (?, ?)`), vid, uid)
	if err != nil {
		return "", err
	}
	return vid, nil
}

func (s *DB) GetSession(ctx context.Context, sess string) (int, error) {
	var session struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := s.conn.GetContext(ctx, &session, s.conn.Rebind(`SELECT * FROM sessions WHERE id = ?`), sess)
	if err != nil {
		return -1, errors.New("Unauthed")
	}
	if time.Since(session.CreatedAt) > time.Hour*24*30 {
		return -1, errors.New("Unauthed")
	}
	return session.UserID, nil
}

func (s *DB) RemoveSession(ctx context.Context, sess string) error {
	_, err := s.conn.ExecContext(ctx, s.conn.Rebind(`DELETE FROM sessions WHERE id = ?`), sess)
	return err
}
