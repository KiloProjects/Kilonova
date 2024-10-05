package sudoapi

import (
	"context"
	"encoding/json"
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

	ErrDisconnected = Statusf(401, "Not connected to Discord")
)

func (s *BaseAPI) initDiscord() *kilonova.StatusError {
	// if len(Token.Value()) < 2 {
	// 	return Statusf(406, "No Discord token provided")
	// }

	discord, err := discordgo.New("Bot " + Token.Value())
	if err != nil {
		return WrapError(err, "Could not connect to Discord")
	}
	s.dSess = discord

	if DiscordEnabled.Value() {
		if err := s.dSess.Open(); err != nil {
			return WrapError(err, "Could not open gateway")
		}
	} else {
		slog.Info("Initializing Discord session unauthed")
	}

	return nil
}

// If both user and error is nil, it means that a user doesn't have a Discord account attached (or that Discord integration is disabled)
// TODO: Cache output
func (s *BaseAPI) GetDiscordIdentity(ctx context.Context, userID int) (*discordgo.User, *StatusError) {
	if !DiscordEnabled.Value() {
		return nil, nil
	}
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil {
		return nil, WrapError(err, "Could not get user")
	}
	if user == nil || user.DiscordID == nil {
		return nil, nil
	}
	dUser, err := s.dSess.User(*user.DiscordID)
	if err != nil {
		return nil, WrapError(err, "Could not get Discord user")
	}
	return dUser, nil
}

func (s *BaseAPI) UnlinkDiscordIdentity(ctx context.Context, userID int) *StatusError {
	user, err := s.db.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil || user == nil {
		return WrapError(err, "Could not get user")
	}
	if user.DiscordID == nil {
		return Statusf(400, "User has no linked Discord identity")
	}
	s.LogVerbose(ctx, "User tried to unlink Discord identity", slog.Any("user", user.ToBrief()), slog.String("discord_id", *user.DiscordID))
	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{SetDiscordID: true, DiscordID: nil})
}

func (s *BaseAPI) DiscordAuthURL(ctx context.Context, userID int) (string, *StatusError) {
	if !DiscordEnabled.Value() {
		return "/", nil
	}
	st, err := s.db.CreateDiscordState(ctx, userID)
	if err != nil {
		return "", WrapError(err, "Could not initialize Discord request")
	}
	return s.discordConfig().AuthCodeURL(st), nil
}

// TODO: Reset discord connection

func (s *BaseAPI) HandleDiscordCallback(w http.ResponseWriter, r *http.Request) {
	uid, err := s.db.GetDiscordState(r.Context(), r.FormValue("state"))
	if err != nil || uid <= 0 {
		Statusf(http.StatusBadRequest, "State does not match").WriteError(w)
		return
	}
	defer s.db.RemoveDiscordState(context.Background(), r.FormValue("state"))

	conf := s.discordConfig()

	token, err := conf.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		WrapError(err, "Could not get Discord token").WriteError(w)
		return
	}

	res, err := conf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")
	if err != nil {
		WrapError(err, "Could not get user").WriteError(w)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		Statusf(500, "%s", res.Status).WriteError(w)
		return
	}

	// Should be able to unmarshal directly into discordgo user
	var dUser discordgo.User
	if err := json.NewDecoder(res.Body).Decode(&dUser); err != nil {
		WrapError(err, "Could not decode Discord response").WriteError(w)
	}

	if err := s.updateUser(r.Context(), uid, kilonova.UserFullUpdate{
		SetDiscordID: true, DiscordID: &dUser.ID,
	}); err != nil {
		err.WriteError(w)
		return
	}

	userAttr := slog.Any("userID", uid)
	user, err := s.db.User(r.Context(), kilonova.UserFilter{ID: &uid})
	if err == nil && user != nil {
		userAttr = slog.Any("user", user.ToBrief())
	}
	s.LogVerbose(r.Context(), "User linked Discord identity", userAttr, slog.String("discord_id", dUser.ID), slog.String("discord_user", dUser.Mention()))

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
