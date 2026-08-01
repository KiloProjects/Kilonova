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
)

const scratchTok = "tok-scratch"

func startScratchServer(t *testing.T) (base string, fs afero.Fs) {
	t.Helper()
	fs = afero.NewMemMapFs()
	reg := scheduler.NewClientRegistry()
	if err := reg.Add(scratchTok, "platform", ""); err != nil {
		t.Fatal(err)
	}
	path, h := scheduler.ScratchHandler(fs, reg)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/scratch", fs
}

func TestHTTPScratchRoundTrip(t *testing.T) {
	base, fs := startScratchServer(t)
	sc, err := scratch.NewHTTP(http.DefaultClient, base, scratchTok)
	if err != nil {
		t.Fatal(err)
	}

	const want = "hello grader"
	id, err := sc.SaveFile(strings.NewReader(want))
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	// The bytes must be on the server's disk the moment SaveFile returns.
	if ok, _ := afero.Exists(fs, id); !ok {
		t.Fatal("file not durable on server after SaveFile returned")
	}

	rc, err := sc.ReadFile(id)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != want {
		t.Fatalf("round-trip mismatch: got %q want %q", got, want)
	}

	if err := sc.DeleteFile(id); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if ok, _ := afero.Exists(fs, id); ok {
		t.Fatal("file still present after DeleteFile")
	}
}

func TestHTTPScratchRejectsPathTraversal(t *testing.T) {
	base, fs := startScratchServer(t)
	// Plant a secret outside the flat id namespace the client can mint.
	afero.WriteFile(fs, "secret", []byte("root pw"), 0o644)

	for _, id := range []string{"..%2Fsecret", "secret", "../secret", "not-a-uuid"} {
		req, _ := http.NewRequest(http.MethodGet, base+"/"+id, nil)
		req.Header.Set("Authorization", "Bearer "+scratchTok)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET %q: %v", id, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("non-UUID id %q served content: %q", id, body)
		}
	}
}

func TestHTTPScratchRejectsBadToken(t *testing.T) {
	base, _ := startScratchServer(t)
	req, _ := http.NewRequest(http.MethodGet, base+"/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token got %d, want 401", resp.StatusCode)
	}
}
