package scratch

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/google/uuid"
)

var _ eval.Scratch = (*httpScratch)(nil)

// httpScratch implements eval.Scratch against a remote grader's /scratch
// endpoint. Bytes stream over the same TLS+token channel as the RPC control
// plane, so there is no separate sshd/data-plane service to run or secure.
// http.Client handles connection pooling and keep-alive for us.
type httpScratch struct {
	client  *http.Client
	baseURL string // grader scratch base, e.g. https://grader:9000/scratch
	token   string // grader-minted bearer token (same one the RPC client carries)
}

// NewHTTP builds a remote scratch over the grader's /scratch endpoint. baseURL
// is that endpoint (typically the RPC endpoint + "/scratch"); token is the
// grader-minted bearer token.
func NewHTTP(client *http.Client, baseURL, token string) (eval.Scratch, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("parse scratch base url: %w", err)
	}
	return &httpScratch{client: client, baseURL: strings.TrimRight(baseURL, "/"), token: token}, nil
}

func (s *httpScratch) do(method, identifier string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, s.baseURL+"/"+identifier, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	return s.client.Do(req)
}

func (s *httpScratch) SaveFile(r io.Reader) (string, error) {
	identifier := uuid.Must(uuid.NewV7()).String()
	// PUT returns only after the grader has flushed the file to its disk, so the
	// identifier is durable before any RunBox3 references it.
	resp, err := s.do(http.MethodPut, identifier, r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("scratch put %s: %s", identifier, resp.Status)
	}
	return identifier, nil
}

func (s *httpScratch) ReadFile(identifier string) (io.ReadCloser, error) {
	resp, err := s.do(http.MethodGet, identifier, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fs.ErrNotExist
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("scratch get %s: %s", identifier, resp.Status)
	}
	return resp.Body, nil
}

func (s *httpScratch) DeleteFile(identifier string) error {
	resp, err := s.do(http.MethodDelete, identifier, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("scratch delete %s: %s", identifier, resp.Status)
	}
	return nil
}
