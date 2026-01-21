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

type userContestIPs struct {
	*repository.User
	IPs []*netip.Addr `db:"ips"`
}

// ContestIPs returns the list of IPs that were used by users participating in the contest.
// Note that it returns the IPs over the ENTIRE given interval, disregarding USACO start times.
// TODO: ContestIPs should be called with a startTime of 1 hour before the contest, to match ContestFilter behaviour
func (s *DB) ContestIPs(ctx context.Context, contestID int, startTime, endTime time.Time) ([]*kilonova.ContestUserIPs, error) {
	ipQuery := sq.Select("user_id", "array_agg(ip_addr ORDER BY last_checked_at DESC) AS ip_addr").
		From("session_clients").
		Where(
			sq.And{
				sq.Expr(
					"tstzrange(created_at, last_checked_at, '[]') && tstzrange(?, ?, '[]')",
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
