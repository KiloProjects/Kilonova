package main

import (
	"compress/flate"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/cookie"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/grader"
	"github.com/KiloProjects/Kilonova/internal/version"
	"github.com/KiloProjects/Kilonova/server"
	"github.com/KiloProjects/Kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "github.com/lib/pq"
)

// go:generate pkger

var (
	logDir     = flag.String("logDir", "/data/knLogs", "Directory to write logs to")
	debug      = flag.Bool("debug", false, "Debug mode")
	dataDir    = flag.String("data", "/data", "Data directory")
	evalSocket = flag.String("evalSocket", "/tmp/kiloeval.sock", "Path to the eval socket, must be the same as the `socketPath` flag in KiloEval")
)

func main() {
	flag.Parse()

	// Print welcome message
	fmt.Printf("Starting Kilonova %s\n", version.Version)

	// Logger setup
	if !path.IsAbs(*logDir) {
		log.Fatal("logDir not absolute")
	}
	if err := os.MkdirAll(*logDir, 0755); err != nil {
		log.Fatalf("Could not create log dir: %v", err)
	}
	logg := log.New(&lumberjack.Logger{
		Filename: path.Join(*logDir, "access.log"),
	}, "", 0)

	// Session Cookie setup
	cookie.Initialize(*dataDir)

	// DB Setup
	dsn := "sslmode=disable host=/var/run/postgresql user=alexv dbname=kilonova"
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Could not connect to DB: %v", err)
	}

	logg.Println("Connected to DB")

	ndb, err := db.Prepare(context.Background(), sqlDB)
	if err != nil {
		log.Fatal(err)
	}

	// Data Manager setup
	manager := datamanager.NewManager(*dataDir)

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

	r.Use(middleware.Compress(flate.DefaultCompression))
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
	API := server.NewAPI(ctx, manager, logg, ndb)
	grader := grader.NewHandler(ctx, ndb, manager, logg)

	r.Mount("/api", API.GetRouter())
	r.Mount("/", web.NewWeb(manager, ndb, logg, *debug).GetRouter())

	grader.Start(*evalSocket)

	// for graceful setup and shutdown
	server := &http.Server{Addr: "127.0.0.1:8070", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			cancel()
			fmt.Println(err)
		}
	}()

	fmt.Println("Successfully started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	defer func() {
		fmt.Println("Shutting Down")
		if err := server.Shutdown(ctx); err != nil {
			fmt.Println(err)
		}
	}()

	select {
	case <-stop:
		cancel()
	case <-ctx.Done():
	}
}
