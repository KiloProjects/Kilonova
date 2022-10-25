package main

import (
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	config.SetConfigPath(*confPath)
	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	if err := initLogger(config.Common.LogDir, config.Common.Debug); err != nil {
		log.Fatal(err)
	}

	// save the config for formatting
	if err := config.Save(); err != nil {
		zap.S().Fatal(err)
	}

	if err := eval.Initialize(); err != nil {
		zap.S().Fatal("Could not initialize the box manager:", err)
	}

	go profiler()

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}

func initLogger(logDir string, debug bool) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	var encConf zapcore.EncoderConfig
	if debug {
		encConf = zap.NewDevelopmentEncoderConfig()
	} else {
		encConf = zap.NewDevelopmentEncoderConfig()
		// encConf = zap.NewProductionEncoderConfig()
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

// blockingly start profiler webserver
func profiler() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return http.ListenAndServe(":6080", mux)
}
