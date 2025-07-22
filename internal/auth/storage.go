package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/repository"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

var _ op.Storage = (*AuthStorage)(nil)

// Outside Proof of Concept, TODO: Implement
// var _ op.TokenExchangeStorage = (*AuthStorage)(nil)
// var _ op.TokenExchangeTokensVerifierStorage = (*AuthStorage)(nil)
// var _ op.DeviceAuthorizationStorage = (*AuthStorage)(nil)
// var _ op.ClientCredentialsStorage = (*AuthStorage)(nil)

type AuthStorage struct {
	conn     *pgxpool.Pool
	key      *signingKey
	userRepo *repository.UserRepository
}

func (s *AuthStorage) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error) {
	if len(authReq.Prompt) == 1 && authReq.Prompt[0] == "none" {
		return nil, oidc.ErrLoginRequired()
	}

	appID, err := uuid.Parse(authReq.ClientID)
	if err != nil {
		return nil, err
	}

	req := &AuthRequest{
		ID:            uuid.Must(uuid.NewV7()),
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
		UiLocales: authReq.UILocales,
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
	`, req.ID, req.ApplicationID, req.CallbackURI, req.TransferState, req.UserID, req.Scopes, req.ResponseType, req.ResponseMode, req.Nonce, req.CodeChallenge, req.CodeChallengeMethod, req.Prompt, req.UiLocales, req.LoginHint, req.ExpiresAt).Scan(&req.CreatedAt); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *AuthStorage) AuthRequestByID(ctx context.Context, reqID string) (op.AuthRequest, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_requests WHERE id = $1", reqID)
	val, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[AuthRequest])
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *AuthStorage) AuthRequestByCode(ctx context.Context, reqCode string) (op.AuthRequest, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_requests WHERE request_code = $1", reqCode)
	val, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[AuthRequest])
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
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_tokens WHERE id = $1 AND token_type = $2", tokenID, TokenTypeAccess)
	return pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Token])
}

func (s *AuthStorage) getRefreshToken(ctx context.Context, tokenID uuid.UUID) (*Token, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM oauth_tokens WHERE id = $1 AND token_type = $2", tokenID, TokenTypeRefresh)
	return pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Token])
}

func (s *AuthStorage) createAccessToken(ctx context.Context, applicationID uuid.UUID, refreshTokenID *uuid.UUID, userID *int, audience []string, scopes []string) (*Token, error) {
	token := &Token{
		ID:            uuid.Must(uuid.NewV7()),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(accessTokenLifetime),
		ApplicationID: applicationID,
		ParentToken:   refreshTokenID,
		UserID:        userID,
		Audience:      audience,
		Scopes:        scopes,
		TokenType:     TokenTypeAccess,
	}

	if _, err := s.conn.Exec(ctx, `
		INSERT INTO oauth_tokens (id, created_at, expires_at, application_id, from_token, user_id, audience, scopes, token_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, token.ID, token.CreatedAt, token.ExpiresAt, token.ApplicationID, token.ParentToken, token.UserID, token.Audience, token.Scopes, token.TokenType); err != nil {
		return nil, err
	}

	return token, nil
}

func (s *AuthStorage) createRefreshToken(ctx context.Context, accessToken *Token, amr []string, authTime time.Time) (*Token, error) {
	token := &Token{
		ID:            uuid.Must(uuid.NewV7()),
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(refreshTokenLifetime),
		ApplicationID: accessToken.ApplicationID,
		ParentToken:   &accessToken.ID,
		UserID:        accessToken.UserID,
		Scopes:        accessToken.Scopes,
		Audience:      accessToken.Audience,
		TokenType:     TokenTypeRefresh,

		AMR:      amr,
		AuthTime: &authTime,
	}

	if _, err := s.conn.Exec(ctx, `
		INSERT INTO oauth_tokens (id, created_at, expires_at, application_id, from_token, user_id, audience, scopes, token_type, amr, auth_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, token.ID, token.CreatedAt, token.ExpiresAt, token.ApplicationID, token.ParentToken, token.UserID, token.Audience, token.Scopes, token.TokenType, token.AMR, token.AuthTime); err != nil {
		return nil, err
	}

	return token, nil
}

func (s *AuthStorage) renewRefreshToken(ctx context.Context, currentRefreshTokenID uuid.UUID, newRefreshTokenID uuid.UUID, accessTokenID uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.conn, func(tx pgx.Tx) error {
		rows, _ := tx.Query(ctx, "SELECT * FROM oauth_tokens WHERE id = $1 AND token_type = $2", currentRefreshTokenID, TokenTypeRefresh)
		refreshToken, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[Token])
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, "DELETE FROM oauth_tokens WHERE id = $1", currentRefreshTokenID)
		if err != nil {
			return err
		}

		refreshToken.ID = newRefreshTokenID
		refreshToken.ExpiresAt = time.Now().Add(refreshTokenLifetime)
		refreshToken.ParentToken = &accessTokenID

		if _, err = tx.Exec(ctx, `INSERT INTO oauth_tokens 
		(id, created_at, expires_at, application_id, from_token, user_id, audience, scopes, token_type, amr, auth_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, newRefreshTokenID, refreshToken.CreatedAt, refreshToken.ExpiresAt, refreshToken.ApplicationID, refreshToken.ParentToken, refreshToken.UserID, refreshToken.Audience, refreshToken.Scopes, refreshToken.TokenType, refreshToken.AMR, refreshToken.AuthTime); err != nil {
			return err
		}

		return nil
	})
}

