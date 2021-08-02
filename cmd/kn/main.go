package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

//go:generate pkger

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

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}
