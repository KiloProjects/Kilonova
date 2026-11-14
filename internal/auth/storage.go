package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"uuid"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user/userpg"
	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/go-jose/go-jose/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/language"
)

const (
	accessTokenLifetime  = 5 * time.Minute
	refreshTokenLifetime = 48 * time.Hour
)

// Lifetimes of auth requests, as SQL intervals: how long the user has to log in and approve,
// and how long the client then has to exchange the code.
const (
	authRequestLifetime = "1 hour"
	authCodeLifetime    = "10 minutes"
)

var _ op.Storage = (*AuthStorage)(nil)

// Token exchange, device authorization and client credentials are not implemented;
// the provider turns those grants off for a storage without the matching interfaces.

type AuthStorage struct {
	conn     *pgxpool.Pool
	key      *signingKey
	userRepo *userpg.Repository

	defaultLanguage func(ctx context.Context) string
	hostURL         *url.URL
}

func (s *AuthStorage) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
	if len(authReq.Prompt) == 1 && authReq.Prompt[0] == "none" {
		return nil, oidc.ErrLoginRequired()
	}

	appID, err := uuid.Parse(authReq.ClientID)
	if err != nil {
		return nil, err
	}

	req := &Request{
		ID:            uuid.NewV7(),
		ApplicationID: appID,
		CallbackURI:   authReq.RedirectURI,
		TransferState: authReq.State,

		Scopes:              authReq.Scopes,
		ResponseType:        authReq.ResponseType,
		ResponseMode:        authReq.ResponseMode,
		Nonce:               authReq.Nonce,
		CodeChallenge:       authReq.CodeChallenge,
		CodeChallengeMethod: authReq.CodeChallengeMethod,

		Prompt:    promptToInternal(authReq.Prompt),
		UILocales: authReq.UILocales,
		LoginHint: authReq.LoginHint,
		ExpiresAt: maxAgeToInternal(time.Now(), authReq.MaxAge),
	}

	if userID != "" {
		uid, err := strconv.Atoi(userID)
		if err != nil {
			return nil, err
		}
		req.UserID = &uid
	}

	if err := s.conn.QueryRow(ctx, `
		INSERT INTO oauth_requests (id, application_id, callback_uri, transfer_state, user_id, scopes, response_type, response_mode, nonce, code_challenge, code_challenge_method, prompt, ui_locales, login_hint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at
	`, req.ID, req.ApplicationID, req.CallbackURI, req.TransferState, req.UserID, req.Scopes, req.ResponseType, req.ResponseMode, req.Nonce, req.CodeChallenge, req.CodeChallengeMethod, req.Prompt, req.UILocales, req.LoginHint, req.ExpiresAt).Scan(&req.CreatedAt); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *AuthStorage) AuthRequestByID(ctx context.Context, reqID string) (op.AuthRequest, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_requests WHERE id = $1 AND created_at > NOW() - $2::interval", reqID, authRequestLifetime)
	val, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Request])
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *AuthStorage) AuthRequestByCode(ctx context.Context, reqCode string) (op.AuthRequest, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_requests WHERE request_code = $1 AND auth_time > NOW() - $2::interval", reqCode, authCodeLifetime)
	val, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Request])
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *AuthStorage) SaveAuthCode(ctx context.Context, reqID string, code string) error {
	_, err := s.conn.Exec(ctx, "UPDATE oauth_requests SET request_code = $1 WHERE id = $2", code, reqID)
	return err
}

func (s *AuthStorage) DeleteAuthRequest(ctx context.Context, reqID string) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_requests WHERE id = $1", reqID)
	return err
}

func (s *AuthStorage) getAccessToken(ctx context.Context, tokenID uuid.UUID) (*Token, error) {
	return s.getToken(ctx, tokenID, TokenTypeAccess)
}

func (s *AuthStorage) getRefreshToken(ctx context.Context, tokenID uuid.UUID) (*Token, error) {
	return s.getToken(ctx, tokenID, TokenTypeRefresh)
}

// getToken only returns live tokens. Expiry is written and compared with the database clock (see insertToken)
func (s *AuthStorage) getToken(ctx context.Context, tokenID uuid.UUID, tokenType TokenType) (*Token, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_tokens WHERE id = $1 AND token_type = $2 AND expires_at > NOW()", tokenID, tokenType)
	return pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Token])
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// insertToken stores a token that expires after lifetime. Both timestamps come from the
// database clock, since the columns have no time zone and are compared against NOW().
func insertToken(ctx context.Context, db execer, t *Token, lifetime time.Duration) error {
	_, err := db.Exec(ctx, `
		INSERT INTO oauth_tokens (id, expires_at, application_id, from_token, user_id, audience, scopes, token_type, amr, auth_time)
		VALUES ($1, NOW() + make_interval(secs => $2), $3, $4, $5, $6, $7, $8, $9, $10)
	`, t.ID, lifetime.Seconds(), t.ApplicationID, t.ParentToken, t.UserID, t.Audience, t.Scopes, t.TokenType, t.AMR, t.AuthTime)
	return err
}

