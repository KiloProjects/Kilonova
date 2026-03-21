// Package server exposes an eval.BoxScheduler as a JSON-over-HTTP service.
// It is used by the knbox binary to allow the monolith's grader to offload
// sandbox execution to a separate Linux host.
//
// Endpoints follow the path convention /knbox.v1.BoxSchedulerService/<Method>,
// mirroring the proto service definition in eval/scheduler/proto/scheduler.proto.
// This makes a future migration to generated ConnectRPC code straightforward.
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/scheduler/wire"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server exposes an eval.BoxScheduler over HTTP as a JSON API.
type Server struct {
	scheduler eval.BoxScheduler
	authToken string
	logger    *slog.Logger
}

// New creates a new Server. If authToken is non-empty, every request must carry
// an "Authorization: Bearer <token>" header matching that value.
func New(scheduler eval.BoxScheduler, authToken string, logger *slog.Logger) *Server {
	return &Server{scheduler: scheduler, authToken: authToken, logger: logger}
}

// Handler returns an http.Handler that serves the BoxScheduler API.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	if s.authToken != "" {
		r.Use(s.requireAuth)
	}
	r.Post("/knbox.v1.BoxSchedulerService/RunBox", s.handleRunBox)
	r.Post("/knbox.v1.BoxSchedulerService/RunMultibox", s.handleRunMultibox)
	r.Get("/knbox.v1.BoxSchedulerService/GetLanguageVersions", s.handleGetLanguageVersions)
	r.Get("/knbox.v1.BoxSchedulerService/Ping", s.handlePing)
	return r
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != s.authToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleRunBox(w http.ResponseWriter, r *http.Request) {
	var req wire.RunBoxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := s.scheduler.RunBox2(r.Context(), wire.FromBox2Request(req.Request), req.MemQuota)
	if err != nil {
		s.logger.WarnContext(r.Context(), "RunBox2 error", slog.Any("err", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, &wire.RunBoxResponse{Response: wire.ToBox2Response(resp)})
}

func (s *Server) handleRunMultibox(w http.ResponseWriter, r *http.Request) {
	var req wire.RunMultiboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	evalReq, managerQuota, individualQuota := wire.FromRunMultiboxRequest(&req)
	managerResp, userStats, err := s.scheduler.RunMultibox(r.Context(), evalReq, managerQuota, individualQuota)
	if err != nil {
		s.logger.WarnContext(r.Context(), "RunMultibox error", slog.Any("err", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wireUserStats := make([]*wire.RunStats, len(userStats))
	for i, us := range userStats {
		wireUserStats[i] = wire.ToRunStats(us)
	}
	writeJSON(w, &wire.RunMultiboxResponse{
		ManagerResponse: wire.ToBox2Response(managerResp),
		UserStats:       wireUserStats,
	})
}

func (s *Server) handleGetLanguageVersions(w http.ResponseWriter, r *http.Request) {
	versions := s.scheduler.LanguageVersions(r.Context())
	writeJSON(w, &wire.GetLanguageVersionsResponse{Versions: versions})
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, &wire.PingResponse{Version: "1"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("Error writing JSON response", slog.Any("err", err))
	}
}
