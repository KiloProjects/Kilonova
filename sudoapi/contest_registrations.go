package sudoapi

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RegisterContestUser(ctx context.Context, contestID, userID int) *StatusError {
	_, err := s.ContestRegistration(ctx, contestID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return WrapError(err, "Couldn't sanity check registration")
	}

	if err := s.db.InsertContestRegistration(ctx, contestID, userID); err != nil {
		return WrapError(err, "Couldn't register user for contest")
	}
	return nil
}

func (s *BaseAPI) ContestRegistrations(ctx context.Context, contestID, limit, offset int) ([]*kilonova.ContestRegistration, *StatusError) {
	regs, err := s.db.ContestRegistrations(ctx, contestID, limit, offset)
	if err != nil {
		return nil, WrapError(err, "Couldn't get registrations")
	}
	return regs, nil
}

func (s *BaseAPI) ContestRegistration(ctx context.Context, contestID, userID int) (*kilonova.ContestRegistration, *StatusError) {
	reg, err := s.db.ContestRegistration(ctx, contestID, userID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get registration")
	}
	if reg == nil {
		return nil, WrapError(ErrNotFound, "Registration not found")
	}

	return reg, nil
}
