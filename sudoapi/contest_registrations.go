package sudoapi

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) RegisterContestUser(ctx context.Context, contest *kilonova.Contest, userID int, invitationID *string, force bool) *StatusError {
	_, err := s.ContestRegistration(ctx, contest.ID, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return WrapError(err, "User already registered")
	}

	if !(force || s.CanJoinContest(contest) || invitationID != nil) {
		return Statusf(400, "Regular joining is disallowed")
	}

	if err := s.db.InsertContestRegistration(ctx, contest.ID, userID, invitationID); err != nil {
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
		return WrapError(err, "Couldn't start USACO-style contest participation")
	}
	return nil
}

func (s *BaseAPI) ContestRegistrations(ctx context.Context, contestID int, fuzzyName *string, inviteID *string, limit, offset int) ([]*kilonova.ContestRegistration, *StatusError) {
	regs, err := s.db.ContestRegistrations(ctx, contestID, fuzzyName, inviteID, limit, offset)
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

func (s *BaseAPI) KickUserFromContest(ctx context.Context, contestID, userID int) *StatusError {
	if err := s.db.DeleteContestRegistration(ctx, contestID, userID); err != nil {
		return WrapError(err, "Couldn't kick contestant")
	}
	if err := s.db.ClearUserContestSubmissions(ctx, contestID, userID); err != nil {
		return WrapError(err, "Couldn't reset contest submissions")
	}
	return nil
}

func (s *BaseAPI) CreateContestInvitation(ctx context.Context, contestID int, author *kilonova.UserBrief) (string, *StatusError) {
	var id *int
	if author != nil {
		id = &author.ID
	}
	invID, err := s.db.CreateContestInvitation(ctx, contestID, id)
	if err != nil {
		return "", WrapError(err, "Couldn't create invitation")
	}
	return invID, nil
}

func (s *BaseAPI) UpdateContestInvitation(ctx context.Context, id string, expired bool) *StatusError {
	if err := s.db.UpdateContestInvitation(ctx, id, expired); err != nil {
		return WrapError(err, "Couldn't update invitation status")
	}
	return nil
}

func (s *BaseAPI) ContestInvitations(ctx context.Context, contestID int) ([]*kilonova.ContestInvitation, *StatusError) {
	invitations, err := s.db.ContestInvitations(ctx, contestID)
	if err != nil {
		return nil, WrapError(err, "Couldn't get invitations")
	}
	return invitations, nil
}

func (s *BaseAPI) ContestInvitation(ctx context.Context, id string) (*kilonova.ContestInvitation, *StatusError) {
	inv, err := s.db.ContestInvitation(ctx, id)
	if err != nil {
		return nil, WrapError(err, "Couldn't get invitation")
	}
	if inv == nil {
		return nil, ErrNotFound
	}
	return inv, nil
}
