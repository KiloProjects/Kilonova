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
	"go.uber.org/zap/zapcore"
)

func Kilonova() error {

	// Print welcome message
	zap.S().Infof("Starting Kilonova %s", kilonova.Version)

	if config.Common.Debug {
		zap.S().Warn("Debug mode activated, expect worse performance")
	}

	printFeatureSet()

	base, err := sudoapi.InitializeBaseAPI(context.Background())
	if err != nil {
		zap.S().Fatal(err)
	}
	defer base.Close()

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	if config.Features.Grader {
		grader := grader.NewHandler(ctx, base)

		go func() {
			err := grader.Start()
			if err != nil {
				zap.S().Error(err)
			}
		}()
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
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	defer func() {
		zap.S().Info("Shutting Down")
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Error(err)
		}
	}()

	<-ctx.Done()

	return nil
}

func initLogger(logDir string, debug bool) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	var encConf zapcore.EncoderConfig
	if debug {
		encConf = zap.NewDevelopmentEncoderConfig()
	} else {
		encConf = zap.NewProductionEncoderConfig()
	}
	encConf.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339))
	})
	encConf.EncodeLevel = zapcore.CapitalColorLevelEncoder

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encConf), zapcore.AddSync(os.Stdout), level)
	logg := zap.New(core, zap.AddCaller())

	zap.ReplaceGlobals(logg)

	return nil
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

	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(corsConfig.Handler)

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(20 * time.Second))
	/*
		r.Use(middleware.Compress(flate.DefaultCompression))
		r.Use(middleware.RequestID)
	*/

	r.Mount("/api", api.New(base).Handler())

	if templWeb {
		r.Mount("/", web.NewWeb(config.Common.Debug, base).Handler())
	}

	return &http.Server{
		Addr:    net.JoinHostPort("localhost", strconv.Itoa(config.Common.Port)),
		Handler: r,
	}
}

func printFeatureSet() {
	isEnabled := func(x bool) string {
		if x {
			return "ENABLED"
		}
		return "DISABLED"
	}

	zap.S().Info("Feature list:")
	zap.S().Infof("Grader: %s", isEnabled(config.Features.Grader))
	zap.S().Infof("Signup: %s", isEnabled(config.Features.Signup))
	zap.S().Infof("Pastes: %s", isEnabled(config.Features.Pastes))
}
