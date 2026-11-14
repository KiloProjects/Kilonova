package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// TestTokenLifecycle needs a Postgres server: set KN_TEST_DSN to run it.
// It works inside a schema of its own, which it drops when done.
func TestTokenLifecycle(t *testing.T) {
	dsn := os.Getenv("KN_TEST_DSN")
	if dsn == "" {
		t.Skip("KN_TEST_DSN not set")
	}
	ctx := context.Background()
	conf, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	conf.ConnConfig.RuntimeParams["search_path"] = "kn_auth_test"
	pool, err := pgxpool.NewWithConfig(ctx, conf)
	require.NoError(t, err)
	defer pool.Close()
	schema, err := os.ReadFile("../../db/psql_schema/007.oauth2_oidc.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "DROP SCHEMA IF EXISTS kn_auth_test CASCADE; CREATE SCHEMA kn_auth_test; CREATE TABLE users (id bigint PRIMARY KEY); INSERT INTO users VALUES (1);"+string(schema))
	require.NoError(t, err)
	defer pool.Exec(ctx, "DROP SCHEMA kn_auth_test CASCADE")
	s := &AuthStorage{conn: pool}

	clientID, _, err := s.CreateClient(ctx, "c", ApplicationTypeWeb, 1, false, []string{"https://x/cb"}, []string{})
	require.NoError(t, err)

	newReq := func() *Request {
		r, err := s.CreateAuthRequest(ctx, &oidc.AuthRequest{ClientID: clientID.String(), RedirectURI: "https://x/cb", Scopes: []string{"openid", "api", "offline_access"}, ResponseType: oidc.ResponseTypeCode}, "")
		require.NoError(t, err)
		require.NoError(t, s.ApproveAuthRequest(ctx, r.GetID(), 1))
		got, err := s.AuthRequestByID(ctx, r.GetID())
		require.NoError(t, err)
		require.True(t, got.Done())
		return got.(*Request)
	}
	count := func() (n int) {
		require.NoError(t, pool.QueryRow(ctx, "SELECT COUNT(*) FROM oauth_tokens").Scan(&n))
		return
	}
	alive := func(id string, typ TokenType) bool {
		_, err := s.getToken(ctx, mustUUID(t, id), typ)
		return err == nil
	}

	req := newReq()
	// code expiry
	require.NoError(t, s.SaveAuthCode(ctx, req.GetID(), "code1"))
	_, err = s.AuthRequestByCode(ctx, "code1")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE oauth_requests SET auth_time = NOW() - interval '11 minutes' WHERE id = $1", req.ID)
	require.NoError(t, err)
	_, err = s.AuthRequestByCode(ctx, "code1")
	require.Error(t, err, "stale code")

	// lone access token
	lone, exp, err := s.CreateAccessToken(ctx, req)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(accessTokenLifetime), exp, time.Second)
	require.True(t, alive(lone, TokenTypeAccess))

	// code flow with refresh
	a1, r1, exp, err := s.CreateAccessAndRefreshTokens(ctx, req, "")
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(accessTokenLifetime), exp, time.Second, "expiration is the access token's")
	require.True(t, alive(a1, TokenTypeAccess))

	// rotation with narrowed scopes
	rr, err := s.TokenRequestByRefreshToken(ctx, r1)
	require.NoError(t, err)
	rr.SetCurrentScopes([]string{"openid"})
	a2, r2, _, err := s.CreateAccessAndRefreshTokens(ctx, rr, r1)
	require.NoError(t, err)
	require.False(t, alive(a1, TokenTypeAccess), "old access token goes with its refresh token")
	require.False(t, alive(r1, TokenTypeRefresh))
	at, err := s.getAccessToken(ctx, mustUUID(t, a2))
	require.NoError(t, err)
	require.Equal(t, []string{"openid"}, at.Scopes)
	rt, err := s.getRefreshToken(ctx, mustUUID(t, r2))
	require.NoError(t, err)
	require.Equal(t, []string{"openid", "api", "offline_access"}, rt.Scopes)
	require.Equal(t, []string{"pwd"}, rt.AMR)
	require.NotNil(t, rt.AuthTime)

	// reuse of the rotated token
	before := count()
	_, _, _, err = s.CreateAccessAndRefreshTokens(ctx, rr, r1)
	require.True(t, errors.Is(err, op.ErrInvalidRefreshToken), "%v", err)
	require.Equal(t, before, count(), "failed rotation leaves nothing behind")

	// revocation: refresh token takes its access token with it
	uid, tid, err := s.GetRefreshTokenInfo(ctx, clientID.String(), r2)
	require.NoError(t, err)
	require.Nil(t, s.RevokeToken(ctx, tid, uid, clientID.String()))
	require.False(t, alive(a2, TokenTypeAccess))
	require.Nil(t, s.RevokeToken(ctx, "garbage", "", clientID.String()))

	// expired refresh token
	_, r3, _, err := s.CreateAccessAndRefreshTokens(ctx, req, "")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, "UPDATE oauth_tokens SET expires_at = NOW() - interval '1 second' WHERE id = $1", r3)
	require.NoError(t, err)
	_, err = s.TokenRequestByRefreshToken(ctx, r3)
	require.Error(t, err)
	_, _, _, err = s.CreateAccessAndRefreshTokens(ctx, rr, r3)
	require.True(t, errors.Is(err, op.ErrInvalidRefreshToken))

	// cleanup
	before = count()
	require.NoError(t, s.Cleanup(ctx))
	require.Equal(t, before, count(), "recently expired rows are kept")
	_, err = pool.Exec(ctx, "UPDATE oauth_tokens SET expires_at = NOW() - interval '3 days'; UPDATE oauth_requests SET created_at = NOW() - interval '2 days'")
	require.NoError(t, err)
	require.NoError(t, s.Cleanup(ctx))
	require.Zero(t, count())
	_, err = s.AuthRequestByID(ctx, req.GetID())
	require.Error(t, err)
}

func mustUUID(t *testing.T, s string) uuid.UUID {
	v, err := uuid.Parse(s)
	require.NoError(t, err)
	return v
}