// CreateAccessToken issues a lone access token for an approved auth request (code flow without offline_access)
func (s *AuthStorage) CreateAccessToken(ctx context.Context, request op.TokenRequest) (string, time.Time, error) {
	req, ok := request.(*Request)
	if !ok {
		return "", time.Time{}, fmt.Errorf("unsupported token request %T", request)
	}
	token := &Token{
		ID:            uuid.NewV7(),
		ApplicationID: req.ApplicationID,
		UserID:        req.UserID,
		Audience:      req.GetAudience(),
		Scopes:        req.GetScopes(),
		TokenType:     TokenTypeAccess,
	}
	if err := insertToken(ctx, s.conn, token, accessTokenLifetime); err != nil {
		return "", time.Time{}, err
	}
	return token.ID.String(), time.Now().Add(accessTokenLifetime), nil
}

// CreateAccessAndRefreshTokens issues an access token together with the refresh token it hangs off:
// the access token points at the refresh token through from_token, so deleting a refresh token
// (rotation, revocation) takes its access tokens with it.
//
// The request is either an approved *Request (code flow with offline_access, currentRefreshToken empty)
// or a *RefreshTokenRequest, in which case currentRefreshToken is rotated out.
// The returned expiration is the access token's.
func (s *AuthStorage) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshTokenID string, expiration time.Time, err error) {
	refresh := &Token{ID: uuid.NewV7(), TokenType: TokenTypeRefresh, Audience: request.GetAudience()}
	switch req := request.(type) {
	case *Request:
		refresh.ApplicationID, refresh.UserID, refresh.Scopes = req.ApplicationID, req.UserID, req.Scopes
		refresh.AMR, refresh.AuthTime = req.GetAMR(), req.AuthTime
	case *RefreshTokenRequest:
		// The new refresh token keeps the original grant, even if this request narrowed the scopes
		refresh.ApplicationID, refresh.UserID, refresh.Scopes = req.ApplicationID, req.UserID, req.Scopes
		refresh.AMR, refresh.AuthTime = req.AMR, req.AuthTime
	default:
		return "", "", time.Time{}, fmt.Errorf("unsupported token request %T", request)
	}
	access := &Token{
		ID:            uuid.NewV7(),
		ApplicationID: refresh.ApplicationID,
		ParentToken:   &refresh.ID,
		UserID:        refresh.UserID,
		Audience:      refresh.Audience,
		Scopes:        request.GetScopes(),
		TokenType:     TokenTypeAccess,
	}

	err = pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		if currentRefreshToken != "" {
			oldID, err := uuid.Parse(currentRefreshToken)
			if err != nil {
				return op.ErrInvalidRefreshToken
			}
			// Deleting doubles as the reuse check: of two concurrent refreshes, one finds no row
			tag, err := tx.Exec(ctx, "DELETE FROM oauth_tokens WHERE id = $1 AND token_type = $2 AND expires_at > NOW()", oldID, TokenTypeRefresh)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return op.ErrInvalidRefreshToken
			}
		}
		if err := insertToken(ctx, tx, refresh, refreshTokenLifetime); err != nil {
			return err
		}
		return insertToken(ctx, tx, access, accessTokenLifetime)
	})
	if err != nil {
		return "", "", time.Time{}, err
	}
	return access.ID.String(), refresh.ID.String(), time.Now().Add(accessTokenLifetime), nil
}

func (s *AuthStorage) TokenRequestByRefreshToken(ctx context.Context, refreshTokenID string) (op.RefreshTokenRequest, error) {
	val, err := uuid.Parse(refreshTokenID)
	if err != nil {
		return nil, err
	}
	token, err := s.getRefreshToken(ctx, val)
	if err != nil {
		return nil, err
	}
	return &RefreshTokenRequest{Token: token}, nil
}

func (s *AuthStorage) TerminateSession(ctx context.Context, userID string, clientID string) error {
	userIDInt, err := strconv.Atoi(userID)
	if err != nil || clientID == "" {
		return oidc.ErrInvalidRequest()
	}

	_, err = s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE user_id = $1 AND application_id = $2", userIDInt, clientID)
	return err
}

// GetRefreshTokenInfo must return ErrInvalidRefreshToken when presented
// with a token that is not a refresh token.
func (s *AuthStorage) GetRefreshTokenInfo(ctx context.Context, clientID string, token string) (userID string, tokenID string, err error) {
	val, err := uuid.Parse(token)
	if err != nil {
		return "", "", op.ErrInvalidRefreshToken
	}
	refreshToken, err := s.getRefreshToken(ctx, val)
	if err != nil {
		return "", "", op.ErrInvalidRefreshToken
	}
	if refreshToken.TokenType != TokenTypeRefresh {
		return "", "", op.ErrInvalidRefreshToken
	}
	if refreshToken.UserID != nil {
		userID = strconv.Itoa(*refreshToken.UserID)
	}
	return userID, refreshToken.ID.String(), nil
}

