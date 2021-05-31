package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/logic"
	"github.com/KiloProjects/kilonova/web"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func executeExperiment(dm kilonova.DataStore, db kilonova.DB) error {
	pb, err := db.Problem(context.Background(), 1)
	if err != nil {
		return err
	}
	pb1, err := db.Problem(context.Background(), 2)
	if err != nil {
		return err
	}

	rd, err := kilonova.GenKNA([]*kilonova.Problem{pb, pb1}, db, dm)
	if err != nil {
		return err
	}

	b, err := io.ReadAll(rd)
	if err != nil {
		return err
	}

	if err := os.WriteFile("./plm.db", b, 0777); err != nil {
		return err
	}

	d, err := os.ReadFile("./plm.db")
	spew.Dump(kilonova.ReadKNA(bytes.NewReader(d)))
	return nil
}

func Kilonova() error {
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
		return &kilonova.Error{Code: kilonova.EINVALID, Message: "logDir not absolute"}
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("Could not create log dir: %w", err)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// DB Setup
	db, err := db.AppropriateDB(context.Background(), config.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Println("Connected to DB")

	// Data Store setup
	manager, err := datastore.NewManager(dataDir)
	if err != nil {
		return err
	}

	//return executeExperiment(manager, db)

	kn, err := logic.New(db, manager, debug)
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
		logg := log.New(&lumberjack.Logger{
			Filename: path.Join(logDir, "access.log"),
		}, "", 0)

		r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
			Logger:  logg,
			NoColor: true,
		}))
	*/

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	grader := grader.NewHandler(ctx, kn, db)

	r.Mount("/api", api.New(kn, db).Handler())
	r.Mount("/cdn", http.StripPrefix("/cdn/", &web.CDN{CDN: manager}))
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
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	defer func() {
		fmt.Println("Shutting Down")
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Println(err)
		}
	}()

	select {
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