func (s *AuthStorage) exchangeRefreshToken(ctx context.Context, req op.TokenExchangeRequest) (accessTokenID uuid.UUID, newRefreshTokenID uuid.UUID, expiration time.Time, err error) {
	applicationID, err := uuid.Parse(req.GetClientID())
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}

	authTime := req.GetAuthTime()

	slog.InfoContext(ctx, "Exchange refresh token", slog.String("subject", req.GetSubject()))
	userID, err := strconv.Atoi(req.GetSubject())
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}

	accessToken, err := s.createAccessToken(ctx, applicationID, nil, &userID, req.GetAudience(), req.GetScopes())
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}

	refreshToken, err := s.createRefreshToken(ctx, accessToken, nil, authTime)
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}

	return accessToken.ID, refreshToken.ID, refreshToken.ExpiresAt, nil
}

// The TokenRequest parameter of CreateAccessToken can be any of:
//
// * TokenRequest as returned by ClientCredentialsStorage.ClientCredentialsTokenRequest,
//
// * AuthRequest as returned by AuthRequestByID or AuthRequestByCode (above)
//
//   - *oidc.JWTTokenRequest from a JWT that is the assertion value of a JWT Profile
//     Grant: https://datatracker.ietf.org/doc/html/rfc7523#section-2.1
//
// * TokenExchangeRequest as returned by ValidateTokenExchangeRequest
func (s *AuthStorage) CreateAccessToken(ctx context.Context, req op.TokenRequest) (string, time.Time, error) {
	var appID uuid.UUID
	switch req := req.(type) {
	case *AuthRequest:
		appID = req.ApplicationID
	case op.TokenExchangeRequest:
		var err error
		appID, err = uuid.Parse(req.GetClientID())
		if err != nil {
			return "", time.Time{}, err
		}
	}

	slog.InfoContext(ctx, "Create access token", slog.String("subject", req.GetSubject()))
	userID, err := strconv.Atoi(req.GetSubject())
	if err != nil {
		return "", time.Time{}, err
	}

	token, err := s.createAccessToken(ctx, appID, nil, &userID, req.GetAudience(), req.GetScopes())
	if err != nil {
		return "", time.Time{}, err
	}
	return token.ID.String(), token.ExpiresAt, nil
}

