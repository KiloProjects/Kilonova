package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"

	_ "github.com/lib/pq"
)

//go:generate pkger

var (
	confPath    = flag.String("config", "./config.toml", "Config path")
	runMigrator = flag.Bool("runMigrator", false, "Run checker migrator")
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

	if *runMigrator {
		if err := RunMigrator(); err != nil {
			zap.S().Fatal(err)
		}
		return
	}

	if err := Kilonova(); err != nil {
		zap.S().Fatal(err)
	}

	os.Exit(0)
}

func RunMigrator() error {
	// Print welcome message
	zap.S().Infof("Starting Checker Migrator")

	dataDir := config.Common.DataDir
	debug := config.Common.Debug

	if debug {
		zap.S().Warn("Debug mode activated, expect worse performance")
	}

	// Data directory setup
	if !path.IsAbs(dataDir) {
		return &kilonova.Error{Code: kilonova.EINVALID, Message: "dataDir not absolute"}
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("Could not create data dir: %w", err)
	}

	ctx := context.Background()

	// DB Setup
	db, err := db.AppropriateDB(ctx, config.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	zap.S().Info("Connected to DB")

	pbs, err := db.Problems(ctx, kilonova.ProblemFilter{})
	if err != nil {
		return err
	}

	zap.S().Infof("Found %d problems", pbs)

	for _, pb := range pbs {
		if strings.TrimSpace(pb.HelperCode) == "" {
			continue
		}
		att := &kilonova.Attachment{
			ProblemID: pb.ID,
			Visible:   false,
			Private:   true,
			Name:      "grader." + pb.HelperCodeLang,
			Data:      []byte(pb.HelperCode),
		}
		if err := db.CreateAttachment(ctx, att); err != nil {
			return err
		}
	}

	return nil
}
