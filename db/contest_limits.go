package db

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/repository"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type contestLimitConfig struct {
	ContestID              int  `db:"contest_id"`
	IPManagementEnabled    bool `db:"ip_management_enabled"`
	WhitelistingEnabled    bool `db:"whitelisting_enabled"`
	PastSubmissionsEnabled bool `db:"past_submissions_enabled"`
}

func (s *DB) UpdateContestLimitConfig(ctx context.Context, contestID int, config *kilonova.ContestLimitConfigUpdate) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		// Upsert defaults, just to make sure row exists
		_, err := tx.Exec(ctx, `INSERT INTO contest_limits (contest_id) VALUES ($1) ON CONFLICT (contest_id) DO NOTHING`, contestID)
		if err != nil {
			return err
		}

		query := sq.Update("contest_limits").Where(sq.Eq{"contest_id": contestID})
		if v := config.PastSubmissionsEnabled; v != nil {
			query = query.Set("past_submissions_enabled", *v)
		}
		if v := config.WhitelistingEnabled; v != nil {
			query = query.Set("whitelisting_enabled", *v)
		}
		if v := config.IPManagementEnabled; v != nil {
			query = query.Set("ip_management_enabled", *v)
		}

		sql, args, err := query.ToSql()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, sql, args...)
		return err
	})
}

func (s *DB) ContestLimitConfig(ctx context.Context, contestID int) (*kilonova.ContestLimitConfig, error) {
	query := sq.Select("*").From("contest_limits").Where(sq.Eq{"contest_id": contestID})
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	rows, _ := s.conn.Query(ctx, sql, args...)
	config, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[contestLimitConfig])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// If not found, assume default
			return &kilonova.ContestLimitConfig{
				ContestID:              contestID,
				IPManagementEnabled:    false,
				WhitelistingEnabled:    false,
				PastSubmissionsEnabled: true,
			}, nil
		}
	}
	return &kilonova.ContestLimitConfig{
		ContestID:              config.ContestID,
		IPManagementEnabled:    config.IPManagementEnabled,
		WhitelistingEnabled:    config.WhitelistingEnabled,
		PastSubmissionsEnabled: config.PastSubmissionsEnabled,
	}, nil
}

type userContestIPs struct {
	*repository.User
	IPs []*netip.Addr `db:"ips"`
}

// ContestIPs returns the list of IPs that were used by users participating in the contest.
// Note that it returns the IPs over the ENTIRE given interval, disregarding USACO start times.
func (s *DB) ContestIPs(ctx context.Context, contestID int, startTime, endTime time.Time) ([]*kilonova.ContestUserIPs, error) {
	ipQuery := sq.Select("user_id", "array_agg(ip_addr ORDER BY last_checked_at DESC) AS ip_addr").
		From("session_clients").
		Where(
			sq.And{
				sq.Expr(
					"tsrange(created_at, last_checked_at) && tsrange(?, ?)",
					startTime, endTime,
				),
				sq.Expr("EXISTS (SELECT 1 FROM contest_registrations WHERE contest_registrations.user_id = session_clients.user_id AND contest_registrations.contest_id = ?)", contestID),
			},
		).GroupBy("user_id")
	sql, args, err := ipQuery.ToSql()
	if err != nil {
		return nil, err
	}
	rows, _ := s.conn.Query(ctx, sql, args...)
	vals, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[userContestIPs])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ips := make([]*kilonova.ContestUserIPs, len(vals))
	for i, val := range vals {
		ips[i] = &kilonova.ContestUserIPs{
			User: val.User.Brief(),
			IPs:  val.IPs,
		}
	}
	return ips, nil
}
