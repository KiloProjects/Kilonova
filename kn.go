package main

import (
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/KiloProjects/Kilonova/api"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/grader"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/KiloProjects/Kilonova/internal/rclient"
	"github.com/KiloProjects/Kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Kilonova(_ *cli.Context) error {
	// Print welcome message
	fmt.Printf("Starting Kilonova %s\n", logic.Version)

	dataDir := config.C.Common.DataDir
	logDir := config.C.Common.LogDir
	debug := config.C.Common.Debug

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
	dbc, err := db.New(config.C.Database.String(), c)
	if err != nil {
		return err
	}
	log.Println("Connected to DB")

	// Data Manager setup
	manager, err := datamanager.NewManager(dataDir)
	if err != nil {
		return err
	}

	kn, err := logic.New(dbc, manager, c, debug)
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

	r.Use(middleware.Compress(flate.BestCompression))
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
	API := api.New(kn)
	grader := grader.NewHandler(ctx, kn)

	r.Mount("/api", API.Router())
	r.Mount("/", web.NewWeb(kn).Router())

	grader.Start()

	// for graceful setup and shutdown
	server := &http.Server{
		Addr:    "127.0.0.1:8070",
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
			cancel()
		}
	}()

	fmt.Println("Successfully started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	defer func() {
		fmt.Println("Shutting Down")
		if err := server.Shutdown(ctx); err != nil {
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
