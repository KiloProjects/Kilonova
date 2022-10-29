package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

type auditLog struct {
	ID        int       `db:"id"`
	LogTime   time.Time `db:"logged_at"`
	SystemLog bool      `db:"system_log"`
	Message   string    `db:"msg"`
	AuthorID  *int      `db:"author_id"`
}

const auditLogCreateQuery = `INSERT INTO audit_logs (
	system_log, msg, author_id
) VALUES (
	?, ?, ?
) RETURNING id;`

func (s *DB) CreateAuditLog(ctx context.Context, msg string, authorID *int, system bool) (int, error) {
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(auditLogCreateQuery), system, msg, authorID)
	return id, err
}

func (s *DB) AuditLogs(ctx context.Context, limit, offset int) ([]*kilonova.AuditLog, error) {
	var logs []*auditLog
	query := s.conn.Rebind("SELECT * FROM audit_logs ORDER BY logged_at DESC " + FormatLimitOffset(limit, offset))
	err := s.conn.SelectContext(ctx, &logs, query)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.AuditLog{}, nil
	} else if err != nil {
		return nil, err
	}

	var realLogs []*kilonova.AuditLog
	for _, log := range logs {
		realLog, err := s.internalToAuditLog(ctx, log)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		realLogs = append(realLogs, realLog)
	}
	return realLogs, nil
}

func (s *DB) AuditLogCount(ctx context.Context) (int, error) {
	var cnt int
	err := s.conn.GetContext(ctx, &cnt, "SELECT COUNT(id) FROM audit_logs")
	return cnt, err
}

func (s *DB) internalToAuditLog(ctx context.Context, a *auditLog) (*kilonova.AuditLog, error) {
	if a == nil {
		return nil, nil
	}
	var author *kilonova.UserBrief
	if a.AuthorID != nil {
		fullAuthor, err := s.User(ctx, *a.AuthorID)
		if err != nil {
			return nil, err
		}
		author = fullAuthor.ToBrief()
	}

	return &kilonova.AuditLog{
		ID:        a.ID,
		LogTime:   a.LogTime,
		SystemLog: a.SystemLog,
		Message:   a.Message,
		Author:    author,
	}, nil
}
