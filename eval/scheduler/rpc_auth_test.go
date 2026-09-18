package scheduler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/eval"
)

// recordingSched records whether a box was actually executed, so we can prove
// rejected requests never reach execution.
type recordingSched struct{ ran bool }

func (r *recordingSched) RunBox3(context.Context, *eval.Box3Request, int64) (*eval.Box3Response, error) {
	r.ran = true
	return &eval.Box3Response{Stats: &eval.RunStats{Status: "OK"}, Files: map[string]string{}}, nil
}
func (r *recordingSched) RunMultibox3(context.Context, *eval.Multibox3Request, int64, int64) (*eval.Box3Response, []*eval.RunStats, error) {
	return nil, nil, nil
}
func (r *recordingSched) Close(context.Context) error { return nil }

// stubLangs is a LanguageManager whose only interesting method is LanguageVersions.
type stubLangs struct {
	eval.LanguageManager
	versions map[string]string
}

func (s stubLangs) LanguageVersions(context.Context) map[string]string { return s.versions }

func startTestGrader(t *testing.T, sched eval.Box3Scheduler, langs eval.LanguageManager, token string) string {
	t.Helper()
	reg := NewClientRegistry()
	if err := reg.Add(token, "platform-test", ""); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(reg.Auth(NewGraderServer(sched, langs).Handler()))
	t.Cleanup(srv.Close)
	return srv.URL
}

func callRunBox3(url, token string) error {
	client := NewGraderClient(http.DefaultClient, url, token)
	_, err := client.RunBox3(context.Background(), &eval.Box3Request{Command: []string{"/bin/true"}}, 0)
	return err
}

func TestAuthValidTokenServed(t *testing.T) {
	sched := &recordingSched{}
	url := startTestGrader(t, sched, nil, "good-token")
	if err := callRunBox3(url, "good-token"); err != nil {
		t.Fatalf("valid token was rejected: %v", err)
	}
	if !sched.ran {
		t.Fatal("valid request did not reach execution")
	}
}

func TestAuthInvalidTokenRejectedBeforeExecution(t *testing.T) {
	sched := &recordingSched{}
	url := startTestGrader(t, sched, nil, "good-token")
	err := callRunBox3(url, "wrong-token")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for wrong token, got %v", err)
	}
	if sched.ran {
		t.Fatal("SECURITY: box executed despite an unregistered token")
	}
}

func TestAuthMissingTokenRejectedBeforeExecution(t *testing.T) {
	sched := &recordingSched{}
	url := startTestGrader(t, sched, nil, "good-token")
	// Raw request with no Authorization header.
	resp, err := http.Post(url+"/run/box3", "application/json", strings.NewReader(`{"request":{"Command":["/bin/true"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 for missing token, got %d", resp.StatusCode)
	}
	if sched.ran {
		t.Fatal("SECURITY: box executed despite a missing token")
	}
}

func TestLanguagesRefusesBuildMismatch(t *testing.T) {
	langs := stubLangs{versions: map[string]string{"cpp": "14"}}
	url := startTestGrader(t, &recordingSched{}, langs, "tok")

	// Same binary on both ends: succeeds and carries the versions through.
	got, err := NewGraderClient(nil, url, "tok").languageVersions(context.Background())
	if err != nil || got["cpp"] != "14" {
		t.Fatalf("same-build fetch failed: %v %v", got, err)
	}

	// A grader on another build: refused before any run can be sent.
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encode(w, languagesResp{Build: "someone-else", Versions: langs.versions})
	}))
	t.Cleanup(other.Close)
	if _, err := NewGraderClient(nil, other.URL, "tok").languageVersions(context.Background()); err == nil || !strings.Contains(err.Error(), "someone-else") {
		t.Fatalf("mismatched build was accepted: %v", err)
	}
}
