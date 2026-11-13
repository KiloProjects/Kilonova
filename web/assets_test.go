package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Catches a missing/stale Vite build as well as a manifest key that no longer
// matches what the templates ask for.
func TestAssetsResolveToEmbeddedFiles(t *testing.T) {
	for _, src := range []string{"app.ts", "vendored.ts", "chroma.css", "vendor.css", "tailwind.css"} {
		url := assets.Asset(t.Context(), src)
		if !assert.True(t, strings.HasPrefix(url, "/static/misc/"), "%s: not in the manifest, got %q", src, url) {
			continue
		}
		_, err := fs.Stat(embedded, strings.TrimPrefix(url, "/"))
		assert.NoError(t, err, src)
	}
}

func TestStaticCacheHeaders(t *testing.T) {
	for url, want := range map[string]string{
		assets.Asset(t.Context(), "app.ts"): "public, max-age=31536000, immutable",
		"/static/favicons/favicon.ico":      "public, max-age=3600",
	} {
		w := httptest.NewRecorder()
		staticFileServer(w, httptest.NewRequest(http.MethodGet, url, nil))
		assert.Equal(t, http.StatusOK, w.Code, url)
		assert.Equal(t, want, w.Header().Get("Cache-Control"), url)
	}
}

func TestStaticNoDirectoryListing(t *testing.T) {
	for _, url := range []string{"/static", "/static/misc", "/static/../go.mod"} {
		w := httptest.NewRecorder()
		staticFileServer(w, httptest.NewRequest(http.MethodGet, url, nil))
		assert.NotEqual(t, http.StatusOK, w.Code, "%s: served %q", url, w.Body.String())
	}
}

// The fonts are referenced from inside the stylesheets, where Vite's `base`
// decides the prefix, not the manifest. Getting it wrong 404s every webfont
// while the stylesheet itself still loads fine.
func TestCSSAssetURLsResolve(t *testing.T) {
	url := regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)
	for _, src := range []string{"chroma.css", "vendor.css", "tailwind.css"} {
		css, err := fs.ReadFile(embedded, strings.TrimPrefix(assets.Asset(t.Context(), src), "/"))
		require.NoError(t, err, src)
		found := 0
		for _, m := range url.FindAllStringSubmatch(string(css), -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "http") {
				continue
			}
			found++
			if !assert.True(t, strings.HasPrefix(ref, "/static/misc/"), "%s: %q is not served from /static/misc", src, ref) {
				continue
			}
			_, err := fs.Stat(embedded, strings.TrimPrefix(ref, "/"))
			assert.NoError(t, err, "%s: %q", src, ref)
		}
		if src == "vendor.css" {
			assert.NotZero(t, found, "vendor.css: no webfont references found, the check is not testing anything")
		}
	}
}
