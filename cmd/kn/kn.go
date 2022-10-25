package main

import (
	"context"
	"net"
	"net/http"
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

func Kilonova() error {

	// Print welcome message
	zap.S().Infof("Starting Kilonova %s", kilonova.Version)

	if config.Common.Debug {
		zap.S().Warn("Debug mode activated, expect worse performance")
	}

	base, err := sudoapi.InitializeBaseAPI(context.Background())
	if err != nil {
		zap.S().Fatal(err)
	}
	defer base.Close()

	// Initialize components
	grader := grader.NewHandler(context.Background(), base)

	go func() {
		err := grader.Start()
		if err != nil {
			zap.S().Error(err)
		}
	}()

	go func() {
		err := webV1(true, base)
		if err != nil {
			zap.S().Warn("Webserver closed unexpectedly:", err)
		}
	}()

	go func() {
		err := sudoAPI(base)
		if err != nil {
			zap.S().Warn("Sudo API closed unexpectedly:", err)
		}
	}()

	zap.S().Info("Successfully started")
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	<-ctx.Done()

	return nil
}

// blockingly initialize sudo webserverf
func sudoAPI(base *sudoapi.BaseAPI) error {
	return http.ListenAndServe(net.JoinHostPort("localhost", "6001"), sudoapi.NewWebHandler(base).GetHandler())
}

// initialize webserver for public api+web
func webV1(templWeb bool, base *sudoapi.BaseAPI) error {

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

	return http.ListenAndServe(net.JoinHostPort("localhost", strconv.Itoa(config.Common.Port)), r)
}
