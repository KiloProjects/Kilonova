package api

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/KiloProjects/kilonova/domain/config"
	_ "github.com/KiloProjects/kilonova/sudoapi/flags" // registers the flags under test
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
)

type memFlagStore struct {
	mu   sync.Mutex
	data map[string]json.RawMessage
}

func (m *memFlagStore) GetFlags(context.Context) (map[string]json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.data), nil
}

func (m *memFlagStore) SetFlag(_ context.Context, key string, value json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

// newFlagAPI registers the flag operations without auth middleware, with fs
// attached to every request the way the instance middleware does it.
func newFlagAPI(t *testing.T) (humatest.TestAPI, *config.Flagset, *memFlagStore) {
	t.Helper()
	_, api := humatest.New(t)
	store := &memFlagStore{data: map[string]json.RawMessage{}}
	fs := config.NewFlagset()
	require.NoError(t, fs.InitializeFlags(t.Context(), store))
	api.UseMiddleware(func(ctx huma.Context, next func(huma.Context)) {
		next(huma.WithContext(ctx, config.WithFlagset(ctx.Context(), fs)))
	})
	registerFlagOps(api)
	return api, fs, store
}

func TestUpdateFlags(t *testing.T) {
	api, _, store := newFlagAPI(t)
	resp := api.Patch("/flags", map[string]any{
		"string_flags": map[string]string{"frontend.navbar.branding": "Test", "no.such.flag": "x"},
		"bool_flags":   map[string]bool{"feature.platform.signup": false},
	})
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())

	var res flagUpdateResult
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
	require.ElementsMatch(t, []string{"frontend.navbar.branding", "feature.platform.signup"}, res.Updated)
	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0], "no.such.flag")
	require.Equal(t, json.RawMessage(`"Test"`), store.data["frontend.navbar.branding"], "update is persisted")
}

func TestExportFlags(t *testing.T) {
	api, _, _ := newFlagAPI(t)
	api.Patch("/flags", map[string]any{"string_flags": map[string]string{
		"frontend.navbar.branding": "Test",
		"oidc.cryptoKey":           "secret",
	}})

	var env flagEnvelope
	resp := api.Get("/flags/export")
	require.Equal(t, http.StatusOK, resp.Code)
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	require.Equal(t, flagExportVersion, env.Version)
	require.Contains(t, env.Flags, "frontend.navbar.branding")
	require.NotContains(t, env.Flags, "oidc.cryptoKey", "secrets need opt-in")
	require.NotContains(t, env.Flags, "feature.platform.signup", "defaults are not exported")

	resp = api.Get("/flags/export?secrets=true")
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &env))
	require.Contains(t, env.Flags, "oidc.cryptoKey")
}

func TestImportFlags(t *testing.T) {
	t.Run("partial file", func(t *testing.T) {
		api, fs, _ := newFlagAPI(t)
		resp := api.Post("/flags/import", map[string]any{"version": 1, "flags": map[string]any{
			"frontend.navbar.branding":  "Imported",
			"oidc.cryptoKey":            "copied",
			"integrations.moss.user_id": "abc",
			"removed.flag":              true,
		}})
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		var res config.ImportResult
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		require.Equal(t, []string{"frontend.navbar.branding", "oidc.cryptoKey"}, res.Applied)
		require.Equal(t, []string{"oidc.cryptoKey"}, res.AppliedSecrets)
		require.Contains(t, res.Skipped, "integrations.moss.user_id")
		require.Contains(t, res.Skipped, "removed.flag")

		branding, _ := config.GetFlag[string]("frontend.navbar.branding")
		require.Equal(t, "Imported", branding.Of(fs))
	})

	t.Run("unsupported version changes nothing", func(t *testing.T) {
		api, fs, store := newFlagAPI(t)
		resp := api.Post("/flags/import", map[string]any{"version": 2, "flags": map[string]any{"frontend.navbar.branding": "x"}})
		require.Equal(t, http.StatusBadRequest, resp.Code)
		require.Empty(t, store.data)
		out, err := fs.Export(true)
		require.NoError(t, err)
		require.Empty(t, out)
	})

	t.Run("invalid JSON changes nothing", func(t *testing.T) {
		api, _, store := newFlagAPI(t)
		resp := api.Post("/flags/import", "Content-Type: application/json", strings.NewReader(`{"version": 1, "flags": `))
		require.GreaterOrEqual(t, resp.Code, 400)
		require.Less(t, resp.Code, 500)
		require.Empty(t, store.data)
	})
}

// A file is exactly what export returned (including huma's $schema link) and must import cleanly.
func TestExportImportRoundTrip(t *testing.T) {
	src, _, _ := newFlagAPI(t)
	src.Patch("/flags", map[string]any{"string_flags": map[string]string{"frontend.navbar.branding": "Copied"}})
	exported := src.Get("/flags/export").Body.String()

	dst, fs, _ := newFlagAPI(t)
	resp := dst.Post("/flags/import", "Content-Type: application/json", strings.NewReader(exported))
	require.Equal(t, http.StatusOK, resp.Code, "export body: %s\nimport response: %s", exported, resp.Body.String())
	branding, _ := config.GetFlag[string]("frontend.navbar.branding")
	require.Equal(t, "Copied", branding.Of(fs))
}