// RevokeToken should revoke a token. In the situation that the original request was to
// revoke an access token, then tokenOrTokenID will be a tokenID and userID will be set
// but if the original request was for a refresh token, then userID will be empty and
// tokenOrTokenID will be the refresh token, not its ID.  RevokeToken depends upon GetRefreshTokenInfo
// to get information from refresh tokens that are not either "<tokenID>:<userID>" strings
// nor JWTs.
func (s *AuthStorage) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	tokenID, err1 := uuid.Parse(tokenOrTokenID)
	userIDInt, err2 := strconv.Atoi(userID)
	if err1 != nil || err2 != nil {
		// Not a token of ours: RFC 7009 wants that to succeed
		return nil
	}
	if _, err := s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE id = $1 AND user_id = $2 AND application_id = $3", tokenID, userIDInt, clientID); err != nil {
		return oidc.ErrServerError()
	}
	return nil
}

// Cleanup deletes dead auth requests and tokens. Tokens linger for one refresh lifetime
// past their expiry: rows issued before access tokens pointed at their refresh token
// have the link the other way around, and deleting such an access token on time
// would cascade to its live refresh token.
func (s *AuthStorage) Cleanup(ctx context.Context) error {
	if _, err := s.conn.Exec(ctx, "DELETE FROM oauth_requests WHERE created_at < NOW() - interval '1 day'"); err != nil {
		return err
	}
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE expires_at < NOW() - make_interval(secs => $1)", refreshTokenLifetime.Seconds())
	return err
}

func (s *AuthStorage) SigningKey(ctx context.Context) (op.SigningKey, error) {
	return s.key, nil
}

func (s *AuthStorage) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}

func (s *AuthStorage) KeySet(ctx context.Context) ([]op.Key, error) {
	return []op.Key{&publicKey{s.key}}, nil
}

func (s *AuthStorage) getClient(ctx context.Context, clientID string) (*Client, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_clients WHERE id = $1", clientID)
	client, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Client])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, oidc.ErrInvalidClient()
		}
		return nil, err
	}
	return client, nil
}

// GetClientByClientID loads a Client. The returned Client is never cached and is only used to
// handle the current request.
func (s *AuthStorage) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	client, err := s.getClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *AuthStorage) AuthorizeClientIDSecret(ctx context.Context, clientID string, clientSecret string) error {
	var secretHash string
	err := s.conn.QueryRow(ctx, "SELECT secret_hash FROM oauth_clients WHERE id = $1", clientID).Scan(&secretHash)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(clientSecret)); err != nil {
		return oidc.ErrInvalidClient()
	}
	return nil
}

func (s *AuthStorage) setUserInfo(ctx context.Context, userinfo *oidc.UserInfo, userID int, clientID string, scopes []string) error {
	user, err := s.userRepo.User(ctx, kilonova.UserFilter{ID: &userID})
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found. Client ID: %q", clientID)
	}
	for _, scope := range scopes {
		switch scope {
		case oidc.ScopeOpenID:
			userinfo.Subject = strconv.Itoa(user.ID)
		case oidc.ScopeEmail:
			userinfo.Email = user.Email
			userinfo.EmailVerified = oidc.Bool(user.VerifiedEmail)
		case oidc.ScopeProfile:
			userinfo.PreferredUsername = user.Name
			userinfo.Picture = s.hostURL.JoinPath("profile", user.Name, "avatar").String() + "?s=128"
			switch user.PreferredLanguage {
			case "en":
				userinfo.Locale = oidc.NewLocale(language.English)
			case "ro":
				userinfo.Locale = oidc.NewLocale(language.Romanian)
			default:
				switch s.defaultLanguage(ctx) {
				case "en":
					userinfo.Locale = oidc.NewLocale(language.English)
				case "ro":
					userinfo.Locale = oidc.NewLocale(language.Romanian)
				default:
					userinfo.Locale = oidc.NewLocale(language.English)
				}
			}
			userinfo.AppendClaims("admin", user.Admin)
			userinfo.AppendClaims("proposer", user.Proposer)
		case oidc.ScopePhone:
			// We don't have phone numbers
			// userInfo.PhoneNumber =
			// userInfo.PhoneNumberVerified =
			// case oidc.Scope:
			// 	// you can also have a custom scope and assert public or custom claims based on that
			// 	userinfo.AppendClaims(CustomClaim, customClaim(clientID))
		}
	}
	return nil
}

