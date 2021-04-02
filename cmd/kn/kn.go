package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/internal/rclient"
	"github.com/KiloProjects/kilonova/psql"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Kilonova(_ *cli.Context) error {
	// Print welcome message
	fmt.Printf("Starting Kilonova %s\n", kilonova.Version)

	dataDir := config.Common.DataDir
	logDir := config.Common.LogDir
	debug := config.Common.Debug

	if debug {
		log.Println("WARNING: debug mode activated, expect worse performance")
	}

	// Data directory setup
	if !path.IsAbs(dataDir) {
		return errors.New("logDir not absolute")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("Could not create log dir: %w", err)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logg := log.New(&lumberjack.Logger{
		Filename: path.Join(logDir, "access.log"),
	}, "", 0)

	// Redis setup
	c, err := rclient.New()
	if err != nil {
		return err
	}

	// DB Setup
	db, err := psql.New(config.Database.String())
	if err != nil {
		return err
	}
	log.Println("Connected to DB")

	// Data Store setup
	manager, err := datastore.NewManager(dataDir)
	if err != nil {
		return err
	}

	kn, err := logic.New(db, manager, c, debug)
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

	//r.Use(middleware.Compress(flate.DefaultCompression))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(20 * time.Second))
	r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger:  logg,
		NoColor: true,
	}))

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	grader := grader.NewHandler(ctx, kn, db)

	r.Mount("/api", api.New(kn, db).Handler())
	r.Mount("/cdn", http.StripPrefix("/cdn/", web.NewCDN(kn, db).Handler()))
	r.Mount("/", web.NewWeb(kn, db).Handler())

	go func() {
		err := grader.Start()
		if err != nil {
			log.Println(err)
		}
	}()

	// for graceful setup and shutdown
	server := &http.Server{
		Addr:    "localhost:8070",
		Handler: r,
	}

	go launchProfiler()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
			cancel()
		}
	}()

	log.Println("Successfully started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		fmt.Println("Shutting Down")
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Println(err)
		}
	}()

	select {
	case <-stop:
		signal.Stop(stop)
		cancel()
	case <-ctx.Done():
	}

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