// The TokenRequest parameter of CreateAccessAndRefreshTokens can be any of:
//
// * TokenRequest as returned by ClientCredentialsStorage.ClientCredentialsTokenRequest
//
// * RefreshTokenRequest as returned by AuthStorage.TokenRequestByRefreshToken
//
//   - AuthRequest as by returned by the AuthRequestByID or AuthRequestByCode (above).
//     Used for the authorization code flow which requested offline_access scope and
//     registered the refresh_token grant type in advance
//
// * TokenExchangeRequest as returned by ValidateTokenExchangeRequest
func (s *AuthStorage) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshTokenID string, expiration time.Time, err error) {
	// generate tokens via token exchange flow if request is relevant
	if teReq, ok := request.(op.TokenExchangeRequest); ok {
		accessTokenID, newRefreshTokenID, expiration, err := s.exchangeRefreshToken(ctx, teReq)
		if err != nil {
			return "", "", time.Time{}, err
		}
		return accessTokenID.String(), newRefreshTokenID.String(), expiration, nil
	}

	// get the information depending on the request type / implementation
	applicationID, authTime, userID, amr := getInfoFromRequest(request)

	// if currentRefreshToken is empty (Code Flow) we will have to create a new refresh token
	if currentRefreshToken == "" {
		accessToken, err := s.createAccessToken(ctx, applicationID, nil, userID, request.GetAudience(), request.GetScopes())
		if err != nil {
			return "", "", time.Time{}, err
		}
		refreshToken, err := s.createRefreshToken(ctx, accessToken, amr, authTime)
		if err != nil {
			return "", "", time.Time{}, err
		}
		return accessToken.ID.String(), refreshToken.ID.String(), refreshToken.ExpiresAt, nil
	}

	// if we get here, the currentRefreshToken was not empty, so the call is a refresh token request
	// we therefore will have to check the currentRefreshToken and renew the refresh token

	refreshTokenID, err := uuid.Parse(currentRefreshToken)
	if err != nil {
		return "", "", time.Time{}, err
	}

	newRefreshToken := uuid.Must(uuid.NewV7())

	accessToken, err := s.createAccessToken(ctx, applicationID, &newRefreshToken, userID, request.GetAudience(), request.GetScopes())
	if err != nil {
		return "", "", time.Time{}, err
	}

	if err := s.renewRefreshToken(ctx, refreshTokenID, newRefreshToken, accessToken.ID); err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken.ID.String(), newRefreshToken.String(), accessToken.ExpiresAt, nil
}

// getInfoFromRequest returns the clientID, authTime and amr depending on the op.TokenRequest type / implementation
func getInfoFromRequest(req op.TokenRequest) (applicationID uuid.UUID, authTime time.Time, userID *int, amr []string) {
	authReq, ok := req.(*AuthRequest) // Code Flow (with scope offline_access)
	if ok {
		t := time.Time{}
		if authReq.AuthTime != nil {
			t = *authReq.AuthTime
		}
		return authReq.ApplicationID, t, authReq.UserID, authReq.GetAMR()
	}
	refreshReq, ok := req.(*RefreshTokenRequest) // Refresh Token Request
	if ok {
		return refreshReq.ApplicationID, refreshReq.GetAuthTime(), refreshReq.UserID, refreshReq.AMR
	}
	return uuid.Nil, time.Time{}, nil, nil
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
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE user_id = $1 AND application_id = $2", userID, clientID)
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
	_, err := s.conn.Exec(ctx, "DELETE FROM oauth_tokens WHERE id = $1 AND user_id = $2 AND application_id = $3", tokenOrTokenID, userID, clientID)
	if err != nil {
		return oidc.ErrServerError()
	}
	return nil
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
		return fmt.Errorf("user not found")
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
			userinfo.Name = user.Name
			userinfo.FamilyName = user.Name
			userinfo.GivenName = user.Name
			switch user.PreferredLanguage {
			case "en":
				userinfo.Locale = oidc.NewLocale(language.English)
			case "ro":
				userinfo.Locale = oidc.NewLocale(language.Romanian)
			default:
				switch config.Common.DefaultLang {
				case "en":
					userinfo.Locale = oidc.NewLocale(language.English)
				case "ro":
					userinfo.Locale = oidc.NewLocale(language.Romanian)
				default:
					userinfo.Locale = oidc.NewLocale(language.English)
				}
			}
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
	token, err := s.getAccessToken(ctx, uuid.Must(uuid.Parse(tokenID)))
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
	if token.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("token expired")
	}

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

func NewAuthStorage(ctx context.Context, conn *pgxpool.Pool) *AuthStorage {
	key, err := getKey()
	if err != nil {
		slog.ErrorContext(ctx, "failed to get RSA key", slog.Any("error", err))
		os.Exit(1)
	}
	return &AuthStorage{
		conn:     conn,
		key:      &signingKey{pkey: key, kid: RSAPrivateKeyID.Value()},
		userRepo: repository.NewUserRepository(conn),
	}
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
	expiresAt := createdAt.Add(time.Duration(*maxAge) * time.Second)
	return &expiresAt
}
