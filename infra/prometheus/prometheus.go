package prometheus

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	enabled = config.GenFlag[bool]("integrations.prometheus.enabled", false, "Enable Prometheus metrics")
	port    = config.GenFlag[int]("integrations.prometheus.port", 8071, "Prometheus metrics port")
)

func InitMetrics(ctx context.Context) {
	if !enabled.Value() {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	go func() {
		s := &http.Server{
			Addr:              fmt.Sprintf(":%d", port.Value()),
			Handler:           mux,
			ReadHeaderTimeout: 1 * time.Minute,
		}
		context.AfterFunc(ctx, func() {
			s.Shutdown(context.WithoutCancel(ctx))
		})

		if err := s.ListenAndServe(); err != nil {
			slog.ErrorContext(ctx, "Error with Prometheus metrics", slog.Any("err", err))
		}
	}()
}
