package scheduler

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestGraderClientRetriesUntilGraderIsUp starts the grader only after the
// client's first attempt has failed with connection refused.
func TestGraderClientRetriesUntilGraderIsUp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	go func() {
		time.Sleep(700 * time.Millisecond)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return
		}
		http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"ok":true}`))
		}))
	}()

	c := NewGraderClient(nil, "http://"+addr, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var out struct{ OK bool }
	if err := c.call(ctx, http.MethodGet, "/x", nil, &out); err != nil || !out.OK {
		t.Fatalf("call = %v, out = %+v", err, out)
	}
}
