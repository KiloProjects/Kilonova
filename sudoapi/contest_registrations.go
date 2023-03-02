package sudoapi

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RegisterContestUser(ctx context.Context, contest *kilonova.Contest, userID int) *StatusError {
	_, err := s.ContestRegistration(ctx, contest.ID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return WrapError(err, "User already registered")
	}

	if err := s.db.InsertContestRegistration(ctx, contest.ID, userID); err != nil {
		return WrapError(err, "Couldn't register user for contest")
	}
	return nil
}

func (s *BaseAPI) StartContestRegistration(ctx context.Context, contest *kilonova.Contest, userID int) *StatusError {
	if contest.PerUserTime == 0 {
		return Statusf(400, "Contest is not USACO-style")
	}

	reg, err := s.ContestRegistration(ctx, contest.ID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		if err := s.RegisterContestUser(ctx, contest, userID); err != nil {
			return err
		}
	}

	if reg.IndividualStartTime != nil {
		return Statusf(400, "User already started participation")
	}

	startTime := time.Now()
	endTime := startTime.Add(contest.PerUserTime * time.Second)
	if err := s.db.StartContestRegistration(ctx, contest.ID, userID, startTime, endTime); err != nil {
		return WrapError(err, "Couldn't start USACO-style contest participation")
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

func (s *BaseAPI) ContestRegistrationCount(ctx context.Context, contestID int) (int, *StatusError) {
	cnt, err := s.db.ContestRegistrationCount(ctx, contestID)
	if err != nil {
		return -1, WrapError(err, "Couldn't get registration count")
	}
	return cnt, nil
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
