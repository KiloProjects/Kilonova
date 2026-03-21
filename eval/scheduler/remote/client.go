// Package remote implements eval.BoxScheduler as a client that calls a remote
// knbox service over HTTP. It is used by the monolith's grader handler when a
// RemoteBoxScheduler URL is configured.
//
// Language metadata queries (Language, Languages, LanguageFromFilename) are
// served locally from the same language registry that the knbox service uses —
// both sides compile the same eval/language package, so no serialization of
// language definitions is needed.
//
// Only sandbox execution (RunBox2, RunMultibox) and LanguageVersions make
// network calls to the knbox service.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scheduler/wire"
)

var _ eval.BoxScheduler = (*Client)(nil)

// Client implements eval.BoxScheduler by delegating execution to a remote
// knbox service. Language queries use the local language registry.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authToken  string
	langs      map[string]language.GraderLang
}

// NewClient creates a remote BoxScheduler client.
// baseURL should be the base URL of the knbox service (e.g. "http://box-host:8091").
// authToken is the shared secret configured on the server; leave empty to disable auth.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		httpClient: &http.Client{},
		baseURL:    strings.TrimRight(baseURL, "/"),
		authToken:  authToken,
		langs:      scheduler.SupportedLanguages(context.Background()),
	}
}

func (c *Client) RunBox2(ctx context.Context, req *eval.Box2Request, memQuota int64) (*eval.Box2Response, error) {
	var resp wire.RunBoxResponse
	if err := c.post(ctx, "/knbox.v1.BoxSchedulerService/RunBox", &wire.RunBoxRequest{
		Request:  wire.ToBox2Request(req),
		MemQuota: memQuota,
	}, &resp); err != nil {
		return nil, err
	}
	return wire.FromBox2Response(resp.Response), nil
}

func (c *Client) RunMultibox(ctx context.Context, req *eval.MultiboxRequest, managerMemQuota, individualMemQuota int64) (*eval.Box2Response, []*eval.RunStats, error) {
	var resp wire.RunMultiboxResponse
	if err := c.post(ctx, "/knbox.v1.BoxSchedulerService/RunMultibox",
		wire.ToRunMultiboxRequest(req, managerMemQuota, individualMemQuota), &resp); err != nil {
		return nil, nil, err
	}
	userStats := make([]*eval.RunStats, len(resp.UserStats))
	for i, us := range resp.UserStats {
		userStats[i] = wire.FromRunStats(us)
	}
	return wire.FromBox2Response(resp.ManagerResponse), userStats, nil
}

// Close is a no-op: the remote service manages its own lifecycle.
func (c *Client) Close(_ context.Context) error { return nil }

func (c *Client) Language(name string) language.GraderLang { return c.langs[name] }

func (c *Client) Languages() map[string]language.GraderLang { return c.langs }

func (c *Client) LanguageFromFilename(filename string) language.GraderLang {
	return scheduler.LanguageFromFilename(c.langs, filename)
}

func (c *Client) LanguageVersions(ctx context.Context) map[string]string {
	var resp wire.GetLanguageVersionsResponse
	if err := c.get(ctx, "/knbox.v1.BoxSchedulerService/GetLanguageVersions", &resp); err != nil {
		return nil
	}
	return resp.Versions
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	return c.doRequest(req, out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	return c.doRequest(req, out)
}

func (c *Client) doRequest(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("knbox error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
