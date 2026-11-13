package scheduler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scratch"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

const scratchTok = "tok-scratch"

func startScratchServer(t *testing.T) (base string, fs afero.Fs) {
	t.Helper()
	fs = afero.NewMemMapFs()
	reg := scheduler.NewClientRegistry()
	require.NoError(t, reg.Add(scratchTok, "platform", ""))
	mux := http.NewServeMux()
	mux.Handle(scheduler.ScratchHandler(fs))
	srv := httptest.NewServer(reg.Auth(mux))
	t.Cleanup(srv.Close)
	return srv.URL + "/scratch", fs
}

func TestHTTPScratchRoundTrip(t *testing.T) {
	base, fs := startScratchServer(t)
	sc, err := scratch.NewHTTP(http.DefaultClient, base, scratchTok)
	require.NoError(t, err)

	const want = "hello grader"
	id, err := sc.SaveFile(strings.NewReader(want))
	require.NoError(t, err, "SaveFile")
	// The bytes must be on the server's disk the moment SaveFile returns.
	ok, _ := afero.Exists(fs, id)
	require.True(t, ok, "file not durable on server after SaveFile returned")

	rc, err := sc.ReadFile(id)
	require.NoError(t, err, "ReadFile")
	got, _ := io.ReadAll(rc)
	rc.Close()
	require.Equal(t, want, string(got))

	require.NoError(t, sc.DeleteFile(id), "DeleteFile")
	ok, _ = afero.Exists(fs, id)
	require.False(t, ok, "file still present after DeleteFile")
}

func TestHTTPScratchRejectsPathTraversal(t *testing.T) {
	base, fs := startScratchServer(t)
	// Plant a secret outside the flat id namespace the client can mint.
	afero.WriteFile(fs, "secret", []byte("root pw"), 0o644)

	for _, id := range []string{"..%2Fsecret", "secret", "../secret", "not-a-uuid"} {
		req, _ := http.NewRequest(http.MethodGet, base+"/"+id, nil)
		req.Header.Set("Authorization", "Bearer "+scratchTok)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "GET %q", id)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NotEqual(t, http.StatusOK, resp.StatusCode, "non-UUID id %q served content: %q", id, body)
	}
}

func TestHTTPScratchRejectsBadToken(t *testing.T) {
	base, _ := startScratchServer(t)
	req, _ := http.NewRequest(http.MethodGet, base+"/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
