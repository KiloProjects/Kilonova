package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/KiloProjects/kilonova/eval"
	graderv1 "github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1"
	"github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1/graderv1connect"
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

func startTestGrader(t *testing.T, sched eval.Box3Scheduler, token string) string {
	t.Helper()
	reg := NewClientRegistry()
	if err := reg.Add(token, "platform-test", ""); err != nil {
		t.Fatal(err)
	}
	_, handler := NewGraderServer(sched, nil).Handler(reg)
	mux := http.NewServeMux()
	path, h := "/", handler
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
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
	url := startTestGrader(t, sched, "good-token")
	if err := callRunBox3(url, "good-token"); err != nil {
		t.Fatalf("valid token was rejected: %v", err)
	}
	if !sched.ran {
		t.Fatal("valid request did not reach execution")
	}
}

func TestAuthInvalidTokenRejectedBeforeExecution(t *testing.T) {
	sched := &recordingSched{}
	url := startTestGrader(t, sched, "good-token")
	err := callRunBox3(url, "wrong-token")
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated for wrong token, got %v", err)
	}
	if sched.ran {
		t.Fatal("SECURITY: box executed despite an unregistered token")
	}
}

func TestAuthMissingTokenRejectedBeforeExecution(t *testing.T) {
	sched := &recordingSched{}
	url := startTestGrader(t, sched, "good-token")
	// Raw generated client with no bearer interceptor => no Authorization header.
	client := graderv1connect.NewGraderServiceClient(http.DefaultClient, url)
	_, err := client.RunBox3(context.Background(), connect.NewRequest(&graderv1.RunBox3Request{
		Request: &graderv1.Box3Request{Command: []string{"/bin/true"}},
	}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated for missing token, got %v", err)
	}
	if sched.ran {
		t.Fatal("SECURITY: box executed despite a missing token")
	}
}
