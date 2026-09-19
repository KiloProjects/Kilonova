package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/infra/maxmind"
	"github.com/KiloProjects/kilonova/infra/otel"
	"github.com/KiloProjects/kilonova/infra/profiler"
	"github.com/KiloProjects/kilonova/net/llm"
	"github.com/KiloProjects/kilonova/sudoapi/flags"

	"github.com/riandyrn/otelchi"
	slogmulti "github.com/samber/slog-multi"
	"github.com/zitadel/oidc/v3/pkg/op"
	"go.opentelemetry.io/contrib/bridges/otelslog"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Kilonova(ctx context.Context) error {

	// Setup context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)

	shutdown, err := otel.SetupOpenTelemetry(ctx, config.Integrations.OtelEnabled)
	if err != nil {
		return err
	}
	defer shutdown(ctx)

	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova", slog.String("version", kilonova.Version))

	if kilonova.DebugMode() {
		slog.WarnContext(ctx, "Debug mode activated, expect worse performance")
	}

	maxmind.Initialize(ctx, config.Integrations.MaxMindDB)

	base, err := initBase(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize BaseAPI", slog.Any("err", err))
		return err
	}
	base.Start(ctx)
	defer base.Close()

	// Initialize components
	graderHandler, err := grader.NewHandler(ctx, base)
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize grader", slog.Any("err", err))
		return err
	}
	defer graderHandler.Close()

	go func() {
		err := graderHandler.Start()
		if err != nil {
			slog.ErrorContext(ctx, "Could not start grader", slog.Any("err", err))
		}
	}()

	if err := base.ResetWaitingSubmissions(ctx); err != nil {
		slog.WarnContext(ctx, "Couldn't reset initial working submissions", slog.Any("err", err))
	}

	// Optional integrations are built here, at the composition root, and
	// handed down; a nil provider means "not configured".
	var llmProvider llm.Provider
	if config.Integrations.OpenAIToken != "" {
		llmProvider = llm.NewOpenAI(config.Integrations.OpenAIToken, config.Integrations.OpenAIModel, config.Integrations.OpenAIVisionModel, cmp.Or(flags.NavbarBranding.Value(), "Kilonova"))
	}

	// for graceful setup and shutdown
	server := webV1(true, base, llmProvider)

	// /healthz sits on an outer mux so it answers without CORS, the OIDC issuer
	// interceptor or session lookup, and keeps answering while we drain.
	var draining atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if draining.Load() {
			http.Error(w, "draining", http.StatusServiceUnavailable)
			return
		}
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := base.PingDB(pingCtx); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "ok")
	})
	mux.Handle("/", server.Handler)
	server.Handler = mux

	go profiler.StartProfiler(ctx, 6080)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Error initializing web server", slog.Any("err", err))
			cancel()
		}
	}()

	slog.InfoContext(ctx, "Successfully started")

	defer func() {
		slog.InfoContext(ctx, "Shutting Down")
		// ctx is the signal context and is already cancelled by the time we get
		// here, so Shutdown needs an independent deadline or it drains nothing.
		draining.Store(true)
		shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutCtx); err != nil {
			slog.ErrorContext(ctx, "Error shutting down", slog.Any("err", err))
		}
	}()

	<-ctx.Done()

	return nil
}

func initLogger(debug, writeFile bool) {

	showUser := slogmulti.NewHandleInlineMiddleware(func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
		if userBrief := user.UserBriefContext(ctx); userBrief != nil {
			record.AddAttrs(slog.Any("user", userBrief))
		}
		if contentUser := user.ContentUserBriefContext(ctx); contentUser != nil {
			record.AddAttrs(slog.Any("contentUser", contentUser))
		}
		return next(ctx, record)
	})

	skipContextCanceled := slogmulti.NewHandleInlineMiddleware(func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
		for attr := range record.Attrs {
			if attr.Key != "err" {
				continue
			}
			if err, isErr := attr.Value.Any().(error); isErr {
				if _, ok := errors.AsType[*net.OpError](err); ok || errors.Is(err, context.Canceled) || errors.Is(err, kilonova.ErrNotFound) || errors.Is(err, kilonova.ErrNoUpdates) {
					return nil
				}
			}
		}
		return next(ctx, record)
	})

	handlers := []slog.Handler{
		slogmulti.Pipe(skipContextCanceled).Handler(kilonova.GetSlogHandler(debug, os.Stdout)),
		otelslog.NewHandler("kilonova"),
	}

	if writeFile && config.LogToFile() {
		file := config.LogWriter("run.log", 80)

		loglevel := slog.LevelInfo
		if debug {
			loglevel = slog.LevelDebug
		}

		handlers = append(handlers, slog.NewJSONHandler(file, &slog.HandlerOptions{
			AddSource: true,
			Level:     loglevel,
		}))
	}

	slog.SetDefault(slog.New(slogmulti.Pipe(showUser).Handler(slogmulti.Fanout(handlers...))))
}

// initialize webserver for public api+web
func webV1(templWeb bool, base *sudoapi.BaseAPI, llmProvider llm.Provider) *http.Server {
	// Initialize router
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			if strings.HasPrefix(r.URL.Path, "/api/v2") {
				return true
			}
			if strings.HasPrefix(r.URL.Path, "/assets") {
				return true
			}
			return strings.ToLower(origin) == kilonova.HostPrefix()
		},
		//AllowedOrigins:   []string{kilonova.HostPrefix()}, // TODO: Do better
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Api-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// r.Use(middleware.ClientIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	//r.Use(middleware.Timeout(1 * time.Minute))
	/*
		r.Use(middleware.Compress(flate.DefaultCompression))
	*/
	r.Use(middleware.RequestID)
	r.Use(otelchi.Middleware("kilonova-web", otelchi.WithChiRoutes(r)))
	r.Use(op.NewIssuerInterceptor(base.OIDCProvider().IssuerFromRequest).Handler)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			addr, _ := base.GetRequestInfo(r)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), user.IPKey, addr)))
		})
	})

	apiSrv := api.New(base, llmProvider)
	r.Mount("/api", apiSrv.HandlerV1())
	r.Mount("/api/v2", apiSrv.HandlerV2())
	r.Mount("/assets", api.NewAssets(base).AssetsRouter())

	if templWeb {
		r.Mount("/", web.NewWeb(base, llmProvider).Handler())
	}

	return &http.Server{
		Addr: config.Server.Listen,

		Handler:           r,
		ReadHeaderTimeout: 1 * time.Minute,
	}
}
