package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) AddDonation(ctx context.Context, donation *kilonova.Donation) *kilonova.StatusError {
	if err := s.db.AddDonation(ctx, donation); err != nil {
		return kilonova.WrapError(err, "Couldn't add donation")
	}
	return nil
}

func (s *BaseAPI) Donations(ctx context.Context) ([]*kilonova.Donation, *kilonova.StatusError) {
	donations, err := s.db.Donations(ctx)
	if err != nil {
		return nil, kilonova.WrapError(err, "Couldn't get donations")
	}
	return donations, nil
}
