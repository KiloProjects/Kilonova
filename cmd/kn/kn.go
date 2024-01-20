package main

import (
	"context"
	"errors"
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
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
)

var graderFeature = config.GenFlag("feature.grader.enabled", true, "Grader")

func Kilonova() error {

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	// Print welcome message
	zap.S().Infof("Starting Kilonova %s", kilonova.Version)

	if config.Common.Debug {
		zap.S().Warn("Debug mode activated, expect worse performance")
	}

	base, err := sudoapi.InitializeBaseAPI(context.Background())
	if err != nil {
		zap.S().Fatal(err)
	}
	base.Start(ctx)
	defer base.Close()

	// Initialize components
	if graderFeature.Value() { // TODO: Hot stopping/starting grader
		grader, err := grader.NewHandler(ctx, base)
		if err != nil {
			zap.S().Fatal(err)
		}
		defer grader.Close()

		go func() {
			err := grader.Start()
			if err != nil {
				zap.S().Error(err)
			}
		}()
	}

	if err := base.ResetWaitingSubmissions(ctx); err != nil {
		zap.S().Warn("Couldn't reset initial working submissions:", err)
	}

	// for graceful setup and shutdown
	server := webV1(true, base)

	go launchProfiler()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.S().Error(err)
			cancel()
		}
	}()

	zap.S().Info("Successfully started")

	defer func() {
		zap.S().Info("Shutting Down")
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Error(err)
		}
	}()

	<-ctx.Done()

	return nil
}

func initLogger(debug bool) {
	core := kilonova.GetZapCore(debug, true, os.Stdout)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)
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

	r.Use(middleware.RealIP)
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
		Addr:              net.JoinHostPort("localhost", strconv.Itoa(config.Common.Port)),
		Handler:           r,
		ReadHeaderTimeout: 1 * time.Minute,
	}
}
