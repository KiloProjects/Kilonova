package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova/integrations/otel"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/riandyrn/otelchi"
	slogmulti "github.com/samber/slog-multi"
	"github.com/zitadel/oidc/v3/pkg/op"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/api"
	"github.com/KiloProjects/kilonova/eval/grader"
	"github.com/KiloProjects/kilonova/integrations/maxmind"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Kilonova() error {

	// Setup context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	shutdown, err := otel.SetupOpenTelemetry(ctx)
	if err != nil {
		return err
	}
	defer shutdown(ctx)

	// Print welcome message
	slog.InfoContext(ctx, "Starting Kilonova", slog.String("version", kilonova.Version))

	if config.Common.Debug {
		slog.WarnContext(ctx, "Debug mode activated, expect worse performance")
	}

	maxmind.Initialize(ctx)

	base, err := sudoapi.InitializeBaseAPI(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize BaseAPI", slog.Any("err", err))
		return err
	}
	base.Start(ctx)
	defer base.Close()

	// Initialize components
	grader, err := grader.NewHandler(ctx, base)
	if err != nil {
		slog.ErrorContext(ctx, "Could not initialize grader", slog.Any("err", err))
		return err
	}
	defer grader.Close()

	go func() {
		err := grader.Start()
		if err != nil {
			slog.ErrorContext(ctx, "Could not start grader", slog.Any("err", err))
		}
	}()

	if err := base.ResetWaitingSubmissions(ctx); err != nil {
		slog.WarnContext(ctx, "Couldn't reset initial working submissions", slog.Any("err", err))
	}

	// for graceful setup and shutdown
	server := webV1(true, base)

	go launchProfiler()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "Error initializing web server", slog.Any("err", err))
			cancel()
		}
	}()

	slog.InfoContext(ctx, "Successfully started")

	defer func() {
		slog.InfoContext(ctx, "Shutting Down")
		if err := server.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "Error shutting down", slog.Any("err", err))
		}
	}()

	<-ctx.Done()

	return nil
}

func initLogger(debug, writeFile bool) {

	showUser := slogmulti.NewHandleInlineMiddleware(func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
		if user := util.UserBriefContext(ctx); user != nil {
			record.AddAttrs(slog.Any("user", user))
		}
		if contentUser := util.ContentUserBriefContext(ctx); contentUser != nil {
			record.AddAttrs(slog.Any("contentUser", contentUser))
		}
		return next(ctx, record)
	})

	skipContextCanceled := slogmulti.NewHandleInlineMiddleware(func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
		ok := true
		for attr := range record.Attrs {
			if attr.Key != "err" {
				continue
			}
			if err, isErr := attr.Value.Any().(error); isErr {
				var opErr *net.OpError
				if errors.As(err, &opErr) || errors.Is(err, context.Canceled) || errors.Is(err, kilonova.ErrNotFound) || errors.Is(err, kilonova.ErrNoUpdates) {
					ok = false
					break
				}
			}
		}
		if !ok {
			return nil
		}
		return next(ctx, record)
	})

	handlers := []slog.Handler{
		slogmulti.Pipe(skipContextCanceled).Handler(kilonova.GetSlogHandler(debug, os.Stdout)),
		otelslog.NewHandler("kilonova"),
	}

	if writeFile {
		file := &lumberjack.Logger{
			Filename: path.Join(config.Common.LogDir, "run.log"),
			MaxSize:  80, //MB
			Compress: true,
		}

		loglevel := slog.LevelInfo
		if debug {
			loglevel = slog.LevelDebug
		}

		handlers = append(handlers, slog.NewTextHandler(file, &slog.HandlerOptions{
			AddSource: true,
			Level:     loglevel,
		}))
	}

	slog.SetDefault(slog.New(slogmulti.Pipe(showUser).Handler(slogmulti.Fanout(handlers...))))
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

var (
	listenHost = config.GenFlag[string]("server.listen.host", "localhost", "Host to listen to")
	listenPort = config.GenFlag[int]("server.listen.port", 8070, "Port to listen on")
)

// initialize webserver for public api+web
func webV1(templWeb bool, base *sudoapi.BaseAPI) *http.Server {
	// Initialize router
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{config.Common.HostPrefix}, // TODO: Do better
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	//r.Use(middleware.Timeout(1 * time.Minute))
	/*
		r.Use(middleware.Compress(flate.DefaultCompression))
	*/
	r.Use(middleware.RequestID)
	r.Use(otelchi.Middleware("kilonova-web", otelchi.WithChiRoutes(r)))
	r.Use(op.NewIssuerInterceptor(base.OIDCProvider().IssuerFromRequest).Handler)

	r.Mount("/api", api.New(base).HandlerV1())
	//r.Mount("/api/v2", api.New(base).HandlerV2())
	r.Mount("/assets", api.NewAssets(base).AssetsRouter())

	if templWeb {
		r.Mount("/", web.NewWeb(base).Handler())
	}

	return &http.Server{
		Addr:              net.JoinHostPort(listenHost.Value(), strconv.Itoa(listenPort.Value())),
		Handler:           r,
		ReadHeaderTimeout: 1 * time.Minute,
	}
}
