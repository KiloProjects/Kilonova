package profiler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"
)

// StartProfiler (synchonously) initializes the pprof server on a given port.
// On context cancellation, the profiler is stopped.
func StartProfiler(ctx context.Context, port int) error {

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 1 * time.Minute,
	}
	context.AfterFunc(ctx, func() {
		err := server.Shutdown(context.WithoutCancel(ctx))
		if err != nil {
			slog.ErrorContext(ctx, "Error shutting down pprof", slog.Any("err", err))
		}
	})

	return server.ListenAndServe()
}
