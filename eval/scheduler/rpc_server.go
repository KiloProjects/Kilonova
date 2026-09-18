package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
)

// Wire envelopes. eval.* structs are encoded directly; these only add the
// quotas that Box3Scheduler takes as extra arguments. Both sides share these
// types, so the wire shape is a Go type, not a doc.
type runBox3Req struct {
	Request  *eval.Box3Request `json:"request"`
	MemQuota int64             `json:"mem_quota"`
}

type runMultibox3Req struct {
	Request            *eval.Multibox3Request `json:"request"`
	ManagerMemQuota    int64                  `json:"manager_mem_quota"`
	IndividualMemQuota int64                  `json:"individual_mem_quota"`
}

type runMultibox3Resp struct {
	ManagerResponse *eval.Box3Response `json:"manager_response"`
	UserStats       []*eval.RunStats   `json:"user_stats"`
}

type languagesResp struct {
	Build    string            `json:"build"`
	Versions map[string]string `json:"versions"`
}

// maxBodySize bounds a control-plane request body. Legit requests are kilobytes.
const maxBodySize = 1 << 20

// BuildID identifies this binary: platform and grader must match, since the
// eval.* struct shape is the wire contract.
var BuildID = sync.OnceValue(func() string {
	id := kilonova.Version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return id
	}
	var rev, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev != "" {
		id += "+" + rev
	}
	// ponytail: two dirty trees at different edits both pass. Dev-only.
	return id + dirty
})

// GraderServer exposes a local BoxManager + LanguageManager as JSON-over-HTTP.
// It holds no platform credentials; every handler just runs the in-process grader.
type GraderServer struct {
	sched eval.Box3Scheduler
	langs eval.LanguageManager
}

func NewGraderServer(sched eval.Box3Scheduler, langs eval.LanguageManager) *GraderServer {
	return &GraderServer{sched: sched, langs: langs}
}

// Handler returns the control-plane routes. Mount it behind ClientRegistry.Auth.
func (s *GraderServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /run/box3", s.runBox3)
	mux.HandleFunc("POST /run/multibox3", s.runMultibox3)
	mux.HandleFunc("GET /languages", s.languages)
	return mux
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize)).Decode(v); err != nil {
		http.Error(w, "bad request body: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func encode(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *GraderServer) runBox3(w http.ResponseWriter, r *http.Request) {
	var in runBox3Req
	if !decode(w, r, &in) {
		return
	}
	resp, err := s.sched.RunBox3(r.Context(), in.Request, in.MemQuota)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encode(w, resp)
}

func (s *GraderServer) runMultibox3(w http.ResponseWriter, r *http.Request) {
	var in runMultibox3Req
	if !decode(w, r, &in) {
		return
	}
	resp, stats, err := s.sched.RunMultibox3(r.Context(), in.Request, in.ManagerMemQuota, in.IndividualMemQuota)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	encode(w, runMultibox3Resp{ManagerResponse: resp, UserStats: stats})
}

func (s *GraderServer) languages(w http.ResponseWriter, r *http.Request) {
	encode(w, languagesResp{Build: BuildID(), Versions: s.langs.LanguageVersions(r.Context())})
}

// --- auth ---

// ClientRegistry maps grader-minted bearer tokens to a named platform client.
// Priority is reserved (documented but unconsumed by this change).
type ClientRegistry struct {
	byToken map[string]ClientIdentity
}

type ClientIdentity struct {
	Name     string
	Priority string
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{byToken: make(map[string]ClientIdentity)}
}

// Add registers a client token. An empty token is rejected so a misconfigured
// registry can never authenticate a caller that sends no token.
func (r *ClientRegistry) Add(token, name, priority string) error {
	if token == "" {
		return errors.New("client registry: empty token for client " + name)
	}
	r.byToken[token] = ClientIdentity{Name: name, Priority: priority}
	return nil
}

type clientNameKey struct{}

// ClientName returns the authenticated client name attached by Auth.
func ClientName(ctx context.Context) string {
	name, _ := ctx.Value(clientNameKey{}).(string)
	return name
}

// Auth is the one bearer-token check for every path on the grader listener:
// control plane and /scratch data plane alike.
func (r *ClientRegistry) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		ident, ok := r.byToken[token]
		if !ok {
			http.Error(w, "unregistered token", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(req.Context(), clientNameKey{}, ident.Name)
		slog.DebugContext(ctx, "Authenticated grader request", slog.String("client", ident.Name), slog.String("path", req.URL.Path))
		next.ServeHTTP(w, req.WithContext(ctx))
	})
}
