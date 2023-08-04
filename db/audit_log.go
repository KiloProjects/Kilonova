package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
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
	$1, $2, $3
) RETURNING id;`

func (s *DB) CreateAuditLog(ctx context.Context, msg string, authorID *int, system bool) (int, error) {
	var id int
	err := s.conn.QueryRow(ctx, auditLogCreateQuery, system, strings.TrimSpace(msg), authorID).Scan(&id)
	return id, err
}

func (s *DB) AuditLogs(ctx context.Context, limit, offset int) ([]*kilonova.AuditLog, error) {
	rows, err := s.conn.Query(ctx, "SELECT * FROM audit_logs ORDER BY logged_at DESC, id DESC "+FormatLimitOffset(limit, offset))
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.AuditLog{}, nil
	} else if err != nil {
		return nil, err
	}

	logs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[auditLog])
	if err != nil {
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
	err := s.conn.QueryRow(ctx, "SELECT COUNT(id) FROM audit_logs").Scan(&cnt)
	return cnt, err
}

func (s *DB) internalToAuditLog(ctx context.Context, a *auditLog) (*kilonova.AuditLog, error) {
	if a == nil {
		return nil, nil
	}
	var author *kilonova.UserBrief
	if a.AuthorID != nil {
		fullAuthor, err := s.User(ctx, kilonova.UserFilter{ID: a.AuthorID})
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
