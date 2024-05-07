package discord

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/bwmarrin/discordgo"
)

var (
	Token = config.GenFlag("integrations.discord.token", "", "Discord token for bot")
	// ProposerRoleID = config.GenFlag("integrations.discord.role_ids.proposer", "", "asdf")

	ErrNoToken = kilonova.Statusf(401, "No Discord token provided")
)

type Bot struct {
	base    *sudoapi.BaseAPI
	session *discordgo.Session
}

func (db *Bot) Close() error {
	return nil
}

func NewBot(ctx context.Context, base *sudoapi.BaseAPI) (*Bot, *kilonova.StatusError) {
	if len(Token.Value()) < 2 {
		return nil, ErrNoToken
	}
	discord, err := discordgo.New("Bot " + "adf")
	if err != nil {
		return nil, kilonova.WrapError(err, "Could not connect to discord")
	}

	return &Bot{base, discord}, nil
}
