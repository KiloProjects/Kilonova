package db

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
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

type SessionDevice struct {
	SessID        string    `db:"session_id"`
	CreatedAt     time.Time `db:"created_at"`
	LastCheckedAt time.Time `db:"last_checked_at"`

	IPAddr    *netip.Addr `db:"ip_addr"`
	UserAgent *string     `db:"user_agent"`
	UserID    *int        `db:"user_id"`
}

type SessionFilter struct {
	ID         *string
	UserID     *int
	UserPrefix *string

	IPAddr   *netip.Addr
	IPPrefix *netip.Prefix

	Limit  int
	Offset int

	Ordering  string
	Ascending bool
}

func (sf *SessionFilter) filterQuery(fb *filterBuilder) {
	if v := sf.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := sf.UserID; v != nil {
		fb.AddConstraint("user_id = %s", v)
	}
	if v := sf.UserPrefix; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM users WHERE users.id = user_id AND name LIKE %s || '%%')", v)
	}

	if v := sf.IPAddr; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM session_clients WHERE session_id = id AND ip_addr = %s)", v)
	}
	if v := sf.IPPrefix; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM session_clients WHERE session_id = id AND ip_addr <<= %s)", v)
	}
}

func (sf *SessionFilter) ordering() string {
	ord := " DESC"
	if sf.Ascending {
		ord = " ASC"
	}
	switch sf.Ordering {
	case "last_access":
		return "ORDER BY (SELECT MAX(last_checked_at) FROM session_clients WHERE session_id = id)" + ord + " NULLS LAST"
	case "user_id":
		return "ORDER BY user_id" + ord
	default:
		return "ORDER BY created_at" + ord
	}
}

func (s *DB) Sessions(ctx context.Context, filter *SessionFilter) ([]*Session, error) {
	fb := newFilterBuilder()
	filter.filterQuery(fb)

	query := fmt.Sprintf("SELECT * FROM sessions WHERE %s %s %s", fb.Where(), filter.ordering(), FormatLimitOffset(filter.Limit, filter.Offset))
	rows, _ := s.conn.Query(ctx, query, fb.Args()...)
	sessions, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Session])
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return []*Session{}, nil
	}
	return sessions, err
}
func (s *DB) CountSessions(ctx context.Context, filter *SessionFilter) (int, error) {
	fb := newFilterBuilder()
	filter.filterQuery(fb)
	var val int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE "+fb.Where(), fb.Args()...).Scan(&val)
	return val, err
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
		return nil, errors.New("unauthed")
	}
	return &session, nil
}

func (s *DB) RemoveSession(ctx context.Context, sess string) error {
	_, err := s.conn.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sess)
	return err
}

var MaxSessionCount = config.GenFlag[int]("behavior.sessions.max_concurrent", 10, "Maximum number of sessions a user can have in total")

func (s *DB) RemoveOldSessions(ctx context.Context, userID int) ([]string, error) {
	q, _ := s.conn.Query(ctx, `DELETE FROM sessions WHERE user_id = $1 AND NOT (id = ANY(SELECT id FROM sessions WHERE user_id = $1 ORDER BY expires_at DESC LIMIT $2))`, userID, MaxSessionCount.Value())
	return pgx.CollectRows(q, pgx.RowTo[string])
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

func (s *DB) RemoveSessions(ctx context.Context, userID int) ([]string, error) {
	q, _ := s.conn.Query(ctx, `DELETE FROM sessions WHERE user_id = $1 RETURNING id`, userID)
	return pgx.CollectRows(q, pgx.RowTo[string])
}

func (s *DB) UpdateSessionDevice(ctx context.Context, sid string, uid int, ip *netip.Addr, userAgent *string) error {
	_, err := s.conn.Exec(ctx, `INSERT INTO session_clients (session_id, ip_addr, user_agent, user_id) VALUES ($1, $2, $3, $4) ON CONFLICT ON CONSTRAINT unique_client_tuple DO UPDATE SET last_checked_at = NOW()`, sid, ip, userAgent, uid)
	if err != nil && strings.Contains(err.Error(), "foreign key constraint") {
		return nil
	}
	return err
}

func (s *DB) SessionDevices(ctx context.Context, sid string) ([]*SessionDevice, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM session_clients WHERE session_id = $1 ORDER BY last_checked_at DESC", sid)
	devices, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[SessionDevice])
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return []*SessionDevice{}, nil
	}
	return devices, err
}
