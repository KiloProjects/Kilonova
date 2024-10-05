package main

import (
	"flag"
	"os"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var (
	confPath = flag.String("config", "./config.toml", "Config path")
	flagPath = flag.String("flags", "./flags.json", "Flag configuration path")
)

func main() {
	flag.Parse()

	config.SetConfigPath(*confPath)
	config.SetConfigV2Path(*flagPath)
	if err := config.Load(); err != nil {
		zap.S().Fatal(err)
	}
	if err := config.LoadConfigV2(); err != nil {
		zap.S().Fatal(err)
	}

	initLogger(config.Common.Debug)

	if err := os.MkdirAll(config.Common.LogDir, 0755); err != nil {
		zap.S().Fatal(err)
	}

	// save the config for formatting
	if err := config.Save(); err != nil {
		zap.S().Fatal(err)
	}

	// save the flags in case any new ones were added
	if err := config.SaveConfigV2(); err != nil {
		zap.S().Fatal(err)
	}

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}

func init() {
	initLogger(true)
}
