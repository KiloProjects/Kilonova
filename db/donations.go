package db

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
)

type donation struct {
	ID        int       `db:"id"`
	DonatedAt time.Time `db:"donated_at"`
	UserID    *int      `db:"user_id"`
	Amount    float64   `db:"amount"`
	Currency  string    `db:"currency"`

	Source kilonova.DonationSource `db:"source"`
	Type   kilonova.DonationType   `db:"type"`

	TransactionID string     `db:"transaction_id"`
	CancelledAt   *time.Time `db:"cancelled_at"`
}

func (s *DB) AddDonation(ctx context.Context, donation *kilonova.Donation) error {
	var userID *int
	if donation.User != nil {
		userID = &donation.User.ID
	}
	var id int
	err := s.conn.QueryRow(ctx,
		"INSERT INTO donations (donated_at, user_id, amount, source, type, transaction_id, cancelled_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id",
		donation.DonatedAt, userID, donation.Amount, donation.Source, donation.Type, donation.TransactionID, donation.CancelledAt,
	).Scan(&id)
	if err == nil {
		donation.ID = id
	}
	return err
}

func (s *DB) Donations(ctx context.Context) ([]*kilonova.Donation, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM donations ORDER BY donated_at DESC")
	donations, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[donation])
	if err != nil {
		return nil, err
	}

	return mapperCtx(ctx, donations, s.internalToDonation), nil
}

func (s *DB) internalToDonation(ctx context.Context, d *donation) (*kilonova.Donation, error) {
	var user *kilonova.UserBrief
	if d.UserID != nil {
		user1, err := s.User(ctx, kilonova.UserFilter{ID: d.UserID})
		if err != nil {
			return nil, err
		}
		user = user1.ToBrief()
	}
	return &kilonova.Donation{
		ID:        d.ID,
		DonatedAt: d.DonatedAt,
		User:      user,
		Amount:    d.Amount,
		Currency:  d.Currency,

		Source:        d.Source,
		Type:          d.Type,
		TransactionID: d.TransactionID,
		CancelledAt:   d.CancelledAt,
	}, nil
}
