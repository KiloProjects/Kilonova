package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/email"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Kilonova() error {

	// Print welcome message
	zap.S().Infof("Starting Kilonova %s", kilonova.Version)

	dataDir := config.Common.DataDir
	debug := config.Common.Debug

	if debug {
		zap.S().Warn("Debug mode activated, expect worse performance")
	}

	// Data directory setup
	if !path.IsAbs(dataDir) {
		return &kilonova.Error{Code: kilonova.EINVALID, Message: "dataDir not absolute"}
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("Could not create data dir: %w", err)
	}

	// DB Setup
	db, err := db.AppropriateDB(context.Background(), config.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	zap.S().Info("Connected to DB")

	// Data Store setup
	manager, err := datastore.NewManager(dataDir)
	if err != nil {
		return err
	}

	// Setup mailer
	mailer, err := email.NewMailer()
	if err != nil {
		return err
	}

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

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	grader := grader.NewHandler(ctx, db, manager, config.Common.Debug)

	r.Mount("/api", api.New(db, manager, mailer).Handler())

	r.Mount("/", web.NewWeb(config.Common.Debug, db, manager, mailer).Handler())

	go func() {
		err := grader.Start()
		if err != nil {
			zap.S().Error(err)
		}
	}()

	// for graceful setup and shutdown
	server := &http.Server{
		Addr:    net.JoinHostPort("localhost", strconv.Itoa(config.Common.Port)),
		Handler: r,
	}

	go launchProfiler()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
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

	select {
	case <-ctx.Done():
	}

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
