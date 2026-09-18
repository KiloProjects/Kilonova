package web

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// Catches a missing/stale Vite build as well as a manifest key that no longer
// matches what the templates ask for.
func TestAssetsResolveToEmbeddedFiles(t *testing.T) {
	for _, src := range []string{"app.ts", "vendored.ts", "chroma.css", "vendor.css", "tailwind.css"} {
		url := assets.Asset(src)
		if !strings.HasPrefix(url, "/static/misc/") {
			t.Errorf("%s: not in the manifest, got %q", src, url)
			continue
		}
		if _, err := fs.Stat(embedded, strings.TrimPrefix(url, "/")); err != nil {
			t.Errorf("%s: %v", src, err)
		}
	}
}

func TestStaticCacheHeaders(t *testing.T) {
	for url, want := range map[string]string{
		assets.Asset("app.ts"):         "public, max-age=31536000, immutable",
		"/static/favicons/favicon.ico": "public, max-age=3600",
	} {
		w := httptest.NewRecorder()
		staticFileServer(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code != http.StatusOK {
			t.Errorf("%s: status %d", url, w.Code)
		}
		if got := w.Header().Get("Cache-Control"); got != want {
			t.Errorf("%s: Cache-Control = %q, want %q", url, got, want)
		}
	}
}

func TestStaticNoDirectoryListing(t *testing.T) {
	for _, url := range []string{"/static", "/static/misc", "/static/../go.mod"} {
		w := httptest.NewRecorder()
		staticFileServer(w, httptest.NewRequest(http.MethodGet, url, nil))
		if w.Code == http.StatusOK {
			t.Errorf("%s: served %q", url, w.Body.String())
		}
	}
}

// The fonts are referenced from inside the stylesheets, where Vite's `base`
// decides the prefix, not the manifest. Getting it wrong 404s every webfont
// while the stylesheet itself still loads fine.
func TestCSSAssetURLsResolve(t *testing.T) {
	url := regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)
	for _, src := range []string{"chroma.css", "vendor.css", "tailwind.css"} {
		css, err := fs.ReadFile(embedded, strings.TrimPrefix(assets.Asset(src), "/"))
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		found := 0
		for _, m := range url.FindAllStringSubmatch(string(css), -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "http") {
				continue
			}
			found++
			if !strings.HasPrefix(ref, "/static/misc/") {
				t.Errorf("%s: %q is not served from /static/misc", src, ref)
				continue
			}
			if _, err := fs.Stat(embedded, strings.TrimPrefix(ref, "/")); err != nil {
				t.Errorf("%s: %q: %v", src, ref, err)
			}
		}
		if src == "vendor.css" && found == 0 {
			t.Error("vendor.css: no webfont references found, the check is not testing anything")
		}
	}
}
