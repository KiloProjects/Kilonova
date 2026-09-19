package prometheus

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InitMetrics serves /metrics on listen (host:port); empty disables the exporter.
func InitMetrics(ctx context.Context, listen string) {
	if listen == "" {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	go func() {
		s := &http.Server{
			Addr:              listen,
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
