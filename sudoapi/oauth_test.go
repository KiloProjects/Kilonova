package sudoapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/go-chi/chi/v5"
)

// The provider's routes must answer when registered on a router that is itself mounted, as cmd/kn does
func TestRegisterOIDCRoutes(t *testing.T) {
	key, err := auth.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	host := &url.URL{Scheme: "https", Host: "kn.example"}
	storage, err := auth.NewAuthStorage(&postgres.DB{}, auth.StorageConfig{
		PrivateKey: key, KeyID: "kid", HostURL: host,
		DefaultLanguage: func(context.Context) string { return "en" },
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := auth.GetProvider(auth.OauthProviderConfig{Storage: storage, HostPrefix: host.String(), CryptoKey: []byte("k")})
	if err != nil {
		t.Fatal(err)
	}

	web := chi.NewRouter()
	(&BaseAPI{oidcProvider: provider}).RegisterOIDCRoutes(web)
	root := chi.NewRouter()
	root.Mount("/", web)

	for path, want := range map[string]int{
		"/.well-known/openid-configuration": http.StatusOK,
		"/keys":                             http.StatusOK,
		"/ready":                            http.StatusNotFound,
		"/nope":                             http.StatusNotFound,
	} {
		w := httptest.NewRecorder()
		root.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != want {
			t.Errorf("%s: got %d, want %d: %s", path, w.Code, want, w.Body)
		}
	}
}
