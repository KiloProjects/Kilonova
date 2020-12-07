package main

import (
	"compress/flate"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/KiloProjects/Kilonova/api"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/KiloProjects/Kilonova/internal/cookie"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/grader"
	"github.com/KiloProjects/Kilonova/internal/logic"
	"github.com/KiloProjects/Kilonova/internal/version"
	"github.com/KiloProjects/Kilonova/web"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"

	_ "github.com/lib/pq"
)

//go:generate pkger

/*
var (
	logDir  = flag.String("logDir", "/data/knLogs", "Directory to write logs to")
	debug   = flag.Bool("debug", false, "Debug mode")
	dataDir = flag.String("data", "/data", "Data directory")
)*/

var (
	confDir = flag.String("config", "./config.toml", "Config path")
)

func main() { // TODO: finish this
	flag.Parse()
	if err := config.Load(*confDir); err != nil {
		panic(err)
	}

	spew.Dump(config.C)

	app := &cli.App{
		Name:    "Kilonova",
		Usage:   "Control the platform",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Config path",
				Value: "./config.toml",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "main",
				Usage:  "Website/API",
				Flags:  []cli.Flag{},
				Action: Main,
			},
			{
				Name:  "eval",
				Usage: "Eval Server",
				Action: func(c *cli.Context) error {
					return nil
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err)
	}
	os.Exit(0)
}

func Main(c *cli.Context) error {
	// Print welcome message
	fmt.Printf("Starting Kilonova %s\n", version.Version)

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

	// Session Cookie setup
	cookie.Initialize(dataDir)

	// DB Setup
	db, err := db.New(config.C.Database.String())
	if err != nil {
		return err
	}
	log.Println("Connected to DB")

	// Data Manager setup
	manager := datamanager.NewManager(dataDir)

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

	return nil
}
