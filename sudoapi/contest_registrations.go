package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RegisterContestUser(ctx context.Context, contest *kilonova.Contest, userID int, invitationID *string, force bool) error {
	_, err := s.ContestRegistration(ctx, contest.ID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("user already registered: %w", err)
	}

	if !(force || s.CanJoinContest(contest) || invitationID != nil) {
		return Statusf(400, "Regular joining is disallowed")
	}

	if err := s.db.InsertContestRegistration(ctx, contest.ID, userID, invitationID); err != nil {
		return fmt.Errorf("couldn't register user for contest: %w", err)
	}
	return nil
}

func (s *BaseAPI) StartContestRegistration(ctx context.Context, contest *kilonova.Contest, userID int) error {
	if contest.PerUserTime == 0 {
		return Statusf(400, "Contest is not USACO-style")
	}

	reg, err := s.ContestRegistration(ctx, contest.ID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		if err := s.RegisterContestUser(ctx, contest, userID, nil, false); err != nil {
			return err
		}
	}

	if reg.IndividualStartTime != nil {
		return Statusf(400, "User already started participation")
	}

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(contest.PerUserTime) * time.Second)
	if err := s.db.StartContestRegistration(ctx, contest.ID, userID, startTime, endTime); err != nil {
		return fmt.Errorf("couldn't start USACO-style contest participation: %w", err)
	}
	return nil
}

func (s *BaseAPI) ContestRegistrations(ctx context.Context, contestID int, fuzzyName *string, inviteID *string, limit, offset uint64) ([]*kilonova.ContestRegistration, error) {
	regs, err := s.db.ContestRegistrations(ctx, contestID, fuzzyName, inviteID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("couldn't get registrations: %w", err)
	}
	return regs, nil
}

func (s *BaseAPI) ContestRegistrationCount(ctx context.Context, contestID int) (int, error) {
	cnt, err := s.db.ContestRegistrationCount(ctx, contestID)
	if err != nil {
		return -1, fmt.Errorf("couldn't get registration count: %w", err)
	}
	return cnt, nil
}

func (s *BaseAPI) ContestRegistration(ctx context.Context, contestID, userID int) (*kilonova.ContestRegistration, error) {
	reg, err := s.db.ContestRegistration(ctx, contestID, userID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get registration: %w", err)
	}
	if reg == nil {
		return nil, fmt.Errorf("registration not found: %w", ErrNotFound)
	}

	return reg, nil
}

func (s *BaseAPI) KickUserFromContest(ctx context.Context, contestID, userID int) error {
	if err := s.db.DeleteContestRegistration(ctx, contestID, userID); err != nil {
		return fmt.Errorf("couldn't kick contestant: %w", err)
	}
	if err := s.db.ClearUserContestSubmissions(ctx, contestID, userID); err != nil {
		return fmt.Errorf("couldn't reset contest submissions: %w", err)
	}
	return nil
}

func (s *BaseAPI) CreateContestInvitation(ctx context.Context, contestID int, author *kilonova.UserBrief, maxUses *int) (string, error) {
	var id *int
	if author != nil {
		id = &author.ID
	}
	invID, err := s.db.CreateContestInvitation(ctx, contestID, id, maxUses)
	if err != nil {
		return "", fmt.Errorf("couldn't create invitation: %w", err)
	}
	return invID, nil
}

func (s *BaseAPI) UpdateContestInvitation(ctx context.Context, id string, expired bool) error {
	if err := s.db.UpdateContestInvitation(ctx, id, expired); err != nil {
		return fmt.Errorf("couldn't update invitation status: %w", err)
	}
	return nil
}

func (s *BaseAPI) ContestInvitations(ctx context.Context, contestID int) ([]*kilonova.ContestInvitation, error) {
	invitations, err := s.db.ContestInvitations(ctx, contestID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get invitations: %w", err)
	}
	return invitations, nil
}

func (s *BaseAPI) ContestInvitation(ctx context.Context, id string) (*kilonova.ContestInvitation, error) {
	inv, err := s.db.ContestInvitation(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't get invitation: %w", err)
	}
	if inv == nil {
		return nil, ErrNotFound
	}
	return inv, nil
}
