package main

import (
	"flag"
	"log"
	"os"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
)

func main() {
	flag.Parse()
	config.SetConfigPath(*confPath)
	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	initLogger(config.Common.Debug)

	if err := os.MkdirAll(config.Common.LogDir, 0755); err != nil {
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
