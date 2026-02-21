package prometheus

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	enabled = config.GenFlag[bool]("integrations.prometheus.enabled", false, "Enable Prometheus metrics")
	port    = config.GenFlag[int]("integrations.prometheus.port", 8071, "Prometheus metrics port")
)

func InitMetrics() {
	if !enabled.Value() {
		return
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port.Value()), mux); err != nil {
			slog.Error("Error with Prometheus metrics", slog.Any("err", err))
		}
	}()
}
