package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func (s *DB) ContestRegistrations(ctx context.Context, contestID int, fuzzyName *string, inviteID *string, limit, offset uint64) ([]*kilonova.ContestRegistration, error) {
	qb := sq.Select("*").From("contest_registrations").Where(sq.Eq{"contest_id": contestID}).OrderBy("created_at ASC")
	if fuzzyName != nil {
		qb = qb.Where(sq.Expr("EXISTS (SELECT 1 FROM users WHERE id = user_id AND position(lower(unaccent(?)) in format('#%s %s', id, lower(unaccent(name)))) > 0)", fuzzyName))
	}
	if inviteID != nil {
		qb = qb.Where(sq.Eq{"invitation_id": inviteID})
	}
	qb = LimitOffset(qb, limit, offset)
	query, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, query, args...)
	regs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[kilonova.ContestRegistration])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*kilonova.ContestRegistration{}, nil
		}
		return nil, err
	}
	return regs, nil
}

func (s *DB) ContestRegistrationCount(ctx context.Context, contestID int) (int, error) {
	var cnt int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM contest_registrations WHERE contest_id = $1", contestID).Scan(&cnt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return -1, nil
		}
		return -1, err
	}
	return cnt, nil
}

func (s *DB) ContestRegistration(ctx context.Context, contestID, userID int) (*kilonova.ContestRegistration, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM contest_registrations WHERE contest_id = $1 AND user_id = $2 LIMIT 1", contestID, userID)
	reg, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[kilonova.ContestRegistration])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &reg, nil
}

func (s *DB) InsertContestRegistration(ctx context.Context, contestID, userID int, invitationID *string) error {
	_, err := s.conn.Exec(ctx, "INSERT INTO contest_registrations (user_id, contest_id, invitation_id) VALUES ($1, $2, $3)", userID, contestID, invitationID)
	return err
}

func (s *DB) StartContestRegistration(ctx context.Context, contestID, userID int, startTime time.Time, endTime time.Time) error {
	_, err := s.conn.Exec(ctx, "UPDATE contest_registrations SET individual_start_at = $1, individual_end_at = $2 WHERE contest_id = $3 AND user_id = $4", startTime, endTime, contestID, userID)
	return err
}

func (s *DB) DeleteContestRegistration(ctx context.Context, contestID, userID int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM contest_registrations WHERE user_id = $1 AND contest_id = $2", userID, contestID)
	return err
}

func (s *DB) ContestInvitations(ctx context.Context, contestID int) ([]*kilonova.ContestInvitation, error) {
	rows, _ := s.conn.Query(ctx, "SELECT *, (SELECT COUNT(*) FROM contest_registrations WHERE invitation_id = inv.id) AS redeem_cnt FROM contest_invitations inv WHERE contest_id = $1 ORDER BY expired ASC, created_at DESC", contestID)
	invitations, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[kilonova.ContestInvitation])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return invitations, nil
}

func (s *DB) ContestInvitation(ctx context.Context, id string) (*kilonova.ContestInvitation, error) {
	rows, _ := s.conn.Query(ctx, "SELECT *, (SELECT COUNT(*) FROM contest_registrations WHERE invitation_id = inv.id) AS redeem_cnt FROM contest_invitations inv WHERE id = $1 LIMIT 1", id)
	invitation, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[kilonova.ContestInvitation])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return invitation, nil
}

func (s *DB) UpdateContestInvitation(ctx context.Context, id string, expired bool) error {
	_, err := s.conn.Exec(ctx, "UPDATE contest_invitations SET expired = $2 WHERE id = $1", id, expired)
	return err
}

func (s *DB) CreateContestInvitation(ctx context.Context, contestID int, creatorID *int, maxUses *int) (string, error) {
	id := kilonova.RandomString(12)
	_, err := s.conn.Exec(ctx, "INSERT INTO contest_invitations (id, contest_id, creator_id, max_invitation_cnt) VALUES ($1, $2, $3, $4)", id, contestID, creatorID, maxUses)
	return id, err
}
