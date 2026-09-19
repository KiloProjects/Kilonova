package sudoapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/bwmarrin/discordgo"
)

func (s *BaseAPI) AnnounceProblemPublished(ctx context.Context, problemID int) {
	slog.DebugContext(ctx, "Announcing problem publish", slog.Int("problem_id", problemID))
	if !s.discord.Enabled() || flags.ProblemAnnouncementChannel.Value() == "" {
		return // noop
	}
	url := kilonova.HostURL().JoinPath("problems", strconv.Itoa(problemID)).String()
	if err := s.discord.SendChannelMessage(flags.ProblemAnnouncementChannel.Value(), "New problem was just published: "+url); err != nil {
		slog.WarnContext(ctx, "Could not announce problem publish", slog.Any("err", err))
	}
}

func (s *BaseAPI) AnnounceProblemReviewRequested(ctx context.Context, problemID int, requestedBy *kilonova.UserBrief) {
	slog.DebugContext(ctx, "Announcing problem request", slog.Int("problem_id", problemID))
	problem, err := s.Problem(ctx, problemID)
	if err != nil {
		s.LogUserAction(ctx, "Requested problem review (could not fetch problem)", slog.Int("problem_id", problemID), slog.Any("requested_by", requestedBy), slog.Any("error", err))
	} else {
		s.LogUserAction(ctx, "Requested problem review", slog.Any("problem", problem), slog.Any("requested_by", requestedBy))
	}
}

// If both user and error is nil, it means that a user doesn't have a Discord account attached (or that Discord integration is disabled)
// TODO: Cache output
func (s *BaseAPI) GetDiscordIdentity(ctx context.Context, userID int) (*discordgo.User, error) {
	if !s.discord.Enabled() {
		return nil, nil
	}
	user, err := s.userRepo.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	if user == nil || user.DiscordID == nil {
		return nil, nil
	}
	dUser, err := s.discord.User(*user.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("could not get Discord user: %w", err)
	}
	return dUser, nil
}

func (s *BaseAPI) UnlinkDiscordIdentity(ctx context.Context, userID int) error {
	user, err := s.userRepo.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil || user == nil {
		return fmt.Errorf("could not get user: %w", err)
	}
	if user.DiscordID == nil {
		return Statusf(400, "User has no linked Discord identity")
	}
	s.LogVerbose(ctx, "User tried to unlink Discord identity", slog.Any("user", user.Brief()), slog.String("discord_id", *user.DiscordID))
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{SetDiscordID: true, DiscordID: nil})
}

func (s *BaseAPI) DiscordAuthURL(ctx context.Context, userID int) (string, error) {
	if !s.discord.Enabled() {
		return "/", nil
	}
	st, err := s.db.CreateDiscordState(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("could not initialize Discord request: %w", err)
	}
	return s.discord.AuthCodeURL(st), nil
}

func (s *BaseAPI) HandleDiscordCallback(w http.ResponseWriter, r *http.Request) {
	uid, err := s.db.GetDiscordState(r.Context(), r.FormValue("state"))
	if err != nil || uid <= 0 {
		kilonova.StatusData(w, "error", "State does not match", http.StatusBadRequest)
		return
	}
	ctx := context.WithoutCancel(r.Context())
	defer s.db.RemoveDiscordState(ctx, r.FormValue("state"))

	dUser, err := s.discord.ExchangeIdentity(ctx, r.FormValue("code"))
	if err != nil {
		kilonova.StatusData(w, "error", err.Error(), 500)
		return
	}

	if err := s.updateUser(ctx, uid, kilonova.UserFullUpdate{
		SetDiscordID: true, DiscordID: &dUser.ID,
	}); err != nil {
		kilonova.StatusData(w, "error", err.Error(), kilonova.ErrorCode(err))
		return
	}

	userAttr := slog.Any("userID", uid)
	user, err := s.userRepo.User(ctx, kilonova.UserFilter{ID: &uid})
	if err == nil && user != nil {
		userAttr = slog.Any("user", user.Brief())
	}
	s.LogVerbose(ctx, "User linked Discord identity", userAttr, slog.String("discord_id", dUser.ID), slog.String("discord_user", dUser.Mention()))

	http.Redirect(w, r, kilonova.HostURL().JoinPath("profile/linked").String(), http.StatusTemporaryRedirect)
}
