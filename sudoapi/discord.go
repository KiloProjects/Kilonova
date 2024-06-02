package sudoapi

import (
	"log/slog"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/bwmarrin/discordgo"
)

var (
	Token = config.GenFlag("integrations.discord.token", "", "Discord token for bot")
	// ProposerRoleID = config.GenFlag("integrations.discord.role_ids.proposer", "", "asdf")

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

	if len(Token.Value()) > 2 {
		if err := s.dSess.Open(); err != nil {
			return WrapError(err, "Could not open gateway")
		}
	} else {
		slog.Info("Initializing Discord session unauthed")
	}

	return nil
}

// TODO: Reset discord connection

// TODO: Reimplement webhook send

// var discordEndpoint = &oauth2.Endpoint{
// 	AuthURL:  "https://discord.com/oauth2/authorize",
// 	TokenURL: "https://discord.com/api/oauth2/token",

// 	AuthStyle: oauth2.AuthStyleInParams,
// }