// SetUserinfoFromScopes implements the op.Storage interface.
// Provide an empty implementation and use SetUserinfoFromRequest instead.
func (s *AuthStorage) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID string, clientID string, scopes []string) error {
	return nil
}

func (s *AuthStorage) SetUserinfoFromRequest(ctx context.Context, userinfo *oidc.UserInfo, token op.IDTokenRequest, scopes []string) error {
	userID, err := strconv.Atoi(token.GetSubject())
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	return s.setUserInfo(ctx, userinfo, userID, token.GetClientID(), scopes)
}

func (s *AuthStorage) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID string, subject string, origin string) error {
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	token, err := s.getAccessToken(ctx, tokenUUID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	if token.UserID == nil {
		return fmt.Errorf("token has no user ID")
	}

	// the userinfo endpoint should support CORS. If it's not possible to specify a specific origin in the CORS handler,
	// and you have to specify a wildcard (*) origin, then you could also check here if the origin which called the userinfo endpoint here directly
	// note that the origin can be empty (if called by a web client)
	// TODO: Better CORS
	// if origin != "" {
	// 	client, err := s.getClient(ctx, token.ApplicationID.String())
	// 	if err != nil {
	// 		return fmt.Errorf("failed to get client: %w", err)
	// 	}
	// 	if err := checkAllowedOrigins(client., origin); err != nil {
	// 		return err
	// 	}
	// }
	return s.setUserInfo(ctx, userinfo, *token.UserID, token.ApplicationID.String(), token.Scopes)
}

func (s *AuthStorage) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID string, subject string, clientID string) error {
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	token, err := s.getAccessToken(ctx, tokenUUID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	if token.UserID == nil {
		return fmt.Errorf("token has no user ID")
	}
	for _, aud := range token.Audience {
		if aud == clientID {
			userInfo := new(oidc.UserInfo)
			err := s.setUserInfo(ctx, userInfo, *token.UserID, clientID, token.Scopes)
			if err != nil {
				return fmt.Errorf("failed to set user info: %w", err)
			}
			introspection.SetUserInfo(userInfo)
			introspection.Scope = token.Scopes
			introspection.ClientID = token.ApplicationID.String()
			return nil
		}
	}
	return fmt.Errorf("token is not valid for this client")
}

func (s *AuthStorage) GetPrivateClaimsFromScopes(ctx context.Context, userID string, clientID string, scopes []string) (map[string]any, error) {
	return map[string]any{}, nil
}

func (s *AuthStorage) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	// TODO: Implement
	return nil, fmt.Errorf("jwt profile authorization grant is not implemented")
}

func (s *AuthStorage) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	// TODO: Implement
	return nil, fmt.Errorf("jwt profile authorization grant is not implemented")
}

func (s *AuthStorage) Health(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// StorageConfig holds the signing key material. The caller owns generating and
// persisting it; NewAuthStorage only validates it.
type StorageConfig struct {
	// PrivateKey is a base64 PKCS#1 RSA key, as produced by GeneratePrivateKey.
	PrivateKey string
	KeyID      string

	// DefaultLanguage is the locale fallback for users with no preferred language.
	DefaultLanguage func(ctx context.Context) string
	// HostURL is the instance's public URL, used to build absolute links in claims.
	HostURL *url.URL
}

func NewAuthStorage(pgx *postgres.DB, conf StorageConfig) (*AuthStorage, error) {
	if conf.PrivateKey == "" || conf.KeyID == "" {
		return nil, errors.New("auth storage needs both a private key and a key ID")
	}
	if conf.DefaultLanguage == nil {
		return nil, errors.New("auth storage needs a default language source")
	}
	if conf.HostURL == nil {
		return nil, errors.New("auth storage needs a host URL")
	}
	key, err := parsePrivateKey(conf.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA private key: %w", err)
	}
	return &AuthStorage{
		conn:     pgx.Pool(),
		key:      &signingKey{pkey: key, kid: conf.KeyID},
		userRepo: userpg.NewRepository(pgx),

		defaultLanguage: conf.DefaultLanguage,
		hostURL:         conf.HostURL,
	}, nil
}

func promptToInternal(oidcPrompt oidc.SpaceDelimitedArray) []string {
	prompts := make([]string, 0, len(oidcPrompt))
	for _, oidcPrompt := range oidcPrompt {
		switch oidcPrompt {
		case oidc.PromptNone,
			oidc.PromptLogin,
			oidc.PromptConsent,
			oidc.PromptSelectAccount:
			prompts = append(prompts, oidcPrompt)
		}
	}
	return prompts
}

func maxAgeToInternal(createdAt time.Time, maxAge *uint) *time.Time {
	if maxAge == nil {
		return nil
	}
	return new(createdAt.Add(time.Duration(*maxAge) * time.Second))
}
