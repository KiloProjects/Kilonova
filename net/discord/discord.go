// Package discord wraps the bot session and OAuth2 identity linking behind a
// Provider. A Provider is always constructed: webhook logging works with no bot
// token, and Enabled reports whether the bot-token features are available.
package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/oauth2"
)

type Config struct {
	Token        string // bot token; empty disables bot features
	ClientID     string // OAuth2 application, for linking user identities
	ClientSecret string
	RedirectURL  string // OAuth2 callback URL on this instance
}

type Provider interface {
	// Enabled reports whether a bot token is configured.
	Enabled() bool
	// Open connects the bot gateway (presence); a no-op when not Enabled.
	Open() error
	Close() error

	User(discordID string) (*discordgo.User, error)
	SendChannelMessage(channelID, content string) error

	// AuthCodeURL starts the OAuth2 identity-link flow for the given state.
	AuthCodeURL(state string) string
	// ExchangeIdentity completes it, returning the Discord user for the code.
	ExchangeIdentity(ctx context.Context, code string) (*discordgo.User, error)

	ExecuteWebhook(webhookID, webhookToken string, params *discordgo.WebhookParams) (*discordgo.Message, error)
	EditWebhookMessage(webhookID, webhookToken, messageID string, edit *discordgo.WebhookEdit) error
}

type provider struct {
	sess    *discordgo.Session
	enabled bool
	oauth   *oauth2.Config
}

var endpoint = oauth2.Endpoint{
	AuthURL:   "https://discord.com/oauth2/authorize",
	TokenURL:  "https://discord.com/api/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

func New(cfg Config) (Provider, error) {
	sess, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("could not create Discord session: %w", err)
	}
	return &provider{
		sess:    sess,
		enabled: cfg.Token != "",
		oauth: &oauth2.Config{
			Endpoint:     endpoint,
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes:       []string{"identify"},
			RedirectURL:  cfg.RedirectURL,
		},
	}, nil
}

func (p *provider) Enabled() bool { return p.enabled }

func (p *provider) Open() error {
	if !p.enabled {
		return nil
	}
	if err := p.sess.Open(); err != nil {
		return fmt.Errorf("could not open Discord gateway: %w", err)
	}
	return nil
}

func (p *provider) Close() error { return p.sess.Close() }

func (p *provider) User(discordID string) (*discordgo.User, error) {
	return p.sess.User(discordID)
}

func (p *provider) SendChannelMessage(channelID, content string) error {
	_, err := p.sess.ChannelMessageSend(channelID, content)
	return err
}

func (p *provider) AuthCodeURL(state string) string { return p.oauth.AuthCodeURL(state) }

func (p *provider) ExchangeIdentity(ctx context.Context, code string) (*discordgo.User, error) {
	token, err := p.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("could not get Discord token: %w", err)
	}
	res, err := p.oauth.Client(ctx, token).Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get user: %s", res.Status)
	}
	var user discordgo.User
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("could not decode Discord response: %w", err)
	}
	return &user, nil
}

func (p *provider) ExecuteWebhook(webhookID, webhookToken string, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return p.sess.WebhookExecute(webhookID, webhookToken, true, params)
}

func (p *provider) EditWebhookMessage(webhookID, webhookToken, messageID string, edit *discordgo.WebhookEdit) error {
	_, err := p.sess.WebhookMessageEdit(webhookID, webhookToken, messageID, edit)
	return err
}
