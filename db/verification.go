package db

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
)

var _ kilonova.Verificationer = &VerificationService{}

type VerificationService struct {
	db *sqlx.DB
}

func (v *VerificationService) CreateVerification(ctx context.Context, id int) (string, error) {
	vid := kilonova.RandomString(16)
	_, err := v.db.ExecContext(ctx, v.db.Rebind(`INSERT INTO verifications (id, user_id) VALUES (?, ?)`), vid, id)
	return vid, err
}

func (v *VerificationService) GetVerification(ctx context.Context, id string) (int, error) {
	var verif struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UserID    int       `db:"user_id"`
	}
	err := v.db.GetContext(ctx, &verif, v.db.Rebind(`SELECT * FROM verifications WHERE id = ?`), id)
	if err != nil {
		return -1, err
	}
	if time.Now().Sub(verif.CreatedAt) > time.Hour*24*30 {
		return -1, err
	}
	return verif.UserID, err
}

func (v *VerificationService) RemoveVerification(ctx context.Context, verif string) error {
	_, err := v.db.ExecContext(ctx, v.db.Rebind(`DELETE FROM verifications WHERE id = ?`), verif)
	return err
}

func NewVerificationService(db *sqlx.DB) kilonova.Verificationer {
	return &VerificationService{db}
}
