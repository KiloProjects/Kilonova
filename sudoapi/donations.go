package sudoapi

import (
	"context"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) AddDonation(ctx context.Context, donation *kilonova.Donation) error {
	if err := s.db.AddDonation(ctx, donation); err != nil {
		return fmt.Errorf("Couldn't add donation: %w", err)
	}
	return nil
}

func (s *BaseAPI) CancelSubscription(ctx context.Context, id int) error {
	if err := s.db.CancelSubscription(ctx, id, time.Now()); err != nil {
		return fmt.Errorf("Couldn't mark subscription as cancelled: %w", err)
	}
	return nil
}

func (s *BaseAPI) Donations(ctx context.Context) ([]*kilonova.Donation, error) {
	donations, err := s.db.Donations(ctx)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get donations: %w", err)
	}
	return donations, nil
}
