package sudoapi

import (
	"context"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) AddDonation(ctx context.Context, donation *kilonova.Donation) error {
	if err := s.db.AddDonation(ctx, donation); err != nil {
		return kilonova.WrapError(err, "Couldn't add donation")
	}
	return nil
}

func (s *BaseAPI) CancelSubscription(ctx context.Context, id int) error {
	if err := s.db.CancelSubscription(ctx, id, time.Now()); err != nil {
		return kilonova.WrapError(err, "Couldn't mark subscription as cancelled")
	}
	return nil
}

func (s *BaseAPI) Donations(ctx context.Context) ([]*kilonova.Donation, error) {
	donations, err := s.db.Donations(ctx)
	if err != nil {
		return nil, kilonova.WrapError(err, "Couldn't get donations")
	}
	return donations, nil
}
