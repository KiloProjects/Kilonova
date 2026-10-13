package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
)

var _ eval.Box3Scheduler = (*GraderClient)(nil)

// retryAttempts bounds how often a request is retried on transport errors;
// with backoff capped at 10s this rides out ~35s of grader downtime.
const retryAttempts = 8

// ErrUnauthenticated is returned when the grader rejects our bearer token.
var ErrUnauthenticated = errors.New("grader rejected bearer token")

// GraderClient is the platform-side eval.Box3Scheduler backed by a remote grader
// over JSON-over-HTTP. It carries the grader-minted bearer token on every request.
type GraderClient struct {
	client  *http.Client
	baseURL string
	token   string
}

func NewGraderClient(httpClient *http.Client, baseURL, token string) *GraderClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GraderClient{client: httpClient, baseURL: strings.TrimRight(baseURL, "/"), token: token}
}

// call does one JSON request/response. A nil in means GET; out is decoded from
// a 2xx body. Any other status becomes an error carrying status and body.
func (c *GraderClient) call(ctx context.Context, method, path string, in, out any) error {
	var buf []byte
	if in != nil {
		var err error
		if buf, err = json.Marshal(in); err != nil {
			return err
		}
	}
	resp, err := c.do(ctx, method, path, buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthenticated
	}
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("grader %s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(msg)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// do sends the request, retrying transport errors (e.g. the grader is
// restarting) with exponential backoff until retryAttempts or ctx runs out.
func (c *GraderClient) do(ctx context.Context, method, path string, buf []byte) (*http.Response, error) {
	delay := 500 * time.Millisecond
	for attempt := 1; ; attempt++ {
		var body io.Reader
		if buf != nil {
			body = bytes.NewReader(buf)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		if buf != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.client.Do(req)
		if err == nil || attempt == retryAttempts || ctx.Err() != nil {
			return resp, err
		}
		slog.WarnContext(ctx, "Grader request failed, retrying", slog.String("path", path), slog.Int("attempt", attempt), slog.Any("err", err))
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(delay):
		}
		delay = min(delay*2, 10*time.Second)
	}
}

func (c *GraderClient) RunBox3(ctx context.Context, req *eval.Box3Request, memQuota int64) (*eval.Box3Response, error) {
	var out eval.Box3Response
	if err := c.call(ctx, http.MethodPost, "/run/box3", runBox3Req{Request: req, MemQuota: memQuota}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *GraderClient) RunMultibox3(ctx context.Context, req *eval.Multibox3Request, managerMemQuota, individualMemQuota int64) (*eval.Box3Response, []*eval.RunStats, error) {
	var out runMultibox3Resp
	if err := c.call(ctx, http.MethodPost, "/run/multibox3", runMultibox3Req{
		Request: req, ManagerMemQuota: managerMemQuota, IndividualMemQuota: individualMemQuota,
	}, &out); err != nil {
		return nil, nil, err
	}
	return out.ManagerResponse, out.UserStats, nil
}

// Close is a client-side no-op: the grader is shared across platform instances,
// so one platform shutting down must not drain the grader's boxes.
func (c *GraderClient) Close(ctx context.Context) error { return nil }

// languageVersions fetches the grader's supported language -> version map and
// refuses a grader built from a different revision than this platform.
func (c *GraderClient) languageVersions(ctx context.Context) (map[string]string, error) {
	var out languagesResp
	if err := c.call(ctx, http.MethodGet, "/languages", nil, &out); err != nil {
		return nil, err
	}
	// Compare only the release version: images built without .git carry no
	// "+revision" suffix. NOTE: same version at different commits passes;
	// stamp the revision via -ldflags in the Dockerfile if that bites.
	grader, _, _ := strings.Cut(out.Build, "+")
	platform, _, _ := strings.Cut(BuildID(), "+")
	if grader != platform && !kilonova.DebugMode() {
		return nil, fmt.Errorf("grader build %q does not match platform build %q; deploy both from the same build", out.Build, BuildID())
	}
	return out.Versions, nil
}
