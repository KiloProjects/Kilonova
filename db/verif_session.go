package db

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/jackc/pgx/v5"
)

const sessionDuration = time.Hour * 24 * 30

type dbVerification struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UserID    int       `db:"user_id"`
}

func (s *DB) CreateVerification(ctx context.Context, id int) (string, error) {
	// If there is a collision, at least it will be from that user already
	vid := kilonova.RandomSaltedString(strconv.Itoa(id))
	_, err := s.conn.Exec(ctx, `INSERT INTO verifications (id, user_id) VALUES ($1, $2)`, vid, id)
	return vid, err
}

func (s *DB) GetVerification(ctx context.Context, id string) (int, error) {
	rows, _ := s.conn.Query(ctx, `SELECT * FROM verifications WHERE id = $1`, id)
	verif, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[dbVerification])
	if err != nil {
		return -1, err
	}
	if time.Since(verif.CreatedAt) > time.Hour {
		return -1, err
	}
	return verif.UserID, err
}

func (s *DB) RemoveVerification(ctx context.Context, verif string) error {
	_, err := s.conn.Exec(ctx, `DELETE FROM verifications WHERE id = $1`, verif)
	return err
}

type Session struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UserID    int       `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
}

func (sess *Session) Expired() bool {
	return sess.ExpiresAt.Before(time.Now())
}

func (s *DB) CreateSession(ctx context.Context, uid int) (string, error) {
	// If there is a collision, at least it will be from that user already
	vid := kilonova.RandomSaltedString(strconv.Itoa(uid))
	_, err := s.conn.Exec(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`, vid, uid, time.Now().Add(sessionDuration))
	if err != nil {
		return "", err
	}
	return vid, nil
}

func (s *DB) UserSessions(ctx context.Context, uid int) ([]*Session, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM sessions WHERE user_id = $1 ORDER BY created_at DESC", uid)
	sessions, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Session])
	return sessions, err
}

func (s *DB) GetSession(ctx context.Context, sess string) (int, error) {
	session, err := s.getSession(ctx, sess)
	if err != nil {
		return -1, err
	}
	return session.UserID, nil
}

func (s *DB) getSession(ctx context.Context, sess string) (*Session, error) {
	var session Session
	err := Get(s.conn, ctx, &session, `SELECT * FROM active_sessions WHERE id = $1`, sess)
	if err != nil {
		return nil, errors.New("Unauthed")
	}
	return &session, nil
}

func (s *DB) RemoveSession(ctx context.Context, sess string) error {
	_, err := s.conn.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sess)
	return err
}

var MaxSessionCount = config.GenFlag[int]("behavior.sessions.max_concurrent", 10, "Maximum number of sessions a user can have in total")

func (s *DB) RemoveOldSessions(ctx context.Context, userID int) (int, error) {
	tag, err := s.conn.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1 AND NOT (id = ANY(SELECT id FROM sessions WHERE user_id = $1 ORDER BY expires_at DESC LIMIT $2))`, userID, MaxSessionCount.Value())
	return int(tag.RowsAffected()), err
}

func (s *DB) ExtendSession(ctx context.Context, sid string) (time.Time, error) {
	sess, err := s.getSession(ctx, sid)
	if err != nil || sess == nil {
		return time.Time{}, err
	}
	var newExpiration time.Time
	err = s.conn.QueryRow(ctx, `UPDATE sessions SET expires_at = $2 WHERE id = $1 RETURNING expires_at`, sess.ID, time.Now().Add(sessionDuration)).Scan(&newExpiration)
	return newExpiration, err
}

func (s *DB) RemoveSessions(ctx context.Context, userID int) error {
	_, err := s.conn.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
