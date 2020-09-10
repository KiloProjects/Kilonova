package main

import (
	"compress/flate"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/KiloProjects/Kilonova/server"
	"github.com/KiloProjects/Kilonova/web"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// go:generate pkger

var (
	masterDB *gorm.DB
	manager  *datamanager.StorageManager
	db       *kndb.DB
	logg     *log.Logger

	logDir     = flag.String("logDir", "/data/knLogs", "Directory to write logs to")
	debug      = flag.Bool("debug", false, "Debug mode")
	dataDir    = flag.String("data", "/data", "Data directory")
	evalSocket = flag.String("evalSocket", "/tmp/kiloeval.sock", "Path to the eval socket, must be the same as the `socketPath` flag in KiloEval")
)

func main() {
	flag.Parse()

	if !path.IsAbs(*logDir) {
		log.Fatal("logDir not absolute")
	}

	if err := os.MkdirAll(*logDir, 0755); err != nil {
		log.Fatal(err)
	}

	logg = log.New(&lumberjack.Logger{
		Filename: path.Join(*logDir, "access.log"),
	}, "", 0)

	logg.Printf("Starting Kilonova %s\n", common.Version)

	common.SetDataDir(*dataDir)
	common.Initialize()

	var err error
	logg.Println("Trying to connect to DB until it works")
	for {
		dsn := "sslmode=disable user=alexv dbname=kilonova"
		masterDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
	}
	logg.Println("Connected to DB")

	db, err = kndb.New(masterDB, logg)
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate()
	db.DB.Logger = logger.Default.LogMode(logger.Warn)

	manager = datamanager.NewManager(*dataDir)

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
	API := server.NewAPI(ctx, db, manager, logg)
	grader := grader.NewHandler(ctx, db, manager, logg)

	r.Mount("/api", API.GetRouter())
	r.Mount("/", web.NewWeb(manager, db, logg, *debug).GetRouter())

	grader.Start(*evalSocket)

	// for graceful setup and shutdown
	server := &http.Server{Addr: "127.0.0.1:8070", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Println("Successfully started")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	cancel()
	fmt.Println("Shutting Down")
	if err := server.Shutdown(ctx); err != nil {
		fmt.Println(err)
	}
}
