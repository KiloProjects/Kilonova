package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/integrations/maxmind"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func Kilonova() error {

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova", slog.String("version", kilonova.Version))

	if config.Common.Debug {
		slog.WarnContext(ctx, "Debug mode activated, expect worse performance")
	}

	maxmind.Initialize(ctx)

	base, err := sudoapi.InitializeBaseAPI(context.Background())
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize BaseAPI", slog.Any("err", err))
		return err
	}
	base.Start(ctx)
	defer base.Close()

	// Initialize components
	grader, err := grader.NewHandler(ctx, base)
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize grader", slog.Any("err", err))
		return err
	}
	defer grader.Close()

	go func() {
		err := grader.Start()
		if err != nil {
			slog.ErrorContext(ctx, "Could not start grader", slog.Any("err", err))
		}
	}()

	if err := base.ResetWaitingSubmissions(ctx); err != nil {
		slog.WarnContext(ctx, "Couldn't reset initial working submissions:", slog.Any("err", err))
	}

	// for graceful setup and shutdown
	server := webV1(true, base)

	go launchProfiler()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Error initializing web server", slog.Any("err", err))
			cancel()
		}
	}()

	slog.InfoContext(ctx, "Successfully started")

	defer func() {
		slog.InfoContext(ctx, "Shutting Down")
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "Error shutting down", slog.Any("err", err))
		}
	}()

	<-ctx.Done()

	return nil
}

func initLogger(debug bool) {
	core := kilonova.GetZapCore(debug, os.Stdout)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)

	slog.SetDefault(slog.New(zapslog.NewHandler(core, zapslog.WithCaller(true))))
}

func launchProfiler() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return http.ListenAndServe(":6080", mux)
}

var (
	listenHost = config.GenFlag[string]("server.listen.host", "localhost", "Host to listen to")
	listenPort = config.GenFlag[int]("server.listen.port", 8070, "Port to listen on")
)

// initialize webserver for public api+web
func webV1(templWeb bool, base *sudoapi.BaseAPI) *http.Server {
	// Initialize router
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{config.Common.HostPrefix}, // TODO: Do better
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	//r.Use(middleware.Timeout(1 * time.Minute))
	/*
		r.Use(middleware.Compress(flate.DefaultCompression))
		r.Use(middleware.RequestID)
	*/

	r.Mount("/api", api.New(base).Handler())
	r.Mount("/assets", api.NewAssets(base).AssetsRouter())

	if templWeb {
		r.Mount("/", web.NewWeb(base).Handler())
	}

	return &http.Server{
		Addr:              net.JoinHostPort(listenHost.Value(), strconv.Itoa(listenPort.Value())),
		Handler:           r,
		ReadHeaderTimeout: 1 * time.Minute,
	}
}
