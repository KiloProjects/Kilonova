package sudoapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
)

var (
	DiscordEnabled = config.GenFlag("integrations.discord.enabled", false, "Enable Discord integration. If checked, you must provide client ID/secret and bot token.")

	Token = config.GenFlag("integrations.discord.token", "", "Discord token for bot")

	// ProposerRoleID = config.GenFlag("integrations.discord.role_ids.proposer", "", "asdf")

	DiscordClientID     = config.GenFlag("integrations.discord.client_id", "", "Discord Client ID")
	DiscordClientSecret = config.GenFlag("integrations.discord.client_secret", "", "Discord Client Secret")

	ProblemAnnouncementChannel = config.GenFlag("integrations.discord.publish_announcement_channel", "", "Discord channel to announce new problems")

	ErrDisconnected = Statusf(401, "Not connected to Discord")
)

func (s *BaseAPI) initDiscord(ctx context.Context) error {
	// if len(Token.Value()) < 2 {
	// 	return Statusf(406, "No Discord token provided")
	// }

	discord, err := discordgo.New("Bot " + Token.Value())
	if err != nil {
		return fmt.Errorf("could not connect to Discord: %w", err)
	}
	s.dSess = discord

	if DiscordEnabled.Value() {
		if err := s.dSess.Open(); err != nil {
			return fmt.Errorf("could not open gateway: %w", err)
		}
	} else {
		slog.InfoContext(ctx, "Initializing Discord session unauthed")
	}

	return nil
}

func (s *BaseAPI) AnnounceProblemPublished(ctx context.Context, problemID int) {
	slog.DebugContext(ctx, "Announcing problem publish", slog.Int("problem_id", problemID))
	if !DiscordEnabled.Value() || ProblemAnnouncementChannel.Value() == "" {
		return // noop
	}

	_, err := s.dSess.ChannelMessageSend(ProblemAnnouncementChannel.Value(), fmt.Sprintf("New problem was just published: %s/problems/%d", config.Common.HostPrefix, problemID))
	if err != nil {
		slog.WarnContext(ctx, "Could not announce problem publish", slog.Any("err", err))
	}
}

// If both user and error is nil, it means that a user doesn't have a Discord account attached (or that Discord integration is disabled)
// TODO: Cache output
func (s *BaseAPI) GetDiscordIdentity(ctx context.Context, userID int) (*discordgo.User, error) {
	if !DiscordEnabled.Value() {
		return nil, nil
	}
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	if user == nil || user.DiscordID == nil {
		return nil, nil
	}
	dUser, err := s.dSess.User(*user.DiscordID)
	if err != nil {
		return nil, fmt.Errorf("could not get Discord user: %w", err)
	}
	return dUser, nil
}

func (s *BaseAPI) UnlinkDiscordIdentity(ctx context.Context, userID int) error {
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &userID})
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
	if !DiscordEnabled.Value() {
		return "/", nil
	}
	st, err := s.db.CreateDiscordState(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("could not initialize Discord request: %w", err)
	}
	return s.discordConfig().AuthCodeURL(st), nil
}

// TODO: Reset discord connection

func (s *BaseAPI) HandleDiscordCallback(w http.ResponseWriter, r *http.Request) {
	uid, err := s.db.GetDiscordState(r.Context(), r.FormValue("state"))
	if err != nil || uid <= 0 {
		kilonova.StatusData(w, "error", "State does not match", http.StatusBadRequest)
		return
	}
	ctx := context.WithoutCancel(r.Context())
	defer s.db.RemoveDiscordState(ctx, r.FormValue("state"))

	conf := s.discordConfig()

	token, err := conf.Exchange(ctx, r.FormValue("code"))
	if err != nil {
		kilonova.StatusData(w, "error", "Could not get Discord token: "+err.Error(), 500)
		return
	}

	res, err := conf.Client(ctx, token).Get("https://discord.com/api/users/@me")
	if err != nil {
		kilonova.StatusData(w, "error", "Could not get user: "+err.Error(), 500)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		kilonova.StatusData(w, "error", res.Status, 500)
		return
	}

	// Should be able to unmarshal directly into discordgo user
	var dUser discordgo.User
	if err := json.NewDecoder(res.Body).Decode(&dUser); err != nil {
		kilonova.StatusData(w, "error", "Could not decode Discord response: "+err.Error(), 500)
		return
	}

	if err := s.updateUser(ctx, uid, kilonova.UserFullUpdate{
		SetDiscordID: true, DiscordID: &dUser.ID,
	}); err != nil {
		kilonova.StatusData(w, "error", err.Error(), kilonova.ErrorCode(err))
		return
	}

	userAttr := slog.Any("userID", uid)
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &uid})
	if err == nil && user != nil {
		userAttr = slog.Any("user", user.Brief())
	}
	s.LogVerbose(ctx, "User linked Discord identity", userAttr, slog.String("discord_id", dUser.ID), slog.String("discord_user", dUser.Mention()))

	http.Redirect(w, r, config.Common.HostPrefix+"/profile/linked", http.StatusTemporaryRedirect)
}

func (s *BaseAPI) discordConfig() *oauth2.Config {
	return &oauth2.Config{
		Endpoint: discordEndpoint,

		ClientID:     DiscordClientID.Value(),
		ClientSecret: DiscordClientSecret.Value(),

		Scopes: []string{"identify"},

		RedirectURL: config.Common.HostPrefix + "/api/webhook/discord_callback",
	}
}

var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",

	AuthStyle: oauth2.AuthStyleInParams,
}
